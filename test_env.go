package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func testEnv() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
	}
	if os.Getenv("JWT_SECRET") != "" {
		fmt.Println("JWT_SECRET: present")
	} else {
		fmt.Println("JWT_SECRET: absent")
	}
}
