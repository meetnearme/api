package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	env := flag.String("env", "", "Environment (dev/prod)")
	schemaPath := flag.String("schema", "", "Path to schema JSON file")
	batchSize := flag.Int("batch-size", 100, "Batch size for migration")
	transformersList := flag.String("transformers", "", "Comma-separated list of transformers")
	flag.Parse()

	if *env == "" || *schemaPath == "" {
		fmt.Println("Environment and schema path must be specified")
		os.Exit(1)
	}

	// Get source URL based on environment
	var sourceURL string
	if *env == "prod" {
		sourceURL = os.Getenv("PROD_MARQO_API_BASE_URL")
	} else {
		sourceURL = os.Getenv("DEV_MARQO_API_BASE_URL")
	}

	if sourceURL == "" {
		fmt.Printf("MARQO_API_BASE_URL not set for %s environment\n", *env)
		os.Exit(1)
	}

	// Construct target URL based on Marqo's URL pattern
	targetURL := sourceURL
	fmt.Printf("Marqo API URL: %s\n", sourceURL)

	apiKey := os.Getenv("MARQO_API_KEY")
	if apiKey == "" {
		fmt.Println("MARQO_API_KEY must be set")
		os.Exit(1)
	}

	// Load schema
	schema, err := LoadSchema(*schemaPath)
	if err != nil {
		fmt.Printf("Failed to load schema: %v\n", err)
		os.Exit(1)
	}

	// Split transformers string into slice
	var transformerNames []string
	if *transformersList != "" && !strings.HasPrefix(*transformersList, "-"){
		transformerNames = strings.Split(*transformersList, ",")
		for i, name := range transformerNames {
			name = strings.TrimSpace(name)
			if !strings.HasPrefix(name, "-") {
				transformerNames[i] = name
			}
		}
	}

	migrator, err := NewMigrator(sourceURL, targetURL, apiKey, *batchSize, transformerNames, schema)
	if err != nil {
		fmt.Printf("Failed to create migrator, %v\n", err)
		os.Exit(1)
	}

	sourceIndex, err := migrator.sourceClient.GetCurrentIndex(*env)
	if err != nil {
		fmt.Printf("Failed to get current index: %v\n", err)
		os.Exit(1)
	}

	timestamp := time.Now().UTC().Format("20060102150405")
	targetIndex := fmt.Sprintf("%s-events-search-index-%s", *env, timestamp)

	fmt.Printf("Starting migration from %s to %s\n", sourceIndex, targetIndex)
	fmt.Printf("Using transformers: %v\n", transformerNames)
	fmt.Printf("Batch size: %d\n", *batchSize)

	// Run migration
	if err := migrator.MigrateEvents(sourceIndex, targetIndex, schema); err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully")

}


func (c *MarqoClient) GetCurrentIndex(envPrefix string) (string, error) {
	url := fmt.Sprintf("%s/indexes", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get indexes: %w", err)
	}
	defer resp.Body.Close()

	var result ListIndexesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// find most recent index with our prefix
	var mostRecent string
	var mostRecentTime time.Time

	prefix := fmt.Sprintf("%s-events-search-index-", envPrefix)
	for _, idx := range result.Results {
		if strings.HasPrefix(idx.IndexName, prefix) {
			parts := strings.Split(idx.IndexName, "-")
			if len(parts) < 4 {
				continue
			}

			timestamp := parts[len(parts)-2]
			indexTime, err := time.ParseInLocation("20060102150405", timestamp, time.UTC)
			if err != nil {
				continue
			}

			if mostRecent == "" || indexTime.After(mostRecentTime) {
				mostRecent = idx.IndexName
				mostRecentTime = indexTime
			}
		}
	}

	if mostRecent == "" {
		return fmt.Sprintf("%s-events-search-index", envPrefix), nil
	}

	return mostRecent, nil
}
