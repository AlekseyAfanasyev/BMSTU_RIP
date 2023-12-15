package repository

import (
	"BMSTU_RIP/internal/app/ds"
	mClient "BMSTU_RIP/internal/app/minio"
	"BMSTU_RIP/internal/app/role"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"os"
	_ "os"
	"slices"
	"strings"
	"time"
)

// пакет отвечающий за обращения к хранилищам данных(БД)
type Repository struct {
	db *gorm.DB
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
		return nil, err
	}

	// Check the connection
	if sqlDB, err := db.DB(); err != nil {
		log.Printf("Failed to initialize the database connection: %v", err)
		return nil, err
	} else {
		if err := sqlDB.Ping(); err != nil {
			log.Printf("Failed to ping the database: %v", err)
			return nil, err
		}
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) GetPassportByID(id int) (*ds.Passports, error) {
	Passport := &ds.Passports{}

	err := r.db.First(Passport, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return Passport, nil
}

func (r *Repository) DeletePassport(passport_name string) error {
	return r.db.Delete(&ds.Passports{}, "name = ?", passport_name).Error
}

// ---------------------------------------------------------------------------------
// --------------------------------- PASSPORTS METHODS ---------------------------------
// ---------------------------------------------------------------------------------

//func (r *Repository) GetAllPassports() ([]ds.Passports, error) {
//	passports := []ds.Passports{}
//
//	err := r.db.Order("id").Find(&passports).Error
//
//	if err != nil {
//		return nil, err
//	}
//
//	return passports, nil
//}

func (r *Repository) GetAllPassports(passportName string) ([]ds.Passports, error) {
	passports := []ds.Passports{}
	if passportName == "" {
		err := r.db.Where("is_free = ?", true).
			Order("id").Find(&passports).Error

		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.Where("is_free = ?", true).Where("name ILIKE ?", "%"+passportName+"%").
			Order("id").Find(&passports).Error

		if err != nil {
			return nil, err
		}
	}

	return passports, nil
}

func (r *Repository) SearchPassports(passportName string) ([]ds.Passports, error) {
	passports := []ds.Passports{}
	passportName = "%" + passportName + "%"

	err := r.db.Where("name ILIKE ?", passportName).Order("id").Find(&passports).Error
	if err != nil {
		return nil, err
	}

	return passports, nil
}

func (r *Repository) FilterPassports(passports []ds.Passports) []ds.Passports {
	var new_passports = []ds.Passports{}

	for i := range passports {
		new_passports = append(new_passports, passports[i])
	}

	return new_passports
}

func (r *Repository) GetPassportByName(name string) (*ds.Passports, error) {
	passport := &ds.Passports{}

	err := r.db.First(passport, "name = ?", name).Error
	if err != nil {
		return nil, err
	}

	return passport, nil
}

func (r *Repository) ChangeAvailability(passportName string) error {
	query := "UPDATE passports SET is_free = NOT is_free WHERE Name = $1"

	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}

	_, err = sqlDB.Exec(query, passportName)

	return err
}

//func (r *Repository) AddPassport(Name, Seria, Issue, Code, Gender, Birthdate, BDplace, Image string) error {
//	NewPassport := &ds.Passports{
//		ID:        uint(len([]ds.Passports{})),
//		Name:      Name,
//		IsFree:    false,
//		Seria:     Seria,
//		Issue:     Issue,
//		Code:      Code,
//		Gender:    Gender,
//		Birthdate: Birthdate,
//		BDplace:   BDplace,
//		Image:     Image,
//	}
//
//	return r.db.Create(NewPassport).Error
//}

func (r *Repository) AddPassport(passport *ds.Passports, imagePath string) error {

	imageURL := "http://127.0.0.1:9000/pc-bucket/DEFAULT.jpg"
	log.Println(imagePath)
	if imagePath != "" {
		var err error
		imageURL, err = r.uploadImageToMinio(imagePath)
		if err != nil {
			return err
		}
	}
	var cntOrbits int64
	err := r.db.Model(&ds.Passports{}).Count(&cntOrbits).Error
	if err != nil {
		log.Println(err)
		return err
	}

	passport.Image = imageURL

	passport.ID = uint(cntOrbits) + 1

	return r.db.Create(passport).Error
}

func (r *Repository) EditPassport(passportID uint, editingPassport ds.Passports) error {
	// Проверяем, изменился ли URL изображения
	originalPassport, err := r.GetPassportByID(int(passportID))
	if err != nil {
		return err
	}

	log.Println("OLD IMAGE: ", originalPassport.Image)
	log.Println("NEW IMAGE: ", editingPassport.Image)

	if editingPassport.Image != originalPassport.Image && editingPassport.Image != "" {
		log.Println("REPLACING IMAGE")

		if originalPassport.Image != "http://127.0.0.1:9000/pc-bucket/DEFAULT.jpg" {
			err := r.deleteImageFromMinio(originalPassport.Image)
			if err != nil {
				return err
			}
		}
		imageURL, err := r.uploadImageToMinio(editingPassport.Image)
		if err != nil {
			return err
		}

		editingPassport.Image = imageURL

		log.Println("IMAGE REPLACED")
	}

	return r.db.Model(&ds.Passports{}).Where("id = ?", passportID).Updates(editingPassport).Error
}

func (r *Repository) uploadImageToMinio(imagePath string) (string, error) {
	// Получаем клиента Minio из настроек
	minioClient := mClient.NewMinioClient()

	// Загрузка изображения в Minio
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// uuid - уникальное имя; parts - имя файла
	//objectName := uuid.New().String() + ".jpg"
	parts := strings.Split(imagePath, "/")
	objectName := parts[len(parts)-1]

	_, err = minioClient.PutObject(context.Background(), "pc-bucket", objectName, file, -1, minio.PutObjectOptions{})
	if err != nil {
		return "!!!", err
	}

	// Возврат URL изображения в Minio
	log.Println("Error MINIO")
	return fmt.Sprintf("http://%s/%s/%s", minioClient.EndpointURL().Host, "pc-bucket", objectName), nil
}

func (r *Repository) deleteImageFromMinio(imageURL string) error {
	minioClient := mClient.NewMinioClient()

	objectName := extractObjectNameFromURL(imageURL)

	return minioClient.RemoveObject(context.Background(), "pc-bucket", objectName, minio.RemoveObjectOptions{})
}

func extractObjectNameFromURL(imageURL string) string {
	parts := strings.Split(imageURL, "/")
	log.Println("\n\nIMG:   ", parts[len(parts)-1])
	return parts[len(parts)-1]
}

//func (r *Repository) EditPassport(ID uint, passport ds.Passports) error {
//	log.Println("FUNC ORBIT: ", passport, "    ", ID)
//	return r.db.Model(&ds.Passports{}).Where("id = ?", ID).Updates(passport).Error
//}

func (r *Repository) GetPassportBySeria(seria string) (*ds.Passports, error) {
	passport := &ds.Passports{}

	err := r.db.First(passport, "seria = ?", seria).Error
	if err != nil {
		return nil, err
	}

	return passport, nil
}

func (r *Repository) DeleteBorderCrossingFactRequest(req_id uint) error {
	if r.db.Where("id = ?", req_id).First(&ds.BorderCrossingFacts{}).Error != nil {

		return r.db.Where("id = ?", req_id).First(&ds.BorderCrossingFacts{}).Error
	}
	return r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", req_id).Update("status", "Удалена").Error
}

// =================================================================================
// ---------------------------------------------------------------------------------
// --------------------------- BORDER_CROSSING_FACTS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) CreateBorderCrossingRequest(requestBody ds.CreateBorderCrossingFactBody, userUUID uuid.UUID) (int, error) {
	var passport_ids []int
	var passport_names []string
	for _, passportName := range requestBody.Passports {
		passport, err := r.GetPassportByName(passportName)
		if err != nil {
			return 0, err
		}
		passport_ids = append(passport_ids, int(passport.ID))
		passport_names = append(passport_names, passport.Name)
	}

	request, err := r.GetCurrentRequest(userUUID)
	if err != nil {
		log.Println(" --- NEW REQUEST --- ", userUUID)

		// Назначение модератора
		moders := []ds.UserUID{}
		err = r.db.Where("role = ?", 2).Find(&moders).Error
		if err != nil {
			return 0, err
		}
		n := rand.Int() % len(moders)
		moder_refer := moders[n].UUID
		log.Println("moder: ", moder_refer)

		// Поля типа Users, связанные с передаваемыми значениями из функции
		client := ds.UserUID{UUID: userUUID}
		moder := ds.UserUID{UUID: moder_refer}

		request = &ds.BorderCrossingFacts{
			ID:            uint(len([]ds.BorderCrossingFacts{})),
			ClientRefer:   userUUID,
			Client:        client,
			ModerRefer:    moder_refer,
			Moder:         moder,
			Status:        "Черновик",
			DateCreated:   time.Now(),
			DateProcessed: nil,
			DateFinished:  nil,
		}

		err := r.db.Create(request).Error
		if err != nil {
			return 0, err
		}
	}

	err = r.SetRequestPassports(int(request.ID), passport_names)
	if err != nil {
		return 0, err
	}

	// Uncomment the following block if needed
	// for _, orbit_id := range orbit_ids {
	// 	transfer_to_orbit := ds.TransferToOrbit{}
	// 	transfer_to_orbit.RequestRefer = request.ID
	// 	transfer_to_orbit.OrbitRefer = uint(orbit_id)
	// 	err = r.CreateTransferToOrbit(transfer_to_orbit)
	//
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// }

	// Return request ID along with nil error
	return int(request.ID), nil
}

func (r *Repository) SetRequestPassports(requestID int, passports []string) error {
	var passport_ids []int
	log.Println(requestID, " - ", passports)
	for _, passport_name := range passports {
		passport, err := r.GetPassportByName(passport_name)
		log.Println("passport: ", passport)
		if err != nil {
			return err
		}

		for _, ele := range passport_ids {
			if ele == int(passport.ID) {
				log.Println("!!!")
				continue
			}
		}
		log.Println("BEFORE :", passport_ids)
		passport_ids = append(passport_ids, int(passport.ID))
		log.Println("AFTER :", passport_ids)
	}

	var existing_links []ds.BorderCrossingPassports
	err := r.db.Model(&ds.BorderCrossingPassports{}).Where("request_refer = ?", requestID).Find(&existing_links).Error
	if err != nil {
		return err
	}
	log.Println("LINKS: ", existing_links)
	for _, link := range existing_links {
		passportFound := false
		passportIndex := -1
		for index, ele := range passport_ids {
			if ele == int(link.PassportRefer) {
				passportFound = true
				passportIndex = index
				break
			}
		}
		log.Println("ORB F: ", passportFound)
		if passportFound {
			log.Println("APPEND: ")
			passport_ids = append(passport_ids[:passportIndex], passport_ids[passportIndex+1:]...)
		} else {
			log.Println("DELETE: ")
			err := r.db.Model(&ds.BorderCrossingPassports{}).Delete(&link).Error
			if err != nil {
				return err
			}
		}
	}

	for _, orbit_id := range passport_ids {
		newLink := ds.BorderCrossingPassports{
			RequestRefer:  uint(requestID),
			PassportRefer: uint(orbit_id),
		}
		log.Println("NEW LINK", newLink.PassportRefer, " --- ", newLink.RequestRefer)
		err := r.db.Model(&ds.BorderCrossingPassports{}).Create(&newLink).Error
		if err != nil {
			return nil
		}
	}

	return nil
}

//func (r *Repository) CreateBorderCrossingRequest(client_id uuid.UUID) (*ds.BorderCrossingFacts, error) {
//	request, err := r.GetCurrentRequest(client_id)
//	if err != nil {
//		log.Println("NO OPENED REQUESTS => CREATING NEW ONE")
//
//		//назначение модератора
//		moders := []ds.UserUID{}
//		err = r.db.Where("role = ?", 2).Find(&moders).Error
//		if err != nil {
//			return nil, err
//		}
//		n := rand.Int() % len(moders)
//		moder_refer := moders[n].UUID
//		log.Println("moder: ", moder_refer)
//
//		//поля типа Users, связанные с передавыемыми значениями из функции
//		client := ds.UserUID{UUID: client_id}
//		moder := ds.UserUID{UUID: moder_refer}
//
//		NewTransferRequest := &ds.BorderCrossingFacts{
//			ID:            uint(len([]ds.BorderCrossingFacts{})),
//			ClientRefer:   client_id,
//			Client:        client,
//			ModerRefer:    moder_refer,
//			Moder:         moder,
//			Status:        "Черновик",
//			DateCreated:   time.Now(),
//			DateProcessed: nil,
//			DateFinished:  nil,
//		}
//		return NewTransferRequest, r.db.Create(NewTransferRequest).Error
//	}
//	return request, nil
//}

func (r *Repository) GetCurrentRequest(client_refer uuid.UUID) (*ds.BorderCrossingFacts, error) {
	request := &ds.BorderCrossingFacts{}
	err := r.db.Where("status = ?", "Черновик").First(request, "client_refer = ?", client_refer).Error
	//если реквеста нет => err = record not found
	if err != nil {
		//request = nil, err = not found
		return nil, err
	}
	//если реквест есть => request = record, err = nil
	return request, nil
}

func (r *Repository) GetAllRequests(userRole any, dateStart, dateFin, status string) ([]ds.BorderCrossingFacts, error) {

	requests := []ds.BorderCrossingFacts{}
	qry := r.db

	if dateStart != "" && dateFin != "" {
		qry = qry.Where("date_processed BETWEEN ? AND ?", dateStart, dateFin)
	} else if dateStart != "" {
		qry = qry.Where("date_processed >= ?", dateStart)
	} else if dateFin != "" {
		qry = qry.Where("date_processed <= ?", dateFin)
	}
	if status != "" {
		qry = qry.Where("status = ?", status)
	}

	if userRole == role.Moderator {
		qry = qry.Where("status = ?", ds.ReqStatuses[1])
	} else {
		qry = qry.Where("status IN ?", ds.ReqStatuses)
	}

	err := qry.
		Preload("Client").Preload("Moder"). //данные для полей типа User: {ID, Name, IsModer)
		Order("id").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

//func (r *Repository) GetRequestByID(id uint) (*ds.BorderCrossingFacts, error) {
//	request := &ds.BorderCrossingFacts{}
//
//	err := r.db.First(request, "id = ?", id).Error
//	if err != nil {
//		return nil, err
//	}
//
//	return request, nil
//}

func (r *Repository) ChangeRequestStatus(id uint, status string) error {
	if slices.Contains(ds.ReqStatuses[2:5], status) {
		err := r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", id).Update("date_finished", time.Now()).Error
		if err != nil {
			return err
		}
	}

	if status == ds.ReqStatuses[1] {
		err := r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", id).Update("date_processed", time.Now()).Error
		if err != nil {
			return err
		}
	}

	err := r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса: %w", err)
	}
	if status == ds.ReqStatuses[2] || status == ds.ReqStatuses[3] {
		err = r.DeleteBorderCrossingPassportsEvery(id)
	}

	return nil
}

func (r *Repository) GetRequestByID(id uint, userUUID uuid.UUID, userRole any) (*ds.BorderCrossingFacts, error) {
	request := &ds.BorderCrossingFacts{}
	qry := r.db

	if userRole == role.Client {
		qry = qry.Where("client_refer = ?", userUUID)
	} else {
		qry = qry.Where("moder_refer = ?", userUUID)
	}

	err := qry.Preload("Client").Preload("Moder").First(request, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return request, nil
}

// =================================================================================
// ---------------------------------------------------------------------------------
// ------------------------- BORDER_CROSSING_PASSPORTS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) AddRequestToBorderCrossingPassports(passport_refer, request_refer int) error {
	passport := ds.Passports{ID: uint(passport_refer)}
	request := ds.BorderCrossingFacts{ID: uint(request_refer)}

	NewMtM := &ds.BorderCrossingPassports{
		ID:            uint(len([]ds.BorderCrossingPassports{})),
		Passport:      passport,
		PassportRefer: uint(passport_refer),
		Request:       request,
		RequestRefer:  uint(request_refer),
	}
	return r.db.Create(NewMtM).Error
}

func (r *Repository) DeleteBorderCrossingPassportsEvery(transfer_id uint) error {
	if r.db.Where("request_refer = ?", transfer_id).First(&ds.BorderCrossingPassports{}).Error != nil {
		return r.db.Where("request_refer = ?", transfer_id).First(&ds.BorderCrossingPassports{}).Error
	}
	return r.db.Where("request_refer = ?", transfer_id).Delete(&ds.BorderCrossingPassports{}).Error
}

func (r *Repository) GetPassportsFromRequest(id int) ([]ds.Passports, error) {
	passport_to_request := []ds.BorderCrossingPassports{}

	err := r.db.Model(&ds.BorderCrossingPassports{}).Where("request_refer = ?", id).Find(&passport_to_request).Error
	if err != nil {
		return []ds.Passports{}, err
	}

	var passports []ds.Passports
	for _, passport_to_requests := range passport_to_request {
		orbit, err := r.GetPassportByID(int(passport_to_requests.PassportRefer))
		if err != nil {
			return []ds.Passports{}, err
		}
		for _, ele := range passports {
			if ele == *orbit {
				continue
			}
		}
		passports = append(passports, *orbit)
	}

	return passports, nil

}

// =================================================================================
// ---------------------------------------------------------------------------------
// ------------------------- USERS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) GetUserByName(name string) (*ds.UserUID, error) {
	user := &ds.UserUID{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) Register(user *ds.UserUID) error {
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}

	return r.db.Create(user).Error
}

func (r *Repository) GenerateHashString(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
