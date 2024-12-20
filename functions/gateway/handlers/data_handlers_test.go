package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	internal_types "github.com/meetnearme/api/functions/gateway/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

var searchUsersByIDs = helpers.SearchUsersByIDs

func init() {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
}

func TestPostEvent(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the authorization header
		authHeader := r.Header.Get("x-api-key")
		// we do nothing here because the underlying implementation of marqo go
		// library implements `WithMarqoCloudAuth` as an option expected in our
		// implementation, so omitting the auth header will result a lib failure
		if authHeader == "" {
			http.Error(w, "Unauthorized, missing x-api-key header", http.StatusUnauthorized)
			return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors:    false,
			IndexName: testMarqoIndexName,
			Items: []marqo.Item{
				{
					ID:     "123",
					Result: "",
					Status: 200,
				},
			},
			ProcessingTimeMS: 0.38569063499744516,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name                    string
		requestBody             string
		mockUpsertFunc          func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
		expectedStatus          int
		expectedBodyCheck       func(body string) error
		expectMissingAuthHeader bool
	}{
		{
			name:           "Valid event",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType": "` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusCreated,
			expectedBodyCheck: func(body string) error {
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(body), &response); err != nil {
					return fmt.Errorf("failed to unmarshal response body: %v", err)
				}
				t.Logf("<<< response: %v", response)
				items, ok := response["items"].([]interface{})
				if !ok || len(items) == 0 {
					return fmt.Errorf("expected non-empty Items array, got '%v'", items)
				}

				firstItem, ok := items[0].(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected first item to be a map, got '%v'", items[0])
				}

				id, ok := firstItem["_id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("expected non-empty ID, got '%v'", id)
				}

				if id != "123" {
					return fmt.Errorf("expected id to be %v, got %v", "123", id)
				}

				return nil
			},
		},
		{
			name:                    "Valid payload, missing auth header",
			expectMissingAuthHeader: true,
			requestBody:             `{ "eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
				res, err := services.BulkUpsertEventToMarqo(client, events, false)
				if err != nil {
					log.Printf("mocked request to upsert event failed: %v", err)
				}
				return res, nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBodyCheck: func(body string) error {
				if strings.Contains(body, "ERR: Failed to upsert event") {
					return nil
				}
				return fmt.Errorf("Expected error message, but none present")
			},
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"name":"Test Event","description":}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(strings.ToLower(body), "failed to extract event from payload: invalid json payload") {
					return fmt.Errorf("expected 'failed to extract event from payload: invalid json payload', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing start time field",
			requestBody:    `{"description":"A test event", "eventOwnerName": "Event Owner",  "eventOwners":["123"],"name":"Test Event","eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "Field validation for 'StartTime' failed on the 'required' tag") {
					return fmt.Errorf("expected 'Field validation for 'StartTime' failed on the 'required' tag', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing name field",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					return fmt.Errorf(`expected "Field validation for 'Name' failed on the 'required' tag", got '%s'`, body)
				}
				return nil
			},
		},
		{
			name:           "Missing eventOwners field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'EventOwners' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'EventOwners' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing eventOwnerName field",
			requestBody:    `{"eventOwners":["123"], "name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'EventOwnerName' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'EventOwnerName' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing timezone field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventOwners":["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'Timezone' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'Timezone' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Invalid timezone field",
			requestBody:    `{"timezone":"Does_Not_Exist/Nowhere","eventOwnerName":"Event Owner","eventOwners":["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `invalid timezone: unknown time zone Does_Not_Exist/Nowhere`) {
					return fmt.Errorf("expected `invalid timezone: unknown time zone Does_Not_Exist/Nowhere`, got '%s'", body)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectMissingAuthHeader {
				originalApiKey := os.Getenv("MARQO_API_KEY")
				os.Setenv("MARQO_API_KEY", "")
				defer os.Setenv("MARQO_API_KEY", originalApiKey)
			}

			marqoClient, err := services.GetMarqoClient()
			if err != nil {
				log.Println("failed to get marqo client")
			}

			mockService := &services.MockMarqoService{
				UpsertEventToMarqoFunc: func(client *marqo.Client, event types.Event) (*marqo.UpsertDocumentsResponse, error) {
					events := []types.Event{event}
					return tt.mockUpsertFunc(marqoClient, events)

				},
			}

			req, err := http.NewRequestWithContext(context.Background(), "PUT", "/events/", bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			rr := httptest.NewRecorder()
			handler := NewMarqoHandler(mockService)

			handler.PostEvent(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if err := tt.expectedBodyCheck(rr.Body.String()); err != nil {
				t.Errorf("Body check failed: %v", err)
			}
		})
	}
}

func TestPostBatchEvents(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)

	// set below in response to binding
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the authorization header
		authHeader := r.Header.Get("x-api-key")
		// we do nothing here because the underlying implementation of marqo go
		// library implements `WithMarqoCloudAuth` as an option expected in our
		// implementation, so omitting the auth header will result a lib failure

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading body in mock: %v", err)
		}

		var createEvent map[string]interface{}
		err = json.Unmarshal(body, &createEvent)
		if err != nil {
			log.Printf("error unmarshaling body in mock: %v", err)
		}

		if authHeader == "" {
			http.Error(w, "Unauthorized, missing x-api-key header", http.StatusUnauthorized)
			return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors:    false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID:     "123",
					Result: "",
					Status: 200,
				},
				{
					ID:     "456",
					Result: "",
					Status: 200,
				},
			},
			ProcessingTimeMS: 0.38569063499744516,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name                    string
		requestBody             string
		mockUpsertFunc          func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
		expectedStatus          int
		expectedBodyCheck       func(body string) error
		expectMissingAuthHeader bool
	}{
		{
			name:        "Valid events",
			requestBody: `{"events":[ {"eventOwnerName": "Event Owner 1", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}, { "eventOwnerName": "Event Owner 2", "eventOwners":["456"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Another Test Event","description":"Another test event","startTime":"2099-05-02T12:00:00Z","address":"456 Test St","lat":51.5075,"long":-0.1279,"timezone":"America/New_York"}]}`,
			mockUpsertFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
				res, err := services.BulkUpsertEventToMarqo(client, events, false)
				if err != nil {
					log.Printf("mocked request to upsert events failed: %v", err)
				}
				return &marqo.UpsertDocumentsResponse{}, fmt.Errorf("mocked request to upsert events res: %v", res)
			},
			expectedStatus: http.StatusCreated,
			expectedBodyCheck: func(body string) error {
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(body), &response); err != nil {
					return fmt.Errorf("failed to unmarshal response body: %v", err)
				}
				items, ok := response["items"].([]interface{})
				if !ok || len(items) == 0 {
					return fmt.Errorf("expected non-empty Items array, got '%v'", items)
				}

				firstItem, ok := items[0].(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected first item to be a map, got '%v'", items[0])
				}

				id, ok := firstItem["_id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("expected non-empty ID, got '%v'", id)
				}

				if id != "123" {
					return fmt.Errorf("expected id to be %v, got %v", "123", id)
				}

				return nil
			},
		},
		{
			name:                    "Valid payload, missing auth header",
			expectMissingAuthHeader: true,
			requestBody:             `{"events":[ {"eventOwnerName": "Event Owner 1", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"},{ "eventOwnerName": "Event Owner 2", "eventOwners":["456"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Another Test Event","description":"Another test event","startTime":"2099-05-02T12:00:00Z","address":"456 Test St","lat":51.5075,"long":-0.1279,"timezone":"America/New_York"}]}`,
			mockUpsertFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
				res, err := services.BulkUpsertEventToMarqo(client, events, false)
				if err != nil {
					log.Printf("mocked request to upsert events failed: %v", err)
				}
				return res, nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBodyCheck: func(body string) error {
				if strings.Contains(body, "ERR: Failed to upsert events") {
					return nil
				}
				return fmt.Errorf("Expected error message, but none present")
			},
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"events":[{"name":"Test Event","description":}]}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "invalid JSON payload") {
					return fmt.Errorf("expected 'invalid JSON payload', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing required field",
			requestBody:    `{"events":[{"description":"A test event"}]}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "invalid event at index 0: Field validation for 'EventOwners' failed on the 'required' tag") {
					return fmt.Errorf("expected 'invalid event at index 0: Field validation for 'EventOwners' failed on the 'required' tag, got '%s'", body)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectMissingAuthHeader {
				originalApiKey := os.Getenv("MARQO_API_KEY")
				os.Setenv("MARQO_API_KEY", "")
				defer os.Setenv("MARQO_API_KEY", originalApiKey)
			}

			marqoClient, err := services.GetMarqoClient()
			if err != nil {
				log.Println("failed to get marqo client")
			}

			mockService := &services.MockMarqoService{
				BulkUpsertEventToMarqoFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
					return tt.mockUpsertFunc(marqoClient, events)
				},
			}

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/events", bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			rr := httptest.NewRecorder()
			handler := NewMarqoHandler(mockService)

			handler.PostBatchEvents(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			err = tt.expectedBodyCheck(rr.Body.String())
			if err != nil {
				t.Errorf("Body check failed: %v", err)
			}
		})
	}
}

func TestSearchEvents(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// if strings.HasPrefix(r.URL.Path, "/indexes/events-search-index/search") {

		// Handle search request
		query := r.URL.Query().Get("q")
		// URL decode the query
		decodedQuery, err := url.QueryUnescape(query)
		if err != nil {
			http.Error(w, "Failed to decode query", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"Hits": []map[string]interface{}{
				{
					"_id":            "123",
					"eventOwnerName": "Event Owner 1",
					"eventOwners":    []interface{}{"789"},
					"name":           "First Test Event",
					"description":    "Description of the first event",
				},
				{
					"_id":            "456",
					"eventOwnerName": "Event Owner 2",
					"eventOwners":    []interface{}{"012"},
					"name":           "Second Test Event",
					"description":    "Description of the second event",
				},
			},
			"Query": decodedQuery,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
		// } else {
		// 	http.Error(w, "Not found", http.StatusNotFound)
		// }
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name           string
		path           string
		expectQuery    bool
		expectedStatus int
		expectedCheck  func(t *testing.T, body []byte)
	}{
		{
			name:           "Search events",
			path:           "/events?q=test+search",
			expectQuery:    true,
			expectedStatus: http.StatusOK,
			expectedCheck: func(t *testing.T, body []byte) {
				var res types.EventSearchResponse
				err := json.Unmarshal(body, &res)
				if err != nil {
					t.Errorf("error marshaling search response to JSON, %v", err)
				}
				events := res.Events
				if len(events) != 2 {
					t.Errorf("Expected 2 events, got %d", len(events))
				}
				if events[0].Id != "123" {
					t.Errorf("Expected first event to have Id 123, got %v", events[0].Id)
				}
				if events[1].Id != "456" {
					t.Errorf("Expected first event to have Id 456, got %v", events[1].Id)
				}

				if res.Query != "keywords: { test search }" {
					t.Errorf("Expected query to be 'keywords: { test search }', got %v", res.Query)
				}

			},
		},
		// {
		// 	name:           "Empty search query",
		// 	path:           "/events?q=",
		//     expectQuery:    true,
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedCheck:  nil,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err != nil {
				t.Fatalf("Failed to get Marqo client: %v", err)
			}

			mockService := &services.MockMarqoService{
				SearchEventsFunc: func(client *marqo.Client, query string, userLocation []float64, maxDistance float64, startTime int64, endTime int64, ownerIds []string) (types.EventSearchResponse, error) {
					return services.SearchMarqoEvents(client, query, userLocation, maxDistance, startTime, endTime, ownerIds, string(""), string(""), "0", []string{helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[0]}, []string{})
				},
			}

			req, err := http.NewRequestWithContext(context.Background(), "GET", tt.path, nil)
			if err != nil {
				t.Errorf("error making mocked request to search: %v", err)
			}
			rr := httptest.NewRecorder()
			handler := NewMarqoHandler(mockService)

			handler.SearchEvents(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedCheck != nil {
				tt.expectedCheck(t, rr.Body.Bytes())
			}
		})
	}
}

func TestBulkUpdateEvents(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the authorization header
		authHeader := r.Header.Get("x-api-key")
		// we do nothing here because the underlying implementation of marqo go
		// library implements `WithMarqoCloudAuth` as an option expected in our
		// implementation, so omitting the auth header will result a lib failure

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading body in mock: %v", err)
		}

		var createEvent map[string]interface{}
		err = json.Unmarshal(body, &createEvent)
		if err != nil {
			log.Printf("error unmarshaling body in mock: %v", err)
		}

		if authHeader == "" {
			http.Error(w, "Unauthorized, missing x-api-key header", http.StatusUnauthorized)
			return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors:    false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID:     "123",
					Result: "",
					Status: 200,
				},
				{
					ID:     "456",
					Result: "",
					Status: 200,
				},
			},
			ProcessingTimeMS: 0.38569063499744516,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name                    string
		payload                 string
		expectedStatus          int
		expectedBody            string
		expectMissingAuthHeader bool
		mockUpsertFunc          func(client *marqo.Client, event types.Event) (*marqo.UpsertDocumentsResponse, error)
	}{
		{
			name:                    "Invalid payload (missing ID in one event)",
			payload:                 `{"events":[{"id":"abc", "eventOwnerName": "Event Owner 1", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364,"timezone":"America/New_York"}, {"eventOwnerName": "Event Owner 2", "eventOwners":["456"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544,"timezone":"America/New_York"}]}`,
			expectedStatus:          http.StatusBadRequest,
			expectedBody:            `invalid event at index 1: event has no id`,
			expectMissingAuthHeader: false,
			mockUpsertFunc:          nil,
		},
		{
			name:                    "Valid payload, missing auth header",
			payload:                 `{"events":[{"id":"abc", "eventOwnerName": "Event Owner 1", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364,"timezone":"America/New_York"},{"id":"xyz", "eventOwnerName": "Event Owner 2", "eventOwners":["456"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544,"timezone":"America/New_York"}]}`,
			expectedStatus:          http.StatusInternalServerError,
			expectedBody:            `Failed to upsert event: error upserting documents: status code: 401`,
			expectMissingAuthHeader: true,
			mockUpsertFunc:          nil,
		},
		{
			name:                    "Valid payload",
			payload:                 `{"events":[{"id":"abc", "eventOwnerName": "Event Owner 1", "eventOwners":["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364,"timezone":"America/New_York"},{"id":"xyz", "eventOwnerName": "Event Owner 2", "eventOwners":["456"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544,"timezone":"America/New_York"}]}`,
			expectedStatus:          http.StatusOK,
			expectedBody:            `"errors":false`,
			expectMissingAuthHeader: false,
			mockUpsertFunc:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectMissingAuthHeader {
				originalApiKey := os.Getenv("MARQO_API_KEY")
				os.Setenv("MARQO_API_KEY", "")
				defer os.Setenv("MARQO_API_KEY", originalApiKey)
			}

			marqoClient, err := services.GetMarqoClient()
			if err != nil {
				log.Println("failed to get marqo client")
			}

			mockService := &services.MockMarqoService{
				UpsertEventToMarqoFunc: func(client *marqo.Client, event types.Event) (*marqo.UpsertDocumentsResponse, error) {
					return tt.mockUpsertFunc(marqoClient, event)
				},
			}

			req, err := http.NewRequestWithContext(context.Background(), "PUT", "/api/events", strings.NewReader(tt.payload))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := NewMarqoHandler(mockService)

			handler.BulkUpdateEvents(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if !strings.Contains(strings.ToLower(rr.Body.String()), strings.ToLower(tt.expectedBody)) {
				t.Errorf("Handler returned unexpected body: got: %v want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestUpdateOneEvent(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the authorization header
		authHeader := r.Header.Get("x-api-key")
		// we do nothing here because the underlying implementation of marqo go
		// library implements `WithMarqoCloudAuth` as an option expected in our
		// implementation, so omitting the auth header will result a lib failure
		if authHeader == "" {
			http.Error(w, "Unauthorized, missing x-api-key header", http.StatusUnauthorized)
			return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors:    false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID:     "123",
					Result: "",
					Status: 200,
				},
			},
			ProcessingTimeMS: 0.38569063499744516,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name                    string
		apiPath                 string
		requestBody             string
		mockUpsertFunc          func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
		expectedStatus          int
		expectedBodyCheck       func(body string) error
		expectMissingAuthHeader bool
	}{
		{
			name:           "Valid event",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789", "eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(body string) error {
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(body), &response); err != nil {
					return fmt.Errorf("failed to unmarshal response body: %v", err)
				}
				items, ok := response["items"].([]interface{})
				if !ok || len(items) == 0 {
					return fmt.Errorf("expected non-empty Items array, got '%v'", items)
				}

				firstItem, ok := items[0].(map[string]interface{})
				if !ok {
					return fmt.Errorf("expected first item to be a map, got '%v'", items[0])
				}

				id, ok := firstItem["_id"].(string)
				if !ok || id == "" {
					return fmt.Errorf("expected non-empty ID, got '%v'", id)
				}

				if id != "123" {
					return fmt.Errorf("expected id to be %v, got %v", "123", id)
				}

				return nil
			},
		},
		{
			name:                    "Valid payload, missing event path parameter",
			apiPath:                 `/`,
			expectMissingAuthHeader: true,
			requestBody:             `{ "id":"abc-789", "eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc:          nil,
			expectedStatus:          http.StatusInternalServerError,
			expectedBodyCheck: func(body string) error {
				if strings.Contains(body, "ERR: Event must have an id") {
					return nil
				}
				return fmt.Errorf("Expected error message, but none present")
			},
		},
		{
			name:                    "Valid payload, missing auth header",
			apiPath:                 `/test-id`,
			expectMissingAuthHeader: true,
			requestBody:             `{ "id":"abc-789", "eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc:          nil,
			expectedStatus:          http.StatusInternalServerError,
			expectedBodyCheck: func(body string) error {
				if strings.Contains(body, "ERR: Failed to upsert event") {
					return nil
				}
				return fmt.Errorf("Expected error message, but none present")
			},
		},
		{
			name:           "Invalid JSON",
			apiPath:        `/test-id`,
			requestBody:    `{"name":"Test Event","description":}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(strings.ToLower(body), "failed to extract event from payload: invalid json payload") {
					return fmt.Errorf("expected 'failed to extract event from payload: invalid json payload', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing id in api path",
			apiPath:        ``,
			requestBody:    `{"eventOwnerName": "Event Owner",  "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event", "startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusInternalServerError,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "Event must have an id") {
					return fmt.Errorf("expected 'Event must have an id', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing start time field",
			apiPath:        `/test-id`,
			requestBody:    `{"id":"abc-789","eventOwnerName": "Event Owner",  "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event", "address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "invalid body: Field validation for 'StartTime' failed on the 'required' tag") {
					return fmt.Errorf("expected 'invalid body: Field validation for 'StartTime' failed on the 'required' tag', got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing name field",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789","eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					return fmt.Errorf(`expected "Field validation for 'Name' failed on the 'required' tag", got '%s'`, body)
				}
				return nil
			},
		},
		{
			name:           "Missing eventOwners field",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789","eventOwnerName": "Event Owner","eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'EventOwners' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'EventOwners' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing eventOwnerName field",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789", "eventOwners": ["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'EventOwnerName' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'EventOwnerName' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing timezone field",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789", "eventOwners": ["123"], "eventOwnerName":"Event Owner","eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'Timezone' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'Timezone' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Missing eventSourceType field",
			apiPath:        `/abc-789`,
			requestBody:    `{ "id":"abc-789", "eventOwners": ["123"],"eventOwnerName":"Event Owner","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, `Field validation for 'EventSourceType' failed on the 'required' tag`) {
					return fmt.Errorf("expected `Field validation for 'EventSourceType' failed on the 'required' tag`, got '%s'", body)
				}
				return nil
			},
		},
		{
			name:           "Invalid eventSourceType field",
			apiPath:        `/test-id`,
			requestBody:    `{ "id":"abc-789", "eventOwners": ["123"],"eventOwnerName":"Event Owner","eventSourceType":"NONEXISTENT","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			mockUpsertFunc: nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(body string) error {
				if !strings.Contains(body, "invalid body: invalid eventSourceType: NONEXISTENT") {
					return fmt.Errorf("expected 'invalid body: invalid eventSourceType: NONEXISTENT', got '%s'", body)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectMissingAuthHeader {
				originalApiKey := os.Getenv("MARQO_API_KEY")
				os.Setenv("MARQO_API_KEY", "")
				defer os.Setenv("MARQO_API_KEY", originalApiKey)
			}

			marqoClient, err := services.GetMarqoClient()
			if err != nil {
				log.Println("failed to get marqo client")
			}

			mockService := &services.MockMarqoService{
				UpsertEventToMarqoFunc: func(client *marqo.Client, event types.Event) (*marqo.UpsertDocumentsResponse, error) {
					return tt.mockUpsertFunc(marqoClient, []types.Event{event})
				},
			}

			// In TestUpdateOneEvent, modify the request creation:
			req, err := http.NewRequestWithContext(context.Background(), "PUT", "/events/"+tt.apiPath, bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Add mux vars to request context
			vars := map[string]string{
				helpers.EVENT_ID_KEY: strings.TrimPrefix(tt.apiPath, "/"),
			}
			req = mux.SetURLVars(req, vars)

			rr := httptest.NewRecorder()
			handler := NewMarqoHandler(mockService)

			handler.UpdateOneEvent(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
			log.Printf(`rr.Body.String() %+v`, rr.Body.String())
			if err := tt.expectedBodyCheck(rr.Body.String()); err != nil {
				t.Errorf("Body check failed: %v", err)
			}
		})
	}
}

func TestHandleCheckoutWebhook(t *testing.T) {
	t.Run("handles checkout.session.completed successfully", func(t *testing.T) {
		// Save original env var
		originalWebhookSecret := os.Getenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET")
		testWebhookSecret := "whsec_test_secret"
		os.Setenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET", testWebhookSecret)
		// Restore original env var after test
		defer os.Setenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET", originalWebhookSecret)

		// Setup mock service first
		mockPurchasesService := &dynamodb_service.MockPurchaseService{
			GetPurchaseByPkFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error) {
				return &internal_types.Purchase{
					EventID:         eventId,
					UserID:          userId,
					CreatedAtString: createdAt,
					Status:          helpers.PurchaseStatus.Pending,
					PurchasedItems: []internal_types.PurchasedItem{
						{
							Name:     "Test Item",
							Quantity: 1,
							Cost:     1000,
						},
					},
				}, nil
			},
			UpdatePurchaseFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string, update internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
				if update.Status != helpers.PurchaseStatus.Settled {
					t.Errorf("expected status %v, got %v", helpers.PurchaseStatus.Settled, update.Status)
				}
				return nil, nil
			},
		}
		// Create handler with mock service
		handler := NewPurchasableWebhookHandler(dynamodb_service.NewPurchasableService(), mockPurchasesService)

		// Setup request data
		now := time.Now()
		nowString := fmt.Sprintf("%020d", now.Unix())
		eventID := "test-event-123"
		userID := "test-user-456"
		clientReferenceID := "event-" + eventID + "-user-" + userID + "-time-" + nowString
		payload := []byte(`{
			"type": "checkout.session.completed",
			"api_version": "2024-09-30.acacia",
			"data": {
				"object": {
					"client_reference_id": "` + clientReferenceID + `",
					"status": "complete"
				}
			}
		}`)
		// Generate signed payload
		timestamp := now.Unix()
		mac := hmac.New(sha256.New, []byte(testWebhookSecret))
		mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
		mac.Write([]byte("."))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		stripeSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)

		r := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(payload))
		ctx := context.WithValue(r.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
			Headers: map[string]string{
				"stripe-signature": stripeSignature,
			},
		})
		r = r.WithContext(ctx)

		w := httptest.NewRecorder()
		// Execute handler
		handler.HandleCheckoutWebhook(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
		}
	})
	t.Run("handles checkout.session.expired successfully", func(t *testing.T) {
		// Save original env var
		originalWebhookSecret := os.Getenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET")
		testWebhookSecret := "whsec_test_secret"
		os.Setenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET", testWebhookSecret)
		// Restore original env var after test
		defer os.Setenv("DEV_STRIPE_CHECKOUT_WEBHOOK_SECRET", originalWebhookSecret)

		tests := []struct {
			name             string
			inventory        int32
			startingQuantity int32
			purchaseQuantity int32
			expectedQuantity int32 // The quantity we expect to be set after the update
		}{
			{
				name:             "Basic inventory restoration",
				inventory:        9,
				startingQuantity: 10,
				purchaseQuantity: 1,
				expectedQuantity: 10,
			},
			{
				name:             "Multiple items purchased",
				inventory:        7,
				startingQuantity: 10,
				purchaseQuantity: 3,
				expectedQuantity: 10,
			},
			{
				name:             "Full inventory restoration",
				inventory:        0,
				startingQuantity: 100,
				purchaseQuantity: 100,
				expectedQuantity: 100,
			},
			{
				name:             "Inventory does not exceed StartingQuantity",
				inventory:        95,
				startingQuantity: 100,
				purchaseQuantity: 10,
				expectedQuantity: 100,
			},
			{
				name:             "Partial purchase cancellation",
				inventory:        95,
				startingQuantity: 100,
				purchaseQuantity: 5,
				expectedQuantity: 100,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Setup test data
				now := time.Now()
				eventID := "test-event-123"
				userID := "test-user-456"
				nowString := fmt.Sprintf("%020d", now.Unix())
				clientReferenceID := "event-" + eventID + "-user-" + userID + "-time-" + nowString

				// Create payload
				payload := []byte(`{
					"type": "checkout.session.expired",
					"api_version": "2024-09-30.acacia",
					"data": {
						"object": {
							"client_reference_id": "` + clientReferenceID + `",
							"status": "expired"
						}
					}
				}`)

				// Generate signed payload
				timestamp := now.Unix()
				mac := hmac.New(sha256.New, []byte(testWebhookSecret))
				mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
				mac.Write([]byte("."))
				mac.Write(payload)
				signature := hex.EncodeToString(mac.Sum(nil))
				stripeSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)

				// Create request
				r := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(payload))
				ctx := context.WithValue(r.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
					Headers: map[string]string{
						"stripe-signature": stripeSignature,
					},
				})
				r = r.WithContext(ctx)
				w := httptest.NewRecorder()

				mockPurchasableService := &dynamodb_service.MockPurchasableService{
					GetPurchasablesByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.Purchasable, error) {
						return &internal_types.Purchasable{
							EventId: eventId,
							PurchasableItems: []internal_types.PurchasableItemInsert{
								{
									Name:             "Test Item",
									Inventory:        tt.inventory,
									Cost:             1000,
									StartingQuantity: tt.startingQuantity,
								},
							},
						}, nil
					},
					UpdatePurchasableInventoryFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, updates []internal_types.PurchasableInventoryUpdate, purchasableMap map[string]internal_types.PurchasableItemInsert) error {
						if len(updates) != 1 {
							t.Errorf("expected 1 update, got %d", len(updates))
						}
						if updates[0].Name != "Test Item" {
							t.Errorf("expected item name %v, got %v", "Test Item", updates[0].Name)
						}
						if updates[0].Quantity != tt.expectedQuantity {
							t.Errorf("expected quantity %v, got %v", tt.expectedQuantity, updates[0].Quantity)
						}
						return nil
					},
				}

				mockPurchaseService := &dynamodb_service.MockPurchaseService{
					GetPurchaseByPkFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error) {
						return &internal_types.Purchase{
							EventID:         eventId,
							UserID:          userId,
							CreatedAtString: createdAt,
							Status:          helpers.PurchaseStatus.Pending,
							PurchasedItems: []internal_types.PurchasedItem{
								{
									Name:     "Test Item",
									Quantity: tt.purchaseQuantity,
									Cost:     1000,
								},
							},
						}, nil
					},
					UpdatePurchaseFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string, update internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
						if update.Status != helpers.PurchaseStatus.Canceled {
							t.Errorf("expected status %v, got %v", helpers.PurchaseStatus.Canceled, update.Status)
						}
						return nil, nil
					},
				}

				handler := NewPurchasableWebhookHandler(mockPurchasableService, mockPurchaseService)

				err := handler.HandleCheckoutWebhook(w, r)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if w.Code != http.StatusOK {
					t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
				}
			})
		}
	})
	t.Run("handles invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer([]byte(`{}`)))
		ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
			Headers: map[string]string{
				"stripe-signature": "invalid_signature",
			},
		})
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		handler := NewPurchasableWebhookHandler(&dynamodb_service.MockPurchasableService{}, &dynamodb_service.MockPurchaseService{})
		err := handler.HandleCheckoutWebhook(w, req)
		if err == nil {
			t.Error("expected error, got nil")
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status code %v, got %v", http.StatusBadRequest, w.Code)
		}
	})
}

func TestGetUsersHandler(t *testing.T) {
	helpers.InitDefaultProtocol()
	// Save original environment variables
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")

	// Set test environment variables

	os.Setenv("ZITADEL_INSTANCE_HOST", helpers.MOCK_ZITADEL_HOST)
	// Defer resetting environment variables
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create a mock HTTP server for Zitadel
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/v2/users") {
			w.Header().Set("Content-Type", "application/json")

			// Parse the request body to get the userIds from the query
			var requestBody struct {
				Queries []struct {
					InUserIdsQuery struct {
						UserIds []string `json:"userIds"`
					} `json:"inUserIdsQuery"`
				} `json:"queries"`
			}

			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Get the userIds from the request
			var userIds []string
			if len(requestBody.Queries) > 0 {
				userIds = requestBody.Queries[0].InUserIdsQuery.UserIds
			}

			var response helpers.ZitadelUserSearchResponse
			response.Details.TotalResult = "1"
			response.Details.Timestamp = "2099-01-01T00:00:00Z"

			switch {
			case len(userIds) == 1 && userIds[0] == "123456789012345678":
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{
					{
						UserID:   "123456789012345678",
						Username: "testuser",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User",
							},
							Email: map[string]interface{}{},
						},
					},
				}
			case len(userIds) == 2:
				response.Details.TotalResult = "2"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{
					{
						UserID:   "123456789012345678",
						Username: "testuser1",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User 1",
							},
							Email: map[string]interface{}{},
						},
					},
					{
						UserID:   "987654321098765432",
						Username: "testuser2",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User 2",
							},
							Email: map[string]interface{}{},
						},
					},
				}
			case len(userIds) == 1 && userIds[0] == "nonexistent":
				response.Details.TotalResult = "0"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{}
			default:
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}

			responseJSON, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Write(responseJSON)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set the mock Zitadel server URL
	mockZitadelServer.Listener.Close()
	mockZitadelServer.Listener, err = net.Listen("tcp", helpers.MOCK_ZITADEL_HOST)
	if err != nil {
		t.Fatalf("Failed to start mock Zitadel server: %v", err)
	}
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Store the original SearchUsersByIDs function
	originalSearchFunc := searchUsersByIDs

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Missing ids parameter",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ERR: Missing required 'ids' parameter",
		},
		{
			name:           "Invalid ID length",
			queryParams:    "?ids=12345", // Less than 18 characters
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ERR: Invalid ID length: 12345. Must be exactly 18 characters",
		},
		{
			name:           "Invalid ID format (non-numeric)",
			queryParams:    "?ids=12345678901234567a", // Contains letter
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ERR: Invalid ID format: 12345678901234567a. Must contain only numbers",
		},
		{
			name:           "Valid single ID",
			queryParams:    "?ids=123456789012345678",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"userId":"123456789012345678","displayName":"Test User"}]`,
		},
		{
			name:           "Valid multiple IDs",
			queryParams:    "?ids=123456789012345678,987654321098765432",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"userId":"123456789012345678","displayName":"Test User 1"},{"userId":"987654321098765432","displayName":"Test User 2"}]`,
		},
		{
			name:           "Search returns no results",
			queryParams:    "?ids=nonexistent",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Invalid ID length: nonexistent. Must be exactly 18 characters`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Restore the original function after the test
			defer func() {
				searchUsersByIDs = originalSearchFunc
			}()

			// Create request with test query parameters
			req := httptest.NewRequest(http.MethodGet, "/users"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Call the handler
			handler := GetUsersHandler(w, req)
			handler.ServeHTTP(w, req)

			log.Printf("\n\n\n\nw.Body: %v", w.Body)
			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			gotBody := strings.TrimSpace(w.Body.String())
			if tt.expectedStatus == http.StatusOK {
				// For JSON responses, compare after normalizing
				var got, expected interface{}
				if err := json.Unmarshal([]byte(gotBody), &got); err != nil {
					t.Fatalf("failed to unmarshal response body: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.expectedBody), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected body: %v", err)
				}
				// if !reflect.DeepEqual(got, expected) {
				if !strings.Contains(gotBody, tt.expectedBody) {
					t.Errorf("expected body %v, got %v", expected, got)
				}
			} else {
				// For error responses, compare strings directly
				// if gotBody != tt.expectedBody {
				if !strings.Contains(gotBody, tt.expectedBody) {
					t.Errorf("expected body %q, got %q", tt.expectedBody, gotBody)
				}
			}

			// Check Content-Type header for successful JSON responses
			if tt.expectedStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type 'application/json', got %q", contentType)
				}
			}
		})
	}
}

func TestSearchUsersHandler(t *testing.T) {
	helpers.InitDefaultProtocol()
	// Save original environment variables
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")

	// Set test environment variables
	os.Setenv("ZITADEL_INSTANCE_HOST", helpers.MOCK_ZITADEL_HOST)
	// Defer resetting environment variables
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create a mock HTTP server for Zitadel
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/v2/users") {
			w.Header().Set("Content-Type", "application/json")

			// Parse the request body
			var requestBody struct {
				Query struct {
					Offset int  `json:"offset"`
					Limit  int  `json:"limit"`
					Asc    bool `json:"asc"`
				} `json:"query"`
				SortingColumn string `json:"sortingColumn"`
				Queries       []struct {
					TypeQuery *struct {
						Type string `json:"type"`
					} `json:"typeQuery,omitempty"`
					OrQuery *struct {
						Queries []struct {
							EmailQuery *struct {
								EmailAddress string `json:"emailAddress"`
								Method       string `json:"method"`
							} `json:"emailQuery,omitempty"`
							UserNameQuery *struct {
								UserName string `json:"userName"`
								Method   string `json:"method"`
							} `json:"userNameQuery,omitempty"`
						} `json:"queries"`
					} `json:"orQuery,omitempty"`
				} `json:"queries"`
			}

			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				log.Printf("Error decoding request body: %v", err)
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Extract search query from the OR query (either email or username)
			searchQuery := ""
			for _, query := range requestBody.Queries {
				if query.OrQuery != nil {
					for _, orQuery := range query.OrQuery.Queries {
						if orQuery.EmailQuery != nil {
							searchQuery = orQuery.EmailQuery.EmailAddress
							break
						}
						if orQuery.UserNameQuery != nil {
							searchQuery = orQuery.UserNameQuery.UserName
							break
						}
					}
				}
			}

			log.Printf("Extracted search query: %s", searchQuery)

			var response helpers.ZitadelUserSearchResponse
			response.Details.Timestamp = "2099-01-01T00:00:00Z"

			switch searchQuery {
			case "test":
				response.Details.TotalResult = "2"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{
					{
						UserID:   "123456789012345678",
						Username: "testuser1",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User 1",
							},
						},
					},
					{
						UserID:   "987654321098765432",
						Username: "testuser2",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User 2",
							},
						},
					},
				}
			case "nonexistent":
				response.Details.TotalResult = "0"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{}
			case "error":
				http.Error(w, "", http.StatusInternalServerError)
				return
			default:
				response.Details.TotalResult = "1"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{
					{
						UserID:   "123456789012345678",
						Username: "defaultuser",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Default User",
							},
							Email: map[string]interface{}{},
						},
					},
				}
			}

			responseJSON, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Write(responseJSON)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set the mock Zitadel server URL
	mockZitadelServer.Listener.Close()
	mockZitadelServer.Listener, err = net.Listen("tcp", helpers.MOCK_ZITADEL_HOST)
	if err != nil {
		t.Fatalf("Failed to start mock Zitadel server: %v", err)
	}
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Search with multiple results",
			queryParams:    "?q=test",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"userId":"123456789012345678","displayName":"Test User 1"},{"userId":"987654321098765432","displayName":"Test User 2"}]`,
		},
		{
			name:           "Search with no results",
			queryParams:    "?q=nonexistent",
			expectedStatus: http.StatusOK,
			expectedBody:   `[]`,
		},
		{
			name:           "Search with error",
			queryParams:    "?q=error",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "ERR: Failed to search users: failed to unmarshal response: unexpected end of JSON input",
		},
		{
			name:           "Search with default result",
			queryParams:    "?q=default",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"userId":"123456789012345678","displayName":"Default User"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users/search"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Call the handler
			handler := SearchUsersHandler(w, req)
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status code %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body
			gotBody := strings.TrimSpace(w.Body.String())
			if tt.expectedStatus == http.StatusOK {
				// For JSON responses, compare after normalizing
				var got, expected interface{}
				if err := json.Unmarshal([]byte(gotBody), &got); err != nil {
					t.Fatalf("failed to unmarshal response body: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.expectedBody), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected body: %v", err)
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("expected body %v, got %v", expected, got)
				}
			} else {
				// For error responses, compare strings directly
				if !strings.Contains(gotBody, tt.expectedBody) {
					t.Errorf("expected body %q, got %q", tt.expectedBody, gotBody)
				}
			}

			// Check Content-Type header for successful JSON responses
			if tt.expectedStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type 'application/json', got %q", contentType)
				}
			}
		})
	}
}
