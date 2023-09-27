package ds

import (
	"gorm.io/datatypes"
)

type Users struct {
	ID      uint `gorm:"primaryKey"`
	IsModer bool
	Name    string
}

type BorderCrossingFacts struct {
	ID                    uint `gorm:"primaryKey"`
	ClientRefer           int
	Client                Users `gorm:"foreignKey:ClientRefer"`
	ModerRefer            int
	Moder                 Users `gorm:"foreignKey:ModerRefer"`
	StatusRefer           int
	Status                RequestStatus `gorm:"foreignKey:StatusRefer"`
	BorderCrossingPurpose string
	DateCreated           datatypes.Date
	DateProcessed         datatypes.Date
	DateFinished          datatypes.Date
}

type RequestStatus struct {
	ID     uint `gorm:"primaryKey"`
	Status string
}

type Passports struct {
	ID        uint `gorm:"primaryKey"`
	Name      string
	IsFree    bool
	Seria     string
	Issue     string
	Code      string
	Gender    string
	Birthdate string
	BDplace   string
	Image     string `gorm:"type:bytea"`
}

type BorderCrossingPassports struct {
	ID            uint `gorm:"primaryKey"`
	RequestRefer  int
	Request       BorderCrossingFacts `gorm:"foreignKey:RequestRefer"`
	PassportRefer int
	Passport      Passports `gorm:"foreignKey:PassportRefer"`
}
