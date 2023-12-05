package app

import (
	"BMSTU_RIP/internal/app/ds"
	"BMSTU_RIP/internal/app/dsn"
	"BMSTU_RIP/internal/app/repository"
	"log"
	"net/http"
	"slices"
	"strconv"

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

	a.r.GET("passports", a.getAllPassports)
	a.r.GET("passports/:passport_name", a.getDetailedPassport)
	a.r.GET("border_crossing_facts", a.getAllRequests)
	a.r.GET("border_crossing_facts/id/:req_id", a.getDetailedRequest)

	a.r.PUT("passports/:passport_seria/edit", a.editPassport)
	a.r.PUT("border_crossing_fact/:req_id/moder_change_status", a.moderChangeTransferRequestStatus)

	a.r.POST("passports/add_new_passport", a.newPassport)
	a.r.POST("border_crossing_fp/:req_id/delete", a.deleteBorderCrossingFactRequest)
	a.r.POST("/:passport_seria/add", a.addPassportToRequest)
	a.r.POST("change_passport_availibility/:passport_name", a.changeAvailability)
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

//func (a *Application) getAllPassports(c *gin.Context) {
//	passportName := c.Query("passport_name")
//
//	if passportName == "" {
//		log.Println("ALL Passports 1")
//
//		allPassports, err := a.repo.GetAllPassports()
//
//		if err != nil {
//			c.Error(err)
//		}
//
//		//для лаб3 нужен хтмл
//		//c.HTML(http.StatusOK, "passports.html", gin.H{
//		//	"passports": a.repo.FilterPassports(allPassports),
//		//})
//
//		//для лаб4 нужен жсон
//		c.JSON(http.StatusOK, gin.H{
//			"passports": a.repo.FilterPassports(allPassports),
//		})
//	} else {
//		log.Println("!!! SEARCHING Passports !!!")
//
//		foundPassports, err := a.repo.SearchPassports(passportName)
//		if err != nil {
//			c.Error(err)
//			return
//		}
//		log.Println("found: ", len(foundPassports))
//
//		//для лаб3 нужен хтмл
//		//c.HTML(http.StatusOK, "passports.html", gin.H{
//		//	"passports":    a.repo.FilterPassports(foundPassports),
//		//	"passportName": passportName,
//		//})
//
//		//для лаб4 нужен жсон
//		c.JSON(http.StatusOK, gin.H{
//			"passports":    a.repo.FilterPassports(foundPassports),
//			"passportName": passportName,
//		})
//	}
//}

func (a *Application) getAllPassports(c *gin.Context) {
	passportName := c.Query("passport_name")

	allPassports, err := a.repo.GetAllPassports(passportName)

	if err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusFound, allPassports)
}

func (a *Application) getDetailedPassport(c *gin.Context) {
	passport_name := c.Param("passport_name")

	if passport_name == "favicon.ico" {
		return
	}

	passport, err := a.repo.GetPassportByName(passport_name)

	if err != nil {
		c.Error(err)
		return
	}

	//c.HTML(http.StatusOK, "passport.html", gin.H{
	//	"Name":      passport.Name,
	//	"Image":     passport.Image,
	//	"Seria":     passport.Seria,
	//	"IsFree":    passport.IsFree,
	//	"Issue":     passport.Issue,
	//	"Code":      passport.Code,
	//	"Gender":    passport.Gender,
	//	"Birthdate": passport.Birthdate,
	//	"BDplace":   passport.BDplace,
	//})

	c.JSON(http.StatusOK, gin.H{
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

func (a *Application) changeAvailability(c *gin.Context) {
	passportName := c.Param("passport_name")
	log.Println("passportName : ", passportName)

	// Call the modified ChangeAvailability method
	err := a.repo.ChangeAvailability(passportName)
	log.Println("err : ", err)

	if err != nil {
		c.Error(err)
		return
	}

	c.Redirect(http.StatusFound, "/passports")
}

func (a *Application) newPassport(c *gin.Context) {
	var requestBody ds.Passports

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
	}

	err := a.repo.AddPassport(&requestBody, requestBody.Image)
	log.Println(requestBody.Name, " is added")

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ID":        requestBody.ID,
		"Name":      requestBody.Name,
		"Seria":     requestBody.Seria,
		"Issue":     requestBody.Issue,
		"Code":      requestBody.Code,
		"Gender":    requestBody.Gender,
		"Birthdate": requestBody.Birthdate,
		"BDplace":   requestBody.BDplace,
		"Image":     requestBody.Image,
	})
}

func (a *Application) editPassport(c *gin.Context) {
	passport_seria := c.Param("passport_seria")
	passport, err := a.repo.GetPassportBySeria(passport_seria)

	var editingPassport ds.Passports

	if err := c.BindJSON(&editingPassport); err != nil {
		c.Error(err)
	}

	err = a.repo.EditPassport(passport.ID, editingPassport)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ID":        editingPassport.ID,
		"Name":      editingPassport.Name,
		"Seria":     editingPassport.Seria,
		"Issue":     editingPassport.Issue,
		"Code":      editingPassport.Code,
		"Gender":    editingPassport.Gender,
		"Birthdate": editingPassport.Birthdate,
		"BDplace":   editingPassport.BDplace,
		"Image":     editingPassport.Image,
	})
}

func (a *Application) addPassportToRequest(c *gin.Context) {
	passport_seria := c.Param("passport_seria")
	passport, err := a.repo.GetPassportBySeria(passport_seria)
	if err != nil {
		c.Error(err)
		return
	}
	// вместо структуры для json использую map
	// map: key-value
	// jsonMap: string-int
	// можно использовать string-interface{} (определяемый тип, в данном случае - пустой)
	// тогда будет jsonMap["client_id"].int
	var jsonMap map[string]int

	if err = c.BindJSON(&jsonMap); err != nil {
		c.Error(err)
		return
	}
	log.Println("c_id: ", jsonMap)

	request := &ds.BorderCrossingFacts{}
	request, err = a.repo.CreateBorderCrossingRequest(jsonMap["client_id"])
	if err != nil {
		c.Error(err)
		return
	}

	err = a.repo.AddRequestToBorderCrossingPassports(int(passport.ID), int(request.ID))
	if err != nil {
		c.Error(err)
		return
	}
}

func (a *Application) getAllRequests(c *gin.Context) {
	requests, err := a.repo.GetAllRequests()

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, requests)
}

func (a *Application) getDetailedRequest(c *gin.Context) {
	req_id, err := strconv.Atoi(c.Param("req_id"))
	if err != nil {
		// ... handle error
		panic(err)
	}

	requests, err := a.repo.GetRequestByID(req_id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, requests)
}

func (a *Application) moderChangeTransferRequestStatus(c *gin.Context) {
	var requestBody ds.ChangeBorderCrossingFactStatus

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}
	log.Println("REQ BODY: ", requestBody)

	currRequest, err := a.repo.GetRequestByID(int(requestBody.BorderCrossingFactID))
	if err != nil {
		c.Error(err)
		return
	}

	currUser, err := a.repo.GetUserByName(requestBody.UserName)
	if err != nil {
		c.Error(err)
		return
	}

	if !slices.Contains(ds.ReqStatuses, requestBody.Status) {
		c.String(http.StatusBadRequest, "Неверный статус")
		return
	}

	if *currUser.IsModer != true {
		c.String(http.StatusForbidden, "У пользователя должна быть роль модератора")
		return
	} else {
		if currRequest.ModerRefer == int(currUser.ID) {
			if slices.Contains(ds.ReqStatuses[len(ds.ReqStatuses)-3:], requestBody.Status) {
				err = a.repo.ChangeRequestStatus(int(requestBody.BorderCrossingFactID), requestBody.Status)

				if err != nil {
					c.Error(err)
					return
				}

				c.String(http.StatusCreated, "Текущий статус: ", requestBody.Status)
				return
			} else {
				c.String(http.StatusForbidden, "Модератор не может установить статус ", requestBody.Status)
				return
			}
		} else {
			c.String(http.StatusForbidden, "Модератор не является ответственным")
			return
		}
	}
}

func (a *Application) deleteBorderCrossingFactRequest(c *gin.Context) {
	req_id, err1 := strconv.Atoi(c.Param("req_id"))
	if err1 != nil {
		// ... handle error
		panic(err1)
	}

	err1, err2 := a.repo.DeleteBorderCrossingFactRequest(uint(req_id)), a.repo.DeleteBorderCrossingPassportsEvery(uint(req_id))

	if err1 != nil || err2 != nil {
		c.Error(err1)
		c.Error(err2)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	c.String(http.StatusCreated, "ALL WAS DELETED")
}
