package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func LoadSchema(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	return schema, nil
}

type Migrator struct {
	sourceClient *MarqoClient
	targetClient *MarqoClient
	batchSize    int
	transformers []TransformFunc
	schema       map[string]interface{}
}

func NewMigrator(sourceURL, targetURL, apiKey string, batchSize int, transformerNames []string, schema map[string]interface{}) (*Migrator, error) {
	// Validate and collect requested transformers
	transformers := make([]TransformFunc, 0)
	if len(transformerNames) > 0 {
		for _, name := range transformerNames {
			transformer, exists := TransformerRegistry[name]
			if !exists {
				return nil, fmt.Errorf("unknown transformer: %s", name)
			}
			transformers = append(transformers, transformer)
		}
	}

	return &Migrator{
		sourceClient: NewMarqoClient(sourceURL, apiKey),
		targetClient: NewMarqoClient(targetURL, apiKey),
		batchSize:    batchSize,
		transformers: transformers,
		schema:       schema,
	}, nil
}

func getAllowedFields(schema map[string]interface{}) map[string]bool {
	allowedFields := make(map[string]bool)
	if allFields, ok := schema["allFields"].([]interface{}); ok {
		for _, field := range allFields {
			if fieldMap, ok := field.(map[string]interface{}); ok {
				if name, ok := fieldMap["name"].(string); ok {
					allowedFields[name] = true
				}
			}
		}
	}
	return allowedFields
}

func removeProtectedFields(doc map[string]interface{}, allowedFields map[string]bool) map[string]interface{} {
	// Create a new map for the cleaned document
	cleaned := make(map[string]interface{})

	for key, value := range doc {
		if key == "_id" || (!strings.HasPrefix(key, "_") && allowedFields[key]) {
			cleaned[key] = value
		}
	}

	return cleaned
}

func (m *Migrator) applyTransformers(doc map[string]interface{}) (map[string]interface{}, error) {
	// Log original document
	eventName, _ := doc["name"].(string)
	eventID, _ := doc["_id"].(string)
	originalJSON, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Printf("\n=== Pre-Transform Event ===\nID: %s\nName: %s\nDocument:\n%s\n",
		eventID, eventName, string(originalJSON))

	allowedFields := getAllowedFields(m.schema)
	result := removeProtectedFields(doc, allowedFields)

	// Track each transformation
	for i, transformer := range m.transformers {
		var err error
		result, err = transformer(result)
		if err != nil {
			return nil, fmt.Errorf("transformer %d failed for document %s: %w", i, eventID, err)
		}
	}

	// Log transformed document
	transformedJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("\n=== Post-Transform Event ===\nID: %s\nName: %s\nDocument:\n%s\n",
		eventID, eventName, string(transformedJSON))

	return result, nil
}

// func (m *Migrator) applyTransformers(doc map[string]interface{}) (map[string]interface{}, error) {
// 	allowedFields := getAllowedFields(m.schema)
//     result := removeProtectedFields(doc, allowedFields)

//     var err error
//     for _, transformer := range m.transformers {
//         result, err = transformer(result)
//         if err != nil {
//             return nil, fmt.Errorf("transformer failed: %w", err)
//         }
//     }

//     return result, nil
// }

func (m *Migrator) MigrateEvents(sourceIndex, targetIndex string, schema map[string]interface{}) error {
	// Create the new index
	endpoint, err := m.targetClient.CreateStructuredIndex(targetIndex, schema)
	if err != nil {
		return fmt.Errorf("failed to create target index: %w", err)
	}

	// Update target client with new endpoint
	m.targetClient = NewMarqoClient(endpoint, m.targetClient.apiKey)
	fmt.Printf("Using target endpoint: %s\n", endpoint)

	offset := 0
	totalMigrated := 0
	batchMap := make(map[string]bool) // Track processed documents by ID
	retryCount := 3                   // Number of retries for failed batches

	// First, get total count of documents
	totalDocs, err := m.sourceClient.GetTotalDocuments(sourceIndex)
	if err != nil {
		return fmt.Errorf("failed to get total document count: %w", err)
	}
	fmt.Printf("Total documents in source index: %d\n", totalDocs)

	for {
		// Fetch batch of documents from source
		docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
		}

		if len(docs) == 0 {
			break
		}

		// Track documents in this batch
		for _, doc := range docs {
			if id, ok := doc["_id"].(string); ok {
				batchMap[id] = true
			}
		}

		// Transform and upsert batch with retry logic
		success := false
		for attempt := 0; attempt < retryCount && !success; attempt++ {
			transformedDocs := make([]map[string]interface{}, 0, len(docs)) // Change to slice with zero length
			for _, doc := range docs {
				transformed, err := m.applyTransformers(doc)
				if err != nil {
					fmt.Printf("Transform error on attempt %d for document %v: %v\n",
						attempt+1, doc["_id"], err)
					continue
				}
				transformedDocs = append(transformedDocs, transformed) // Only append successful transformations
			}

			// Skip empty batches
			if len(transformedDocs) == 0 {
				fmt.Printf("Warning: No documents successfully transformed in batch\n")
				continue
			}

			if err := m.targetClient.UpsertDocuments(targetIndex, transformedDocs); err != nil {
				fmt.Printf("Upsert error on attempt %d: %v\n", attempt+1, err)
				if attempt < retryCount-1 {
					time.Sleep(time.Second * 5) // Wait before retry
					continue
				}
				return err
			}
			success = true
		}

		totalMigrated += len(docs)
		fmt.Printf("Migrated and transformed %d/%d documents (%.2f%%)\n",
			totalMigrated, totalDocs, float64(totalMigrated)/float64(totalDocs)*100)

		offset += m.batchSize
	}

	// Verify migration
	targetDocs, err := m.targetClient.GetTotalDocuments(targetIndex)
	if err != nil {
		return fmt.Errorf("failed to verify target documents: %w", err)
	}

	fmt.Printf("\nMigration Summary:\n")
	fmt.Printf("Source documents: %d\n", totalDocs)
	fmt.Printf("Target documents: %d\n", targetDocs)
	fmt.Printf("Processed documents: %d\n", len(batchMap))

	if targetDocs < totalDocs {
		return fmt.Errorf("migration incomplete: source=%d, target=%d, missing=%d",
			totalDocs, targetDocs, totalDocs-targetDocs)
	}

	return nil
}

// Add this method to MarqoClient
func (c *MarqoClient) GetTotalDocuments(indexName string) (int, error) {
	// Use search with size=0 to get total count
	url := fmt.Sprintf("%s/indexes/%s/search", c.baseURL, indexName)

	requestBody := map[string]interface{}{
		"q":     "*",
		"limit": 0,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}

	c.addHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Total int `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result.Total, nil
}
