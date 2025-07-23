package main

import (
	"log"
)

func main() {
	if err := PrepareAndValidateEnv(); err != nil {
		log.Fatalf("Environment preparation failed: %v", err)
	}
	log.Println("Environment validated successfully.")
}
