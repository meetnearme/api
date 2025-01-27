package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	offset := 0
	totalMigrated := 0

	for {
		fmt.Printf("\n=== Processing batch at offset %d ===\n", offset)

		docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
		}

		if len(docs) == 0 {
			break
		}

		// Simply process all documents without any duplicate checking
		transformedDocs := make([]map[string]interface{}, 0, len(docs))
		for _, doc := range docs {
			transformed, err := m.applyTransformers(doc)
			if err != nil {
				fmt.Printf("Warning: Failed to transform document %v: %v\n", doc["_id"], err)
				continue
			}
			transformedDocs = append(transformedDocs, transformed)
		}

		if err := m.targetClient.UpsertDocuments(targetIndex, transformedDocs); err != nil {
			fmt.Printf("ERROR: Upsert failed: %v\n", err)
			continue
		}
		totalMigrated += len(transformedDocs)

		if len(docs) < m.batchSize {
			break
		}

		offset += m.batchSize
		fmt.Printf("Offset progression: %d -> %d\n", offset, offset+m.batchSize)
	}

	fmt.Printf("\n=== Migration Summary ===\n")
	fmt.Printf("Total documents migrated: %d\n", totalMigrated)

	return nil
}
