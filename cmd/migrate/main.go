package main

import (
	"BMSTU_RIP/internal/app/ds"
	"BMSTU_RIP/internal/app/dsn"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema

	err = db.AutoMigrate(&ds.Users{})
	if err != nil {
		panic("cant migrate db")
	}

	err = db.AutoMigrate(&ds.BorderCrossingFacts{})
	if err != nil {
		panic("cant migrate db")
	}

	err = db.AutoMigrate(&ds.Passports{})
	if err != nil {
		panic("cant migrate db")
	}

	err = db.AutoMigrate(&ds.BorderCrossingPassports{})
	if err != nil {
		panic("cant migrate db")
	}

}
