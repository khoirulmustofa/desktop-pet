package main

import (
	"log"
	"os"

	"desktop-pet/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Printf("desktop-pet: %v", err)
		os.Exit(1)
	}
}
