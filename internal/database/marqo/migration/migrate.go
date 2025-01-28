package main

import (
	"encoding/json"
	"fmt"
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
	eventName, _ := doc["name"].(string)
	eventID, _ := doc["_id"].(string)

	// Only log detailed transformation for sampling (e.g., every 50th document)
	shouldLogDetail := eventID[len(eventID)-2:] == "00" // Sample docs ending in "00"

	if shouldLogDetail {
		originalJSON, _ := json.MarshalIndent(doc, "", "  ")
		fmt.Printf("\n=== Sample Transform Event ===\nID: %s\nName: %s\nDocument:\n%s\n",
			eventID, eventName, string(originalJSON))
	}

	allowedFields := getAllowedFields(m.schema)
	result := removeProtectedFields(doc, allowedFields)

	// Track each transformation
	for i, transformer := range m.transformers {
		var err error
		result, err = transformer(result)
		if err != nil {
			// Always log errors in detail
			originalJSON, _ := json.MarshalIndent(doc, "", "  ")
			fmt.Printf("\n=== Failed Transform Event ===\nID: %s\nName: %s\nDocument:\n%s\nError: %v\n",
				eventID, eventName, string(originalJSON), err)
			return nil, fmt.Errorf("transformer %d failed for document %s: %w", i, eventID, err)
		}
	}

	if shouldLogDetail {
		transformedJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("\n=== Sample Transform Result ===\nID: %s\nName: %s\nDocument:\n%s\n",
			eventID, eventName, string(transformedJSON))
	}

	return result, nil
}

func (m *Migrator) MigrateEvents(sourceIndex, targetIndex string, schema map[string]interface{}) error {
	// Create the new index
	endpoint, err := m.targetClient.CreateStructuredIndex(targetIndex, schema)
	if err != nil {
		return fmt.Errorf("failed to create target index: %w", err)
	}

	// Update target client with new endpoint
	m.targetClient = NewMarqoClient(endpoint, m.targetClient.apiKey)
	fmt.Printf("Using target endpoint: %s\n", endpoint)

	// Get source index stats
	stats, err := m.sourceClient.GetIndexStats(sourceIndex)
	if err != nil {
		return fmt.Errorf("failed to get source index stats: %w", err)
	}
	fmt.Printf("Source index has %d documents\n", stats.NumberOfDocuments)

	// Get all documents from source
	documents, err := m.sourceClient.Search(sourceIndex, "*", 0, stats.NumberOfDocuments)
	if err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}

	// Save raw documents to temp file
	tempFile := fmt.Sprintf("temp_migration_%s_%d.json", sourceIndex, time.Now().Unix())
	if err := SaveDocumentsToFile(documents, tempFile); err != nil {
		return fmt.Errorf("failed to save documents to file: %w", err)
	}
	fmt.Printf("Saved %d documents to %s\n", len(documents), tempFile)

	// Load documents from temp file
	documents, err = LoadDocumentsFromFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to load documents from file: %w", err)
	}

	// Apply transformers to each document
	transformedDocs := make([]map[string]interface{}, 0, len(documents))
	for _, doc := range documents {
		transformed, err := m.applyTransformers(doc)
		if err != nil {
			fmt.Printf("Warning: Failed to transform document: %v\n", err)
			continue
		}

		// Debug logging for transformation
		fmt.Printf("Document ID: %s\n", transformed["_id"])
		fmt.Printf("Name: %s\n", transformed["name"])
		if configID, ok := transformed["competitionConfigId"]; ok {
			fmt.Printf("Competition Config ID: %v\n", configID)
		} else {
			fmt.Printf("Competition Config ID field missing\n")
		}

		transformedDocs = append(transformedDocs, transformed)
		fmt.Printf("Added competitionConfigId for event: %s\n", transformed["name"])
	}

	// Save transformed documents to file
	transformedFile := fmt.Sprintf("transformed_migration_%s_%d.json", sourceIndex, time.Now().Unix())
	if err := SaveDocumentsToFile(transformedDocs, transformedFile); err != nil {
		return fmt.Errorf("failed to save transformed documents to file: %w", err)
	}
	fmt.Printf("Saved %d transformed documents to %s\n", len(transformedDocs), transformedFile)

	// Load transformed documents for upserting
	transformedDocs, err = LoadDocumentsFromFile(transformedFile)
	if err != nil {
		return fmt.Errorf("failed to load transformed documents from file: %w", err)
	}

	// Process in batches for upserting
	batchSize := 50
	for i := 0; i < len(transformedDocs); i += batchSize {
		end := i + batchSize
		if end > len(transformedDocs) {
			end = len(transformedDocs)
		}

		batch := transformedDocs[i:end]
		fmt.Printf("Upserting batch %d to %d\n", i, end-1)

		if err := m.targetClient.UpsertDocuments(targetIndex, batch); err != nil {
			fmt.Printf("ERROR: Upsert failed: %v\n", err)
			continue
		}
	}

	// Verify final count
	targetStats, err := m.targetClient.GetIndexStats(targetIndex)
	if err != nil {
		return fmt.Errorf("failed to get target index stats: %w", err)
	}

	fmt.Printf("\n=== Migration Summary ===\n")
	fmt.Printf("Source documents: %d\n", stats.NumberOfDocuments)
	fmt.Printf("Transformed documents: %d\n", len(transformedDocs))
	fmt.Printf("Target documents: %d\n", targetStats.NumberOfDocuments)

	return nil
}
