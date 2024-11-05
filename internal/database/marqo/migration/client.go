package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"	
	"time"
)

type MarqoClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewMarqoClient(baseURL, apiKey string) *MarqoClient {
	// Ensure baseURL ends with /api/v2
    if !strings.HasSuffix(baseURL, "/api/v2") {
        if strings.HasSuffix(baseURL, "/") {
            baseURL = baseURL + "api/v2"
        } else {
            baseURL = baseURL + "/api/v2"
        }
    }

	return &MarqoClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

type CreateIndexRequest struct {
	Type                string             `json:"type"`
	IndexName           string             `json:"indexName"`
	VectorNumericType   string             `json:"vectorNumericType"`
	Model               string             `json:"model"`
	NormalizeEmbeddings bool               `json:"normalizeEmbeddings"`
	TextPreprocessing   *TextPreprocessing `json:"textPreprocessing,omitempty"`
	AllFields           []Field            `json:"allFields"`
	TensorFields        []string           `json:"tensorFields"`
	AnnParameters       *AnnParameters     `json:"annParameters,omitempty"`
	NumberOfShards      int                `json:"numberOfShards"`
	NumberOfReplicas    int                `json:"numberOfReplicas"`
	InferenceType       string             `json:"inferenceType"`
	StorageClass        string             `json:"storageClass"`
	NumberOfInferences  int                `json:"numberOfInferences"`
}

type TextPreprocessing struct {
	SplitLength  int    `json:"splitLength"`
	SplitOverlap int    `json:"splitOverlap"`
	SplitMethod  string `json:"splitMethod"`
}

type Field struct {
	Name            string             `json:"name"`
	Type            string             `json:"type"`
	Features        []string           `json:"features,omitempty"`
	DependentFields map[string]float64 `json:"dependentFields,omitempty"`
}

type AnnParameters struct {
	SpaceType  string     `json:"spaceType"`
	Parameters Parameters `json:"parameters"`
}

type Parameters struct {
	EfConstruction int `json:"efConstruction"`
	M              int `json:"m"`
}

func (c *MarqoClient) CreateStructuredIndex(indexName string, schema map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/v2/indexes/%s", c.baseURL, indexName)

	// Convert schema to proper request format
	req, err := CreateIndexRequestFromSchema(schema)
	if err != nil {
		return fmt.Errorf("failed to create index request: %w", err)
	}

	// Set required fields if not present
	if req.NumberOfShards == 0 {
		req.NumberOfShards = 1
	}
	if req.NumberOfReplicas == 0 {
		req.NumberOfReplicas = 0
	}
	if req.InferenceType == "" {
		req.InferenceType = "marqo.CPU.large"
	}
	if req.StorageClass == "" {
		req.StorageClass = "marqo.basic"
	}
	if req.NumberOfInferences == 0 {
		req.NumberOfInferences = 1
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index: status=%d body=%s",
			resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Helper function to create a structured index request from schema
func CreateIndexRequestFromSchema(schema map[string]interface{}) (*CreateIndexRequest, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	var req CreateIndexRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema into request: %w", err)
	}

	return &req, nil
}

// Add these methods to MarqoClient

func (c *MarqoClient) Search(indexName string, query string, offset, limit int) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/indexes/%s/search", c.baseURL, indexName)

	requestBody := map[string]interface{}{
		"q":      query,
		"offset": offset,
		"limit":  limit,
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
	req.Header.Set("x-api-key", c.apiKey)

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
	url := fmt.Sprintf("%s/api/v2/indexes/%s/documents", c.baseURL, indexName)

	body, err := json.Marshal(documents)
	if err != nil {
		return fmt.Errorf("failed to marshal documents: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

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
	url := fmt.Sprintf("%s/api/v2/indexes", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

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
