package repository

import (
	"BMSTU_RIP/internal/app/ds"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
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

func (r *Repository) SearchPassports(passportName string) ([]ds.Passports, error) {
	passports := []ds.Passports{}
	passportName = "%" + passportName + "%"

	err := r.db.Where("name ILIKE ?", passportName).Order("id").Find(&passports).Error
	if err != nil {
		return nil, err
	}

	return passports, nil
}

func (r *Repository) DeletePassport(passport_name string) error {
	return r.db.Delete(&ds.Passports{}, "name = ?", passport_name).Error
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

func (r *Repository) GetAllPassports() ([]ds.Passports, error) {
	passports := []ds.Passports{}

	err := r.db.Order("id").Find(&passports).Error

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
