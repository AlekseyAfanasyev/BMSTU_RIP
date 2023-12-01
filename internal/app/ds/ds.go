package ds

import (
	"BMSTU_RIP/internal/app/role"
	"github.com/google/uuid"
	"time"
)

type UserUID struct {
	UUID uuid.UUID `gorm:"type:uuid"`
	Name string    `json:"name"`
	Role role.Role `sql:"type:string;"`
	Pass string
}

type Users struct {
	ID       uint `gorm:"primaryKey;AUTO_INCREMENT"`
	IsModer  *bool
	Name     string `gorm:"type:varchar(50);unique;not null"`
	Password string `gorm:"type:varchar(50);not null"`
}

type BorderCrossingFacts struct {
	ID                    uint `gorm:"primaryKey;AUTO_INCREMENT"`
	ClientRefer           int
	Client                Users `gorm:"foreignKey:ClientRefer"`
	ModerRefer            int
	Moder                 Users      `gorm:"foreignKey:ModerRefer"`
	Status                string     `gorm:"type:varchar(20);not null"`
	BorderCrossingPurpose string     `gorm:"type:varchar(50)"`
	DateCreated           time.Time  `gorm:"type:timestamp"`
	DateProcessed         *time.Time `gorm:"type:timestamp"`
	DateFinished          *time.Time `gorm:"type:timestamp"`
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
	Image     string `gorm:"column:image"`
}

type BorderCrossingPassports struct {
	ID            int `gorm:"primaryKey;AUTO_INCREMENT"`
	RequestRefer  int
	Request       BorderCrossingFacts `gorm:"foreignKey:RequestRefer"`
	PassportRefer int
	Passport      Passports `gorm:"foreignKey:PassportRefer"`
}

type ChangeBorderCrossingFactStatus struct {
	BorderCrossingFactID uint
	Status               string
	UserName             string
}

var ReqStatuses = []string{"Черновик", "Удалена", "Отклонена", "Оказана", "На рассмотрении"}
