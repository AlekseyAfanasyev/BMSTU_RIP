package repository

import (
	"BMSTU_RIP/internal/app/ds"
	mClient "BMSTU_RIP/internal/app/minio"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"os"
	_ "os"
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
		err := r.deleteImageFromMinio(originalPassport.Image)
		if err != nil {
			return err
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

// =================================================================================
// ---------------------------------------------------------------------------------
// --------------------------- BORDER_CROSSING_FACTS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) CreateBorderCrossingRequest(client_refer int) (*ds.BorderCrossingFacts, error) {
	//проверка есть ли открытая заявка у клиента
	request, err := r.GetCurrentRequest(client_refer)
	if err != nil {
		log.Println("NO OPENED REQUESTS => CREATING NEW ONE")

		//назначение модератора
		users := []ds.Users{}
		err = r.db.Where("is_moder = ?", true).Find(&users).Error
		if err != nil {
			return nil, err
		}
		n := rand.Int() % len(users)
		moder_refer := users[n].ID
		log.Println("moder: ", moder_refer)

		//поля типа Users, связанные с передавыемыми значениями из функции
		client := ds.Users{ID: uint(client_refer)}
		moder := ds.Users{ID: moder_refer}

		NewBorderCrossingRequest := &ds.BorderCrossingFacts{
			ID:            uint(len([]ds.BorderCrossingFacts{})),
			ClientRefer:   client_refer,
			Client:        client,
			ModerRefer:    int(moder_refer),
			Moder:         moder,
			Status:        "Черновик",
			DateCreated:   time.Now(),
			DateProcessed: nil,
			DateFinished:  nil,
		}
		return NewBorderCrossingRequest, r.db.Create(NewBorderCrossingRequest).Error
	}
	return request, nil
}

func (r *Repository) GetCurrentRequest(client_refer int) (*ds.BorderCrossingFacts, error) {
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

func (r *Repository) GetAllRequests() ([]ds.BorderCrossingFacts, error) {

	requests := []ds.BorderCrossingFacts{}

	err := r.db.
		Preload("Client").Preload("Moder"). //данные для полей типа User: {ID, Name, IsModer)
		Order("id").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *Repository) GetRequestByID(id int) (*ds.BorderCrossingFacts, error) {
	request := &ds.BorderCrossingFacts{}

	err := r.db.First(request, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (r *Repository) ChangeRequestStatus(id int, status string) error {
	return r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) DeleteBorderCrossingFactRequest(req_id uint) error {
	if r.db.Where("id = ?", req_id).First(&ds.BorderCrossingFacts{}).Error != nil {

		return r.db.Where("id = ?", req_id).First(&ds.BorderCrossingFacts{}).Error
	}
	return r.db.Model(&ds.BorderCrossingFacts{}).Where("id = ?", req_id).Update("status", "Удалена").Error
}

// =================================================================================
// ---------------------------------------------------------------------------------
// ------------------------- BORDER_CROSSING_PASSPORTS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) AddRequestToBorderCrossingPassports(passport_refer, request_refer int) error {
	passport := ds.Passports{ID: uint(passport_refer)}
	request := ds.BorderCrossingFacts{ID: uint(request_refer)}

	NewMtM := &ds.BorderCrossingPassports{
		ID:            int(uint(len([]ds.BorderCrossingPassports{}))),
		Passport:      passport,
		PassportRefer: passport_refer,
		Request:       request,
		RequestRefer:  request_refer,
	}
	return r.db.Create(NewMtM).Error
}

func (r *Repository) DeleteBorderCrossingPassportsEvery(transfer_id uint) error {
	if r.db.Where("request_refer = ?", transfer_id).First(&ds.BorderCrossingPassports{}).Error != nil {
		return r.db.Where("request_refer = ?", transfer_id).First(&ds.BorderCrossingPassports{}).Error
	}
	return r.db.Where("request_refer = ?", transfer_id).Delete(&ds.BorderCrossingPassports{}).Error
}

// =================================================================================
// ---------------------------------------------------------------------------------
// ------------------------- USERS METHODS ---------------------------
// ---------------------------------------------------------------------------------

func (r *Repository) GetUserRole(name string) (*bool, error) {
	user := &ds.Users{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return nil, err
	}

	return user.IsModer, nil
}

func (r *Repository) GetUserByName(name string) (*ds.Users, error) {
	user := &ds.Users{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}
