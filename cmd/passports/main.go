package main

import (
	"BMSTU_RIP/internal/pkg/app"
	"context"
	"log"
)

// @title passports
// @version 0.0-0
// @description passports

// @host localhost:8000
// @schemes http
// @BasePath /

func main() {
	log.Println("Application start!")

	a, err := app.New(context.Background())
	if err != nil {
		log.Println(err)

		return
	}

	a.StartServer()

	log.Println("Application terminated!")
}
