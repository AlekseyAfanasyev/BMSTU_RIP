package app

import (
	"BMSTU_RIP/internal/app/dsn"
	"BMSTU_RIP/internal/app/repository"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Application struct {
	repo repository.Repository
	r    *gin.Engine
}

func New() Application {
	app := Application{}

	repo, _ := repository.New(dsn.FromEnv())

	app.repo = *repo

	return app

}

func (a *Application) StartServer() {
	log.Println("Server started")

	a.r = gin.Default()

	a.r.LoadHTMLGlob("templates/*.html")
	a.r.Static("/css", "./templates")

	a.r.GET("/", a.loadGeneral)
	a.r.GET("/:passport_name", a.loadDetail)

	a.r.POST("/delete_passport/:passport_name", func(c *gin.Context) {
		passportName := c.Param("passport_name")

		err := a.repo.ChangeAvailability(passportName)

		if err != nil {
			c.Error(err)
			return
		}

		c.Redirect(http.StatusFound, "/")
	})

	a.r.Run(":8000")

	log.Println("Server is down")
}

func (a *Application) loadGeneral(c *gin.Context) {
	passportName := c.Query("passport_name")

	if passportName == "" {
		log.Println("ALL ORBITS 1")

		allPassports, err := a.repo.GetAllPassports()

		if err != nil {
			c.Error(err)
		}

		c.HTML(http.StatusOK, "passports.html", gin.H{
			"passports": a.repo.FilterPassports(allPassports),
		})
	} else {
		log.Println("!!! SEARCHING PASSPORTS !!!")

		foundPassports, err := a.repo.SearchPassports(passportName)
		if err != nil {
			c.Error(err)
			return
		}

		c.HTML(http.StatusOK, "passports.html", gin.H{
			"passports":    a.repo.FilterPassports(foundPassports),
			"passportName": passportName,
		})
	}
}

func (a *Application) loadDetail(c *gin.Context) {
	passport_name := c.Param("passport_name")

	if passport_name == "favicon.ico" {
		return
	}

	passport, err := a.repo.GetPassportByName(passport_name)

	if err != nil {
		c.Error(err)
		return
	}

	c.HTML(http.StatusOK, "passport.html", gin.H{
		"Name":      passport.Name,
		"Image":     passport.Image,
		"Seria":     passport.Seria,
		"IsFree":    passport.IsFree,
		"Issue":     passport.Issue,
		"Code":      passport.Code,
		"Gender":    passport.Gender,
		"Birthdate": passport.Birthdate,
		"BDplace":   passport.BDplace,
	})

}
