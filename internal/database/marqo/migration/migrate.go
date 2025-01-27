package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

	// Track all processed documents by ID
	processedIds := make(map[string]bool)
	failedDocs := make(map[string]string)

	// First, get all document IDs from source
	allSourceIds := make(map[string]bool)
	offset := 0
	for {
		docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
		}
		if len(docs) == 0 {
			break
		}

		// Collect all IDs
		for _, doc := range docs {
			if id, ok := doc["_id"].(string); ok {
				allSourceIds[id] = true
			}
		}
		// maybe add a minus one so offset is 99 and the item begins at 100 since we are probably zero indexing
		offset += m.batchSize - 1
	}

	fmt.Printf("Found %d unique document IDs in source\n", len(allSourceIds))

	// Add debug tracking
	batchStats := make([]struct {
		offset    int
		batchSize int
		docsFound int
		firstId   string
		lastId    string
		uniqueIds map[string]bool
	}, 0)

	// Track cumulative stats
	cumulativeIds := make(map[string]bool)
	lastProcessedId := ""

	// Reset offset for actual migration
	offset = 0
	totalMigrated := 0

	// Process documents
	for {
		batchStart := offset
		fmt.Printf("\n=== Batch Analysis (Offset: %d) ===\n", offset)

		docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
		}

		// Collect batch statistics
		batchUnique := make(map[string]bool)
		firstId, lastId := "", ""

		for i, doc := range docs {
			if id, ok := doc["_id"].(string); ok {
				if i == 0 {
					firstId = id
				}
				lastId = id
				batchUnique[id] = true

				if lastProcessedId != "" {
					fmt.Printf("Gap check - Last ID: %s, Current ID: %s\n",
						lastProcessedId, id)
				}

				if cumulativeIds[id] {
					fmt.Printf("WARNING: Duplicate document found - ID: %s (offset: %d)\n",
						id, offset)
				}
				cumulativeIds[id] = true
			}
		}

		stats := struct {
			offset    int
			batchSize int
			docsFound int
			firstId   string
			lastId    string
			uniqueIds map[string]bool
		}{
			offset:    batchStart,
			batchSize: m.batchSize,
			docsFound: len(docs),
			firstId:   firstId,
			lastId:    lastId,
			uniqueIds: batchUnique,
		}
		batchStats = append(batchStats, stats)

		fmt.Printf("Batch Details:\n")
		fmt.Printf("- Offset: %d\n", batchStart)
		fmt.Printf("- Documents found: %d\n", len(docs))
		fmt.Printf("- Unique documents in batch: %d\n", len(batchUnique))
		fmt.Printf("- First document ID: %s\n", firstId)
		fmt.Printf("- Last document ID: %s\n", lastId)

		if len(docs) > 0 && len(docs) < m.batchSize {
			fmt.Printf("WARNING: Partial batch found (expected %d, got %d)\n",
				m.batchSize, len(docs))
		}

		// Track new vs already processed documents in this batch
		newDocs := 0
		skipDocs := 0
		for _, doc := range docs {
			if id, ok := doc["_id"].(string); ok {
				if !processedIds[id] {
					newDocs++
				} else {
					skipDocs++
					fmt.Printf("Warning: Document %s has already been processed\n", id)
				}
			}
		}
		fmt.Printf("Batch contains %d documents (%d new, %d already processed)\n",
			len(docs), newDocs, skipDocs)

		// Process documents we haven't seen before
		transformedDocs := make([]map[string]interface{}, 0, len(docs))
		for _, doc := range docs {
			id, _ := doc["_id"].(string)
			if processedIds[id] {
				continue // Skip already processed documents
			}

			transformed, err := m.applyTransformers(doc)
			if err != nil {
				failedDocs[id] = fmt.Sprintf("transform error: %v", err)
				fmt.Printf("ERROR: Transform failed for document %v: %v\n", id, err)
				continue
			}
			transformedDocs = append(transformedDocs, transformed)
			processedIds[id] = true
		}

		if len(transformedDocs) > 0 {
			if err := m.targetClient.UpsertDocuments(targetIndex, transformedDocs); err != nil {
				fmt.Printf("ERROR: Upsert failed: %v\n", err)
				continue
			}
			totalMigrated += len(transformedDocs)
		}

		lastProcessedId = lastId
		previousOffset := offset
		offset += m.batchSize
		fmt.Printf("Offset progression: %d -> %d\n", previousOffset, offset)
	}

	// Final Analysis
	fmt.Printf("\n=== Pagination Analysis ===\n")
	fmt.Printf("Total batches processed: %d\n", len(batchStats))
	fmt.Printf("Total unique documents found: %d\n", len(cumulativeIds))

	// Check for gaps between batches
	for i := 1; i < len(batchStats); i++ {
		prevBatch := batchStats[i-1]
		currBatch := batchStats[i]
		expectedOffset := prevBatch.offset + prevBatch.docsFound

		if currBatch.offset != expectedOffset {
			fmt.Printf("WARNING: Potential gap between batches:\n")
			fmt.Printf("- Previous batch ended at offset: %d\n", prevBatch.offset+prevBatch.docsFound)
			fmt.Printf("- Next batch started at offset: %d\n", currBatch.offset)
			fmt.Printf("- Gap size: %d\n", currBatch.offset-expectedOffset)
		}
	}

	// Verify migration
	missingIds := []string{}
	for id := range allSourceIds {
		if !processedIds[id] {
			missingIds = append(missingIds, id)
		}
	}

	fmt.Printf("\n=== Migration Summary ===\n")
	fmt.Printf("Total source documents: %d\n", len(allSourceIds))
	fmt.Printf("Successfully processed: %d\n", len(processedIds))
	fmt.Printf("Successfully migrated: %d\n", totalMigrated)

	if len(missingIds) > 0 {
		fmt.Printf("\nMissing Documents (%d):\n", len(missingIds))
		for _, id := range missingIds {
			fmt.Printf("- %s\n", id)
		}
	}

	if len(failedDocs) > 0 {
		fmt.Printf("\nFailed Documents (%d):\n", len(failedDocs))
		for id, reason := range failedDocs {
			fmt.Printf("- ID: %s\n  Reason: %s\n", id, reason)
		}
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
