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

type IndexInfo struct {
	IndexName     string `json:"indexName"`
	IndexStatus   string `json:"indexStatus"`
	MarqoEndpoint string `json:"marqoEndpoint"`
	Created       string `json:"Created"`
}

type ListIndexesResponse struct {
	Results []IndexInfo `json:"results"`
}

type Schema struct {
	Type                string             `json:"type"`
	VectorNumericType   string             `json:"vectorNumericType"`
	Model               string             `json:"model"`
	NormalizeEmbeddings bool               `json:"normalizeEmbeddings"`
	TextPreprocessing   *TextPreprocessing `json:"textPreprocessing,omitempty"`
	AllFields           []Field            `json:"allFields"`
	TensorFields        []string           `json:"tensorFields"`
	AnnParameters       *AnnParameters     `json:"annParameters,omitempty"`
}

// Validate checks if the schema meets our requirements
func (s *Schema) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("type is required")
	}
	if s.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(s.AllFields) == 0 {
		return fmt.Errorf("at least one field is required")
	}

	// Validate each field
	for _, field := range s.AllFields {
		if field.Name == "" {
			return fmt.Errorf("field name is required")
		}
		if field.Type == "" {
			return fmt.Errorf("field type is required for field %s", field.Name)
		}
	}

	return nil
}

type CreateIndexRequest struct {
	Type                string             `json:"type"`
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

func NewMarqoClient(baseURL, apiKey string) *MarqoClient {
	// This trimming is not strictly needed but it adds clarity and readability to all of the Sprintf functions called in Search, Upsert etc below
	baseURL = strings.TrimRight(baseURL, "/")

	return &MarqoClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

func (c *MarqoClient) CreateStructuredIndex(indexName string, schema map[string]interface{}) (string, error) {
	url := fmt.Sprintf("https://api.marqo.ai/api/v2/indexes/%s", indexName)
	fmt.Printf("Creating index at URL: %s\n", url)

	// Convert schema to proper request format
	req, err := CreateIndexRequestFromSchema(schema)
	if err != nil {
		return "", fmt.Errorf("failed to create index request: %w", err)
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
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	fmt.Printf("Request body:\n%s\n", string(body))

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", c.apiKey)

	resp, err := c.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create index: status=%d body=%s",
			resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("Index creation initiated, waiting for index to be ready...\n")

	endpoint, err := c.waitForIndexReady(indexName, 16*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed waiting for index: %w", err)
	}

	return endpoint, nil
}

// Helper function to create a structured index request from schema
func CreateIndexRequestFromSchema(schemaMap map[string]interface{}) (*CreateIndexRequest, error) {
	data, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema map: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshall schema: %w", err)
	}

	// Validate the schema
	if err := schema.Validate(); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	return &CreateIndexRequest{
		Type:                schema.Type,
		VectorNumericType:   schema.VectorNumericType,
		Model:               schema.Model,
		NormalizeEmbeddings: schema.NormalizeEmbeddings,
		TextPreprocessing:   schema.TextPreprocessing,
		AllFields:           schema.AllFields,
		TensorFields:        schema.TensorFields,
		AnnParameters:       schema.AnnParameters,
		NumberOfShards:      1,
		NumberOfReplicas:    0,
		InferenceType:       "marqo.CPU.large",
		StorageClass:        "marqo.basic",
		NumberOfInferences:  1,
	}, nil
}

// Add these methods to MarqoClient

func (c *MarqoClient) Search(indexName string, query string, offset, limit int) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/indexes/%s/search", c.baseURL, indexName)
	fmt.Printf("URL: %v", url)

	requestBody := map[string]interface{}{
		"q":      "*", // use wildcard or empty query to retrieve all documents
		"limit":  limit,
		"offset": offset,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("Response body: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: status=%d body=%s",
			resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Hits []map[string]interface{} `json:"hits"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Hits, nil
}

func (c *MarqoClient) UpsertDocuments(indexName string, documents []map[string]interface{}) error {
	url := fmt.Sprintf("%s/indexes/%s/documents", c.baseURL, indexName)
	fmt.Printf("Upserting documents to URL: %s\n", url)

	requestBody := map[string]interface{}{
		"documents": documents,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal documents: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.addHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("Upsert response status: %d\n", resp.StatusCode)
	fmt.Printf("Upsert response body: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upsert documents: status=%d, documents request body=%s", resp.StatusCode, string(body))
	}

	// Parse response to verify success
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Add verification of the response content here
	if errors, ok := result["error"]; ok {
		if errorsList, ok := errors.([]interface{}); ok && len(errorsList) > 0 {
			return fmt.Errorf("upsert had errors: %v", errorsList)
		}
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

func (c *MarqoClient) addHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	// Add CSRF protection headers
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://api.marqo.ai")
}

func (c *MarqoClient) waitForIndexReady(indexName string, timeout time.Duration) (string, error) {
	start := time.Now()
	checkInterval := 15 * time.Second
	maxAttempts := int(timeout / checkInterval)
	attempt := 1

	fmt.Printf("\n=== Starting Index Ready Check ===\n")
	fmt.Printf("Index Name: %s\n", indexName)
	fmt.Printf("Timeout: %v\n", timeout)
	fmt.Printf("Check Interval: %v\n", checkInterval)
	fmt.Printf("Max Attempts: %d\n\n", maxAttempts)

	for {
		if time.Since(start) > timeout {
			return "", fmt.Errorf("timeout waiting for index %s to be ready after %v (made %d attempts)",
				indexName, timeout, attempt)
		}

		url := fmt.Sprintf("%s/indexes", c.baseURL)
		fmt.Printf("Attempt %d/%d: Checking URL: %s\n", attempt, maxAttempts, url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return "", err
		}

		c.addHeaders(req)
		// Debug headers
		fmt.Printf("Request Headers:\n")
		for key, values := range req.Header {
			fmt.Printf("  %s: %v\n", key, values)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			fmt.Printf("Attempt %d/%d: Error making request: %v\n", attempt, maxAttempts, err)
			time.Sleep(checkInterval)
			attempt++
			continue
		}

		// Debug response status
		fmt.Printf("Response Status: %s\n", resp.Status)

		var result ListIndexesResponse
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			resp.Body.Close()
			time.Sleep(checkInterval)
			attempt++
			continue
		}

		// Debug raw response
		fmt.Printf("Raw Response Body:\n%s\n", string(body))

		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Error unmarshaling response: %v\n", err)
			resp.Body.Close()
			time.Sleep(checkInterval)
			attempt++
			continue
		}
		resp.Body.Close()

		fmt.Printf("Found %d indexes in response\n", len(result.Results))

		indexFound := false
		for _, idx := range result.Results {
			fmt.Printf("Checking index: %s (Status: %s)\n", idx.IndexName, idx.IndexStatus)
			if idx.IndexName == indexName {
				indexFound = true
				fmt.Printf("Found target index!\n")
				fmt.Printf("Status: %s\n", idx.IndexStatus)
				fmt.Printf("Endpoint: %s\n", idx.MarqoEndpoint)

				switch idx.IndexStatus {
				case "READY":
					fmt.Printf("Index is READY! Returning endpoint: %s\n", idx.MarqoEndpoint)
					return idx.MarqoEndpoint, nil
				case "FAILED":
					return "", fmt.Errorf("index creation failed for %s", indexName)
				case "CREATING":
					fmt.Printf("Index still creating, will check again in %v\n", checkInterval)
					break
				default:
					fmt.Printf("Unknown index status: %s\n", idx.IndexStatus)
				}
				break
			}
		}

		if !indexFound {
			fmt.Printf("WARNING: Index %s not found in response!\n", indexName)
		}

		fmt.Printf("Attempt %d/%d: Index not ready yet, waiting %v...\n\n",
			attempt, maxAttempts, checkInterval)
		time.Sleep(checkInterval)
		attempt++
	}
}
