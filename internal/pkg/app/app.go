package app

import (
	"BMSTU_RIP/docs"
	"BMSTU_RIP/internal/app/config"
	"BMSTU_RIP/internal/app/ds"
	"BMSTU_RIP/internal/app/dsn"
	"BMSTU_RIP/internal/app/redis"
	"BMSTU_RIP/internal/app/repository"
	"BMSTU_RIP/internal/app/role"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Application struct {
	repo   *repository.Repository
	r      *gin.Engine
	config *config.Config
	redis  *redis.Client
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResp struct {
	Login       string `json:"login"`
	Role        int    `json:"role"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerResp struct {
	Ok bool `json:"ok"`
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	repo, err := repository.New(dsn.FromEnv())
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}

	return &Application{
		config: cfg,
		repo:   repo,
		redis:  redisClient,
	}, nil
}

func (a *Application) StartServer() {
	log.Println("Server started")

	a.r = gin.Default()

	a.r.LoadHTMLGlob("templates/*.html")
	a.r.Static("/css", "./templates")

	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	a.r.POST("/login", a.login)
	a.r.POST("/register", a.register)
	a.r.POST("/logout", a.logout)

	a.r.GET("passports", a.getAllPassports)
	a.r.GET("passports/:passport_name", a.getDetailedPassport)

	clientMethods := a.r.Group("", a.WithAuthCheck(role.Client))
	{
		clientMethods.POST("/border_crossing_facts/create", a.createBorderCrossingFactRequest)
		//clientMethods.POST("/:passport_seria/add", a.addPassportToRequest)
		clientMethods.POST("border_crossing_fp/:req_id/delete", a.deleteBorderCrossingFactRequest)
		clientMethods.PUT("/border_crossing_facts/set_passports", a.setRequestPassports)
	}
	moderMethods := a.r.Group("", a.WithAuthCheck(role.Moderator))
	{
		moderMethods.PUT("passports/:passport_seria/edit", a.editPassport)
		moderMethods.POST("passports/add_new_passport", a.newPassport)
		moderMethods.POST("change_passport_availibility/:passport_name", a.changeAvailability)
		moderMethods.GET("/ping", a.ping)
	}

	authorizedMethods := a.r.Group("", a.WithAuthCheck(role.Client, role.Moderator))
	{
		authorizedMethods.GET("/border_crossing_passport/:req_id", a.getPassportsFromRequest)
		authorizedMethods.GET("/border_crossing_facts", a.getAllRequests)
		authorizedMethods.GET("/border_crossing_facts/:req_id", a.getDetailedRequest)
		authorizedMethods.PUT("/border_crossing_facts/change_status", a.changeRequestStatus)
	}

	a.r.POST("/delete_passport/:passport_name", func(c *gin.Context) {
		passportName := c.Param("passport_name")

		err := a.repo.ChangeAvailability(passportName)

		if err != nil {
			c.Error(err)
			return
		}

		c.Redirect(http.StatusOK, "/")
	})

	a.r.Run(":8000")

	log.Println("Server is down")
}

// @Summary Зарегистрировать нового пользователя
// @Description Добавляет в БД нового пользователя
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {object} registerResp
// @Param request_body body registerReq true "Данные для регистрации"
// @Router /register [post]
func (a *Application) register(gCtx *gin.Context) {
	req := &registerReq{}

	err := json.NewDecoder(gCtx.Request.Body).Decode(req)
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if req.Password == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("pass is empty"))
		return
	}

	if req.Login == "" {
		gCtx.AbortWithError(http.StatusBadRequest, fmt.Errorf("name is empty"))
		return
	}

	err = a.repo.Register(&ds.UserUID{
		UUID: uuid.New(),
		Role: role.Client,
		Name: req.Login,
		Pass: a.repo.GenerateHashString(req.Password), // пароли делаем в хешированном виде и далее будем сравнивать хеши, чтобы их не угнали с базой вместе
	})
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	gCtx.JSON(http.StatusOK, &registerResp{
		Ok: true,
	})
}

// @Summary Вход в систему
// @Description Проверяет данные для входа и в случае успеха возвращает токен для входа
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {object} loginResp
// @Param request_body body loginReq true "Данные для входа"
// @Router /login [post]
func (a *Application) login(c *gin.Context) {
	cfg := a.config
	req := &loginReq{}
	err := json.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)

		return
	}

	user, err := a.repo.GetUserByName(req.Login)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	if req.Login == user.Name && user.Pass == a.repo.GenerateHashString(req.Password) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &ds.JWTClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Second * 3600).Unix(), //1h
				IssuedAt:  time.Now().Unix(),
				Issuer:    "web-admin",
			},
			UserUUID: user.UUID,
			Role:     user.Role,
		})

		if token == nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Токен = nil"))

			return
		}

		strToken, err := token.SignedString([]byte(cfg.JWT.Token))
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Невозможно получить строку из токена"))

			return
		}

		//httpOnly=true, secure=true -> не могу читать куки на фронте ...
		c.SetCookie("orbits-api-token", "Bearer "+strToken, int(time.Now().Add(time.Second*3600).
			Unix()), "", "", true, true)

		c.JSON(http.StatusOK, loginResp{
			Login:       user.Name,
			Role:        int(user.Role),
			AccessToken: strToken,
			TokenType:   "Bearer",
			ExpiresIn:   int(cfg.JWT.ExpiresIn.Seconds()),
		})
		log.Println("\nUSER: ", user.Name, "\n", strToken, "\n")
		c.AbortWithStatus(http.StatusOK)
	} else {
		c.AbortWithStatus(http.StatusForbidden)
	}
}

// @Summary Выйти из системы
// @Details Деактивирует текущий токен пользователя, добавляя его в блэклист в редисе
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200
// @Router /logout [post]
func (a *Application) logout(gCtx *gin.Context) {
	// получаем заголовок
	jwtStr := gCtx.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
		gCtx.AbortWithStatus(http.StatusBadRequest) // отдаем что нет доступа

		return // завершаем обработку
	}

	// отрезаем префикс
	jwtStr = jwtStr[len(jwtPrefix):]

	_, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.JWT.Token), nil
	})
	if err != nil {
		gCtx.AbortWithError(http.StatusBadRequest, err)
		log.Println(err)

		return
	}

	// сохраняем в блеклист редиса
	err = a.redis.WriteJWTToBlackList(gCtx.Request.Context(), jwtStr, a.config.JWT.ExpiresIn)
	if err != nil {
		gCtx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	gCtx.Status(http.StatusOK)
}

type pingReq struct{}
type pingResp struct {
	Status string `json:"status"`
}

// Ping godoc
// @Summary      Show hello text
// @Description  very friendly response
// @Tags         Tests
// @Produce      json
// @Success      200  {object}  pingResp
// @Router       /ping/{name} [get]
func (a *Application) ping(gCtx *gin.Context) {
	name := gCtx.Param("name")
	gCtx.String(http.StatusOK, "Hello %s", name)
}

// @Summary Получение всех паспортов
// @Description Возвращает все доступные паспорта
// @Tags Паспорта
// @Accept json
// @Produce json
// @Success 302 {} json
// @Param passport_name query string false "Название паспорта или его часть"
// @Router /passports [get]
func (a *Application) getAllPassports(c *gin.Context) {
	passportName := c.Query("passport_name")

	allPassports, err := a.repo.GetAllPassports(passportName)

	if err != nil {
		c.Error(err)
	}

	c.JSON(http.StatusOK, allPassports)
}

// @Summary      Получение детализированной информации о паспорте
// @Description  Возвращает подробную информацию о паспорте по его названию
// @Tags         Паспорта
// @Produce      json
// @Param passport_name path string true "Название паспорта"
// @Success      200  {object}  string
// @Router       /passports/{passport_name} [get]
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

	c.Redirect(http.StatusOK, "/passports")
}

// @Summary      Добавление нового паспорта
// @Description  Добавляет паспорт с полями, указанныим в JSON
// @Tags Орбиты
// @Accept json
// @Produce      json
// @Param passport body ds.Passports true "Данные нового паспорта"
// @Success      201  {object}  string
// @Router       /orbits/add_new_passport [post]
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

// @Summary      Изменение паспорта
// @Description  Обновляет данные о паспорте, основываясь на полях из JSON
// @Tags         Паспорта
// @Accept 		 json
// @Produce      json
// @Param passport body ds.Passports false "Паспорт"
// @Success      201  {object}  string
// @Router       /passports/{passport_seria}/edit [put]
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

	c.JSON(http.StatusCreated, gin.H{
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

// @Summary      Добавление паспорта в заявку
// @Description  Создает заявку в статусе (или добавляет в открытую) и добавляет выбранный паспорт
// @Tags Общее
// @Accept json
// @Produce      json
// @Success      200  {object}  string
// @Param Body body jsonMap true "Данные заказа"
// @Router       /{passport_seria}/add [post]
//func (a *Application) addPassportToRequest(c *gin.Context) {
//	passport_seria := c.Param("passport_seria")
//	passport, err := a.repo.GetPassportBySeria(passport_seria)
//	if err != nil {
//		c.Error(err)
//		return
//	}
//
//	userUUID, exists := c.Get("userUUID")
//	if !exists {
//		panic(exists)
//	}
//
//	request := &ds.BorderCrossingFacts{}
//	request, err = a.repo.CreateBorderCrossingRequest(userUUID.(uuid.UUID))
//	if err != nil {
//		c.Error(err)
//		return
//	}
//
//	err = a.repo.AddRequestToBorderCrossingPassports(int(passport.ID), int(request.ID))
//	if err != nil {
//		c.Error(err)
//		return
//	}
//}

func (a *Application) createBorderCrossingFactRequest(c *gin.Context) {
	var request_body ds.CreateBorderCrossingFactBody

	if err := c.BindJSON(&request_body); err != nil {
		c.String(http.StatusBadGateway, "Не могу распознать json")
		return
	}

	_userUUID, ok := c.Get("userUUID")

	if !ok {
		c.String(http.StatusInternalServerError, "Вы сначала должны залогиниться")
		return
	}

	userUUID := _userUUID.(uuid.UUID)
	reqID, err := a.repo.CreateBorderCrossingRequest(request_body, userUUID)

	if err != nil {
		c.Error(err)
		c.String(http.StatusNotFound, "Не могу добавить паспорт")
		return
	}

	c.JSON(http.StatusCreated, reqID)
}

// @Summary      Получение всех заявок
// @Description  Получает все заявки
// @Tags         Заявки
// @Produce      json
// @Success      200  {object}  string
// @Router       /border_crossing_facts [get]
func (a *Application) getAllRequests(c *gin.Context) {
	dateStart := c.Query("date_start")
	dateFin := c.Query("date_fin")
	status := c.Query("status")
	log.Println(status)

	userRole, exists := c.Get("userRole")
	if !exists {
		panic(exists)
	}
	//userUUID, exists := c.Get("userUUID")
	//if !exists {
	//	panic(exists)
	//}

	requests, err := a.repo.GetAllRequests(userRole, dateStart, dateFin, status)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, requests)
}

// @Summary      Получение детализированной заявки
// @Description  Получает подробную информаицю о заявке
// @Tags         Заявки
// @Produce      json
// @Param req_id path string true "ID заявки"
// @Success      301  {object}  string
// @Router       /border_crossing_facts/id/{req_id} [get]
func (a *Application) getDetailedRequest(c *gin.Context) {
	req_id, err := strconv.Atoi(c.Param("req_id"))
	if err != nil {
		log.Println("REQ ID: ", req_id)
		panic(err)
	}

	userUUID, exists := c.Get("userUUID")
	if !exists {
		panic(exists)
	}
	userRole, exists := c.Get("userRole")
	if !exists {
		panic(exists)
	}

	request, err := a.repo.GetRequestByID(uint(req_id), userUUID.(uuid.UUID), userRole)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, request)
}
func (a *Application) changeRequestStatus(c *gin.Context) {
	var requestBody ds.ChangeBorderCrossingFactStatus

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}
	log.Println(requestBody)

	userRole, exists := c.Get("userRole")
	if !exists {
		panic(exists)
	}
	userUUID, exists := c.Get("userUUID")
	if !exists {
		panic(exists)
	}
	log.Println("ok: ", userRole, userUUID)
	currRequest, err := a.repo.GetRequestByID(requestBody.BorderCrossingFactID, userUUID.(uuid.UUID), userRole)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}
	log.Println("ok curr")
	if !slices.Contains(ds.ReqStatuses, requestBody.Status) {
		c.String(http.StatusBadRequest, "Неверный статус")
		return
	}

	if userRole == role.Client {
		if currRequest.ClientRefer == userUUID {
			if slices.Contains(ds.ReqStatuses[:3], requestBody.Status) {
				if currRequest.Status != ds.ReqStatuses[0] {
					c.String(http.StatusBadRequest, "Нельзя поменять статус с ", currRequest.Status,
						" на ", requestBody.Status)
					return
				}
				err = a.repo.ChangeRequestStatus(requestBody.BorderCrossingFactID, requestBody.Status)

				if err != nil {
					c.Error(err)
					return
				}

				c.String(http.StatusCreated, "Текущий статус: ", requestBody.Status)
				return
			} else {
				c.String(http.StatusForbidden, "Клиент не может установить статус ", requestBody.Status)
				return
			}
		} else {
			c.String(http.StatusForbidden, "Клиент не является ответственным")
			return
		}
	} else {
		if currRequest.ModerRefer == userUUID {
			if slices.Contains(ds.ReqStatuses[len(ds.ReqStatuses)-2:], requestBody.Status) {
				err = a.repo.ChangeRequestStatus(requestBody.BorderCrossingFactID, requestBody.Status)

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

func (a *Application) getPassportsFromRequest(c *gin.Context) { // нужно добавить проверку на авторизацию пользователя
	req_id, err := strconv.Atoi(c.Param("req_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Ошибка в ID заявки")
		return
	}

	orbits, err := a.repo.GetPassportsFromRequest(req_id)
	log.Println(orbits)
	if err != nil {
		c.String(http.StatusInternalServerError, "Ошибка при получении орбит из заявки")
		return
	}

	c.JSON(http.StatusOK, orbits)

}

// @Summary      Логическое удаление заявки
// @Description  Изменяет статус заявки на "Удалена"
// @Tags         Заявки
// @Produce      json
// @Success      200  {object}  string
// @Param req_id path string true "ID заявки"
// @Router /border_crossing_fp/{req_id}/delete [post]

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

func (a *Application) setRequestPassports(c *gin.Context) {
	var requestBody ds.SetRequestPassportsRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Не получается распознать json запрос")
		return
	}

	err := a.repo.SetRequestPassports(requestBody.RequestID, requestBody.Passports)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получилось задать регионы для заявки\n"+err.Error())
	}

	c.String(http.StatusCreated, "Регионы заявки успешно заданы!")

}
