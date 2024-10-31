package migration

import (
	"flag"
	"fmt"
	"os"
)

func main() {
    env := flag.String("env", "", "Environment (dev/prod)")
    schemaPath := flag.String("schema", "", "Path to schema JSON file")
    batchSize := flag.Int("batch-size", 100, "Batch size for migration")
    flag.Parse()

    if *env == "" || *schemaPath == "" {
        fmt.Println("Environment and schema path must be specified")
        os.Exit(1)
    }

    // Reference existing environment variables from marqo_service.go
    // See lines 38-54 in api/functions/gateway/services/marqo_service.go
    sourceURL := os.Getenv("DEV_MARQO_API_BASE_URL")
    if *env == "prod" {
        sourceURL = os.Getenv("PROD_MARQO_API_BASE_URL")
    }

    targetURL := os.Getenv("TARGET_MARQO_URL")
    apiKey := os.Getenv("MARQO_API_KEY")

    migrator := NewMigrator(sourceURL, targetURL, apiKey, *batchSize)
    schema, err := loadSchema(*schemaPath)
    if err != nil {
        fmt.Printf("Failed to load schema: %v\n", err)
        os.Exit(1)
    }

    sourceIndex := fmt.Sprintf("%s-events-search-index", *env)
    targetIndex := fmt.Sprintf("%s-events-search-index-new", *env)

    fmt.Printf("Starting migration from %s to %s\n", sourceIndex, targetIndex)

    if err := migrator.MigrateEvents(sourceIndex, targetIndex, schema); err != nil {
        fmt.Printf("Migration failed: %v\n", err)
        os.Exit(1)
    }
}

