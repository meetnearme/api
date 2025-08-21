package startup

import (
	"context"
	"fmt"
	"log"

	"github.com/meetnearme/api/functions/gateway/services"

	_ "github.com/joho/godotenv/autoload"
)

// InitWeaviate sets up the Weaviate schema
func InitWeaviate() error {
	log.Println("Connecting to Weaviate to ensure schema is defined...")

	client, err := services.GetWeaviateClient()
	if err != nil {
		return fmt.Errorf("could not get Weaviate client: %w", err)
	}
	log.Println("Successfully connected to Weaviate.")

	err = services.CreateWeaviateSchemaIfMissing(context.Background(), client)
	if err != nil {
		return fmt.Errorf("could not define Weaviate schema: %w", err)
	}

	log.Println("Schema setup check complete. Weaviate is ready.")
	return nil
}

// init function for backward compatibility
func init() {
	if err := InitWeaviate(); err != nil {
		log.Fatalf("Weaviate setup failed: %v", err)
	}
}
