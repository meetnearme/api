package main

import (
	"context"
	"log"

	"github.com/meetnearme/api/functions/gateway/constants"
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
	exists, err := client.Schema().ClassExistenceChecker().WithClassName(constants.WeaviateEventClassName).Do(context.Background())
	if err != nil {
		log.Fatalf("FATAL: Could not check if %s class exists: %v", constants.WeaviateEventClassName, err)
	}

	if exists {
		log.Printf("%s class exists. Deleting it...", constants.WeaviateEventClassName)
		err = client.Schema().ClassDeleter().WithClassName(constants.WeaviateEventClassName).Do(context.Background())
		if err != nil {
			log.Fatalf("FATAL: Could not delete existing %s class: %v", constants.WeaviateEventClassName, err)
		}
		log.Printf("Successfully deleted existing %s class.", constants.WeaviateEventClassName)
	} else {
		log.Printf("%s class does not exist. Proceeding to create new schema.", constants.WeaviateEventClassName)
	}

	log.Println("Schema cleanup and setup complete. Weaviate is ready.")
}
