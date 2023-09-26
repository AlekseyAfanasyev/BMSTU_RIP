package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Passport struct {
	ID        int
	Title     string
	Seria     string
	Issue     string
	Code      string
	Gender    string
	Birthdate string
	BDplace   string
	Image     string
	OneCard   string
}

func StartServer() {
	log.Println("Server start up")

	Seria := []string{
		"4577 999123", "8866 343789", "1233 541856", "7182 90128712", "0288 63628978", "9192 10902222",
	}

	Issue := []string{
		"11.05.2023", "23.09.2020", "13.09.2013", "12.05.2015", "30.10.2010", "18.06.2018",
	}

	Code := []string{
		"770-045", "225-189", "909-450", "123-909", "890-456", "398-330",
	}

	Gender := []string{
		"МУЖ.", "МУЖ.", "ЖЕН.", "МУЖ.", "ЖЕН.", "ЖЕН.",
	}

	Birthdate := []string{
		"11.03.1992", "26.12.2000", "07.09.1973", "15.10.1999", "30.03.1983", "27.04.2002",
	}

	BDplace := []string{
		"Город Москва", "Город Москва", "Город Калуга", "Город Электросталь", "Город Железнодорожный", "Город Санкт-Петербург",
	}

	Passports := []Passport{
		{0, "Афанасьев Евгений Геннадьевич", Seria[0], Issue[0], Code[0], Gender[0], Birthdate[0], BDplace[0], "../resources/AAA.jpg", "AAA"},
		{1, "Лаптев Григорий Сергеевич", Seria[1], Issue[1], Code[1], Gender[1], Birthdate[1], BDplace[1], "../resources/LGS.jpg", "LGS"},
		{2, "Афанасьева Елена Ивановна", Seria[2], Issue[2], Code[2], Gender[2], Birthdate[2], BDplace[2], "../resources/AEI.jpg", "AEI"},
		{3, "Сорокин Денис Игоревич", Seria[3], Issue[3], Code[3], Gender[3], Birthdate[3], BDplace[3], "../resources/SDI.jpg", "SDI"},
		{4, "Петрова Наталья Валерьевна", Seria[4], Issue[4], Code[4], Gender[4], Birthdate[4], BDplace[4], "../resources/PNV.jpg", "PNV"},
		{5, "Яковлева София Ивановна", Seria[5], Issue[5], Code[5], Gender[5], Birthdate[5], BDplace[5], "../resources/YSI.jpg", "YSI"},
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/resources", "./resources")
	r.Static("/css", "./templates")

	r.GET("/home", func(c *gin.Context) {

		query := c.DefaultQuery("query", "")

		//поиск в Orbits основываясь на query
		var filteredPassports []Passport
		if query != "" {
			for i := 0; i < len(Passports); i++ {
				if strings.Contains(strings.ToLower(Passports[i].Title), strings.ToLower(query)) {
					filteredPassports = append(filteredPassports, Passports[i])
				}
			}
		} else {
			filteredPassports = Passports
		}

		c.HTML(http.StatusOK, "passports.html", gin.H{
			"passports": filteredPassports,
		})
	})

	r.GET("/home/:OneCard ", func(c *gin.Context) {
		card := c.Param("OneCard ")

		for i := range Passports {
			if card == Passports[i].OneCard {
				c.HTML(http.StatusOK, "passport.html", gin.H{
					"Title":     Passports[i].Title,
					"Image":     Passports[i].Image,
					"Seria":     Passports[i].Seria,
					"Issue":     Passports[i].Issue,
					"Code":      Passports[i].Code,
					"Gender":    Passports[i].Gender,
					"Birthdate": Passports[i].Birthdate,
					"BDplace":   Passports[i].BDplace,
				})
			}
		}
	})

	err := r.Run(":8080")
	if err != nil {
		return
	}

	log.Println("Server down")
}
