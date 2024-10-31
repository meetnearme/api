package migration

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)
func main() {
    env := flag.String("env", "", "Environment (dev/prod)")
    schemaPath := flag.String("schema", "", "Path to schema JSON file")
    batchSize := flag.Int("batch-size", 100, "Batch size for migration")
    transformers := flag.String("transformers", "", "Comma-separated list of transformers")
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
    targetURL := constructTargetURL(sourceURL, *env)
    fmt.Printf("Source Marqo URL: %s\n", sourceURL)
    fmt.Printf("Target Marqo URL: %s\n", targetURL)

    apiKey := os.Getenv("MARQO_API_KEY")
    if apiKey == "" {
        fmt.Println("MARQO_API_KEY must be set")
        os.Exit(1)
    }
    // ... rest of the code ...
}

func constructTargetURL(sourceURL, env string) string {
    // Example source URL: https://dev-events-search-index-xv8ywa-g2amp25x.dp1.marqo.ai
    // We want: https://dev-events-search-index-v2-[new-uuid]-g2amp25x.dp1.marqo.ai

    parts := strings.Split(sourceURL, "://")
    if len(parts) != 2 {
        return sourceURL
    }

    protocol := parts[0]
    domain := parts[1]

    // Split domain into parts
    domainParts := strings.Split(domain, ".")
    if len(domainParts) < 2 {
        return sourceURL
    }

    // Get the first part which contains the index identifier
    _ = domainParts[0]

    // Generate new UUID for the target index
    newUUID := strings.ToLower(uuid.New().String()[:6])

    // Construct new index identifier
    // Format: {env}-events-search-index-v2-{uuid}-g2amp25x
    newIndexPart := fmt.Sprintf("%s-events-search-index-v2-%s-g2amp25x", env, newUUID)

    // Replace the index part in domain parts
    domainParts[0] = newIndexPart

    // Reconstruct the URL
    return fmt.Sprintf("%s://%s", protocol, strings.Join(domainParts, "."))
}
