package ds

import (
	"time"
)

type Users struct {
	ID      uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	IsModer bool   `gorm:"not null"`
	Name    string `gorm:"type:varchar(50);unique;not null"`
}

type BorderCrossingFacts struct {
	ID                    uint `gorm:"primaryKey;AUTO_INCREMENT"`
	ClientRefer           int
	Client                Users `gorm:"foreignKey:ClientRefer"`
	ModerRefer            int
	Moder                 Users `gorm:"foreignKey:ModerRefer"`
	StatusRefer           int
	Status                string    `gorm:"type:varchar(20);not null"`
	BorderCrossingPurpose string    `gorm:"type:varchar(50)"`
	DateCreated           time.Time `gorm:"type:timestamp"`
	DateProcessed         time.Time `gorm:"type:timestamp"`
	DateFinished          time.Time `gorm:"type:timestamp"`
}

type Passports struct {
	ID        uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Name      string `gorm:"type:varchar(50)"`
	IsFree    bool
	Seria     string `gorm:"type:varchar(20)"`
	Issue     string `gorm:"type:varchar(50)"`
	Code      string `gorm:"type:varchar(20)"`
	Gender    string `gorm:"type:varchar(20)"`
	Birthdate string `gorm:"type:varchar(20)"`
	BDplace   string `gorm:"type:varchar(50)"`
	Image     string `gorm:"type:bytea"`
}

type BorderCrossingPassports struct {
	ID            uint `gorm:"primaryKey;AUTO_INCREMENT"`
	RequestRefer  int
	Request       BorderCrossingFacts `gorm:"foreignKey:RequestRefer"`
	PassportRefer int
	Passport      Passports `gorm:"foreignKey:PassportRefer"`
}
