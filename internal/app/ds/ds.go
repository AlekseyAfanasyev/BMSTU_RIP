package ds

import (
	"BMSTU_RIP/internal/app/role"
	"github.com/google/uuid"
	"time"
)

type UserUID struct {
	UUID uuid.UUID `gorm:"type:uuid;unique"`
	Name string    `json:"Name"`
	Role role.Role `sql:"type:string;"`
	Pass string
}

type BorderCrossingFacts struct {
	ID            uint       `gorm:"primaryKey;AUTO_INCREMENT"`
	ClientRefer   uuid.UUID  `gorm:"type:uuid"`
	Client        UserUID    `gorm:"foreignKey:ClientRefer;references:UUID"`
	ModerRefer    uuid.UUID  `gorm:"type:uuid"`
	Moder         UserUID    `gorm:"foreignKey:ModerRefer;references:UUID"`
	Status        string     `gorm:"type:varchar(20); not null"`
	DateCreated   time.Time  `gorm:"type:timestamp"` //timestamp without time zone
	DateProcessed *time.Time `gorm:"type:timestamp"`
	DateFinished  *time.Time `gorm:"type:timestamp"`
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
	ID            uint `gorm:"primaryKey;AUTO_INCREMENT"`
	RequestRefer  uint
	Request       BorderCrossingFacts `gorm:"foreignKey:RequestRefer"`
	PassportRefer uint
	Passport      Passports `gorm:"foreignKey:PassportRefer"`
	IsBiometry    *bool     `gorm:"type:bool"`
}

type ChangeBorderCrossingFactStatus struct {
	BorderCrossingFactID uint   `json:"reqID"`
	Status               string `json:"status"`
}

type CreateBorderCrossingFactBody struct {
	Passports []string
}

type SetRequestPassportsRequestBody struct {
	RequestID int
	Passports []string
}

var ReqStatuses = []string{
	"Черновик",
	"На рассмотрении",
	"Удалена",
	"Отклонена",
	"Оказана"}

type AsyncBody struct {
	RequestID  int  `json:"request_refer"`
	PassportID int  `json:"passport_refer"`
	Fact       bool `json:"is_biometry"`
}

type TestReqBody struct {
	Passport string
}

type TestDelBody struct {
	Passport string
	Req      string
}

type DelTransfReqRequestBody struct {
	Req int
}
