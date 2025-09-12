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
		log.Fatalf("FATAL: Could not check if %s class exists: %v", helpers.WeaviateEventClassName, err)
	}

	if exists {
		log.Println("%s class exists. Deleting it...", helpers.WeaviateEventClassName)
		err = client.Schema().ClassDeleter().WithClassName(helpers.WeaviateEventClassName).Do(context.Background())
		if err != nil {
			log.Fatalf("FATAL: Could not delete existing %s class: %v", helpers.WeaviateEventClassName, err)
		}
		log.Println("Successfully deleted existing %s class.", helpers.WeaviateEventClassName)
	} else {
		log.Println("%s class does not exist. Proceeding to create new schema.", helpers.WeaviateEventClassName)
	}

	log.Println("Schema cleanup and setup complete. Weaviate is ready.")
}
