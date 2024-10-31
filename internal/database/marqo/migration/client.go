package migration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type MarqoClient struct {
	baseURL string
	apiKey string
	client *http.Client
}

func NewMarqoClient(baseURL, apiKey string) *MarqoClient {
	return &MarqoClient{
		baseURL: baseURL,
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (c *MarqoClient) CreateStructuredIndex(indexName string, schema map[string]interface{}) error {
	url := fmt.Sprintf("%s/indexes", c.baseURL)

	// Add index name to schema
	schema["indexName"] = indexName

	body, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// Add these methods to MarqoClient

func (c *MarqoClient) Search(indexName string, query string, offset, limit int) ([]map[string]interface{}, error) {
    url := fmt.Sprintf("%s/indexes/%s/search", c.baseURL, indexName)

    requestBody := map[string]interface{}{
        "q": query,
        "offset": offset,
        "limit": limit,
    }

    body, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal search request: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        Hits []map[string]interface{} `json:"hits"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return result.Hits, nil
}

func (c *MarqoClient) UpsertDocuments(indexName string, documents []map[string]interface{}) error {
    url := fmt.Sprintf("%s/indexes/%s/documents", c.baseURL, indexName)

    body, err := json.Marshal(documents)
    if err != nil {
        return fmt.Errorf("failed to marshal documents: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("failed to upsert documents: status=%d body=%s", resp.StatusCode, string(body))
    }

    return nil
}

func (c *MarqoClient) withRetry(fn func() error) error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        if i < maxRetries-1 {
            time.Sleep(time.Second * time.Duration(i+1))
        }
    }
    return fmt.Errorf("operation failed after %d retries", maxRetries)
}

func (c *MarqoClient) ListIndexes() ([]string, error) {
    url := fmt.Sprintf("%s/indexes", c.baseURL)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        Results []struct {
            IndexName string `json:"indexName"`
        } `json:"results"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    indexes := make([]string, len(result.Results))
    for i, idx := range result.Results {
        indexes[i] = idx.IndexName
    }

    return indexes, nil
}
