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
    sourceClient  *MarqoClient
    targetClient  *MarqoClient
    batchSize     int
    transformers  []TransformFunc
	schema map[string]interface{}
}

func NewMigrator(sourceURL, targetURL, apiKey string, batchSize int, transformerNames []string, schema map[string]interface{}) (*Migrator, error) {
    // Validate and collect requested transformers
    transformers := make([]TransformFunc, 0, len(transformerNames))
    for _, name := range transformerNames {
        transformer, exists := TransformerRegistry[name]
        if !exists {
            return nil, fmt.Errorf("unknown transformer: %s", name)
        }
        transformers = append(transformers, transformer)
    }

    return &Migrator{
        sourceClient:  NewMarqoClient(sourceURL, apiKey),
        targetClient:  NewMarqoClient(targetURL, apiKey),
        batchSize:     batchSize,
        transformers:  transformers,
		schema: schema,
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

	// Copy all fields except protected ones (those starting with _)
	for key, value := range doc {
		if !strings.HasPrefix(key, "_") && allowedFields[key] {
			cleaned[key] = value
		}
	}

	return cleaned
}

func (m *Migrator) applyTransformers(doc map[string]interface{}) (map[string]interface{}, error) {
	allowedFields := getAllowedFields(m.schema)
    result := removeProtectedFields(doc, allowedFields)

	// Create the name_description_address field
	result["name_description_address"] = map[string]interface{} {
		"name": result["name"],
		"eventOwnerName": result["eventOwnerName"],
		"description": result["description"],
		"address": result["address"],
	}

    var err error
    for _, transformer := range m.transformers {
        result, err = transformer(result)
        if err != nil {
            return nil, fmt.Errorf("transformer failed: %w", err)
        }
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

	type batchResult struct {
		count int
		err error
	}

	workers := 4
	batchChan := make(chan []map[string]interface{}, workers)
	resultChan := make(chan batchResult, workers)

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go func() {
			for batch := range batchChan {
				// Transform and upsert batch
				transformedDocs := make([]map[string]interface{}, len(batch))
				for i, doc := range batch {
					transformed, err := m.applyTransformers(doc)
					if err != nil {
						resultChan <- batchResult{err: err}
						return
					}
					transformedDocs[i] = transformed
				}

				if err := m.targetClient.UpsertDocuments(targetIndex, transformedDocs); err != nil {
					resultChan <- batchResult{err: err}
					return
				}

				resultChan <- batchResult{count: len(batch)}
			}
		}()
	}

    offset := 0
    totalMigrated := 0
	activeBatches := 0

    for {
        // Fetch batch of documents from source
        docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
        if err != nil {
            return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
        }

        if len(docs) == 0 {
            break
        }

		// Send batch to worker
		batchChan <- docs
		activeBatches++
		offset += m.batchSize

		if activeBatches == workers {
			result := <-resultChan
			if result.err != nil {
				close(batchChan)
				return result.err
			}
			totalMigrated += result.count
			activeBatches--
			fmt.Printf("Migrated and transformed %d documents\n", totalMigrated)
		}
    }

	// close batch channel and wait for remaining results
	close(batchChan)
	for activeBatches > 0 {
		result := <-resultChan
		if result.err != nil {
			return result.err
		}

		totalMigrated += result.count
		activeBatches--
		fmt.Printf("Migrated and transformed %d documents\n", totalMigrated)
	}

    fmt.Printf("Migration completed. Total documents migrated and transformed: %d\n", totalMigrated)
    return nil
}
