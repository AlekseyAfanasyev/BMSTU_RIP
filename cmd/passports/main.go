package main

import (
	"BMSTU_RIP/internal/api"
	"log"
)

func main() {
	log.Println("Application starts!")
	api.StartServer()
	log.Println("Application terminated!")
}
