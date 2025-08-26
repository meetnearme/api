package main

import (
	"context"
	"log"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	log.Println("Connecting to Weaviate to clean and redefine schema...")

	client, err := services.GetWeaviateClient()
	if err != nil {
		log.Fatalf("FATAL: Could not get Weaviate client: %v", err)
	}
	log.Println("Successfully connected to Weaviate.")

	// Check if the event class exists and delete it
	exists, err := client.Schema().ClassExistenceChecker().WithClassName(helpers.WeaviateEventClassName).Do(context.Background())
	if err != nil {
		log.Fatalf("FATAL: Could not check if Event class exists: %v", err)
	}

	if exists {
		log.Println("Event class exists. Deleting it...")
		err = client.Schema().ClassDeleter().WithClassName("Event").Do(context.Background())
		if err != nil {
			log.Fatalf("FATAL: Could not delete existing Event class: %v", err)
		}
		log.Println("Successfully deleted existing Event class.")
	} else {
		log.Println("Event class does not exist. Proceeding to create new schema.")
	}

	log.Println("Schema cleanup and setup complete. Weaviate is ready.")
}
