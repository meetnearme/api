package migration

import (
	"encoding/json"
	"fmt"
	"os"
)


func loadSchema(path string) (map[string]interface{}, error) {
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
}

func NewMigrator(sourceURL, targetURL, apiKey string, batchSize int, transformerNames []string) (*Migrator, error) {
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
    }, nil
}

func (m *Migrator) applyTransformers(doc map[string]interface{}) (map[string]interface{}, error) {
    result := doc
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
    if err := m.targetClient.CreateStructuredIndex(targetIndex, schema); err != nil {
        return fmt.Errorf("failed to create target index: %w", err)
    }

    offset := 0
    totalMigrated := 0

    for {
        // Fetch batch of documents from source
        docs, err := m.sourceClient.Search(sourceIndex, "*", offset, m.batchSize)
        if err != nil {
            return fmt.Errorf("failed to fetch documents at offset %d: %w", offset, err)
        }

        if len(docs) == 0 {
            break
        }

        // Apply transformers to each document
        transformedDocs := make([]map[string]interface{}, len(docs))
        for i, doc := range docs {
            transformed, err := m.applyTransformers(doc)
            if err != nil {
                return fmt.Errorf("failed to transform document at offset %d: %w", offset+i, err)
            }
            transformedDocs[i] = transformed
        }

        // Upsert transformed documents to target
        if err := m.targetClient.UpsertDocuments(targetIndex, transformedDocs); err != nil {
            return fmt.Errorf("failed to upsert documents at offset %d: %w", offset, err)
        }

        totalMigrated += len(docs)
        offset += m.batchSize
        fmt.Printf("Migrated and transformed %d documents\n", totalMigrated)
    }

    fmt.Printf("Migration completed. Total documents migrated and transformed: %d\n", totalMigrated)
    return nil
}
