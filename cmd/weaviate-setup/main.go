package main

import (
	"context"
	"log"

	"github.com/meetnearme/api/functions/gateway/services"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	log.Println("Connecting to Weaviate to ensure schema is defined...")

	client, err := services.GetWeaviateClient()
	if err != nil {
		log.Fatalf("FATAL: Could not get Weaviate client: %v", err)
	}
	log.Println("Successfully connected to Weaviate.")

	err = services.DefineWeaviateSchema(context.Background(), client)
	if err != nil {
		log.Fatalf("FATAL: Could not define Weaviate schema: %v", err)
	}

	log.Println("Schema setup check complete. Weaviate is ready.")
}
