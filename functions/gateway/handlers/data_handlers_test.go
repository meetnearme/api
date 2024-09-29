package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

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
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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
            return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors: false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID: "123",
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
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
        t.Log("Started mock Marqo server")
    }
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

    tests := []struct {
        name string
        requestBody string
        mockUpsertFunc    func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
        expectMissingAuthHeader bool
    }{
        {
            name:        "Valid event",
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
                res, err := services.BulkUpsertEventToMarqo(client, events, false)
                if err != nil {
                    log.Printf("mocked request to upsert event failed: %v", err)
                }
                return &marqo.UpsertDocumentsResponse{}, fmt.Errorf("mocked request to upsert event res: %v", res)
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
            name:        "Valid payload, missing auth header",
            expectMissingAuthHeader: true,
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
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
            requestBody:    `{"description":"A test event", "eventOwners":["123"],"lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "startTime is required") {
                    return fmt.Errorf("expected 'startTime is required', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:           "Missing name field",
            requestBody:    `{"eventOwners":["123"],"startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278}`,
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
            name:        "Missing eventOwners field",
            requestBody: `{"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, `Field validation for 'EventOwners' failed on the 'required' tag`) {
                    return fmt.Errorf("expected `Field validation for 'EventOwners' failed on the 'required' tag`, got '%s'", body)
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

            req, err := http.NewRequestWithContext(context.Background(), "POST", "/event", bytes.NewBufferString(tt.requestBody))
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
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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
			Errors: false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID: "123",
					Result: "",
					Status: 200,
				},
				{
					ID: "456",
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
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
        t.Log("Started mock Marqo server")
    }
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

    tests := []struct {
        name string
        requestBody string
        mockUpsertFunc    func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
        expectMissingAuthHeader bool
    }{
        {
            name:        "Valid events",
            requestBody: `{"events":[{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278},{"eventOwners":["456"],"name":"Another Test Event","description":"Another test event","startTime":"2099-05-02T12:00:00Z","address":"456 Test St","lat":51.5075,"long":-0.1279}]}`,
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
            name:        "Valid payload, missing auth header",
            expectMissingAuthHeader: true,
            requestBody: `{"events":[{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278},{"eventOwners":["456"],"name":"Another Test Event","description":"Another test event","startTime":"2099-05-02T12:00:00Z","address":"456 Test St","lat":51.5075,"long":-0.1279}]}`,
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
                if !strings.Contains(body, "Event at index 0 is missing eventOwners") {
                    return fmt.Errorf("expected 'Event at index 0 is missing eventOwners, got '%s'", body)
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
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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
                        "_id":          "123",
                        "eventOwners": []interface{}{"789"},
                        "name":        "First Test Event",
                        "description": "Description of the first event",
                    },
                    {
                        "_id":          "456",
                        "eventOwners": []interface{}{"012"},
                        "name":        "Second Test Event",
                        "description": "Description of the second event",
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
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

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
                    return services.SearchMarqoEvents(client, query, userLocation, maxDistance, startTime, endTime, ownerIds, string(""))
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
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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
			Errors: false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID: "123",
					Result: "",
					Status: 200,
				},
				{
					ID: "456",
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
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

    tests := []struct {
		name           string
		payload        string
		expectedStatus int
		expectedBody   string
        expectMissingAuthHeader bool
        mockUpsertFunc    func(client *marqo.Client, event types.Event) (*marqo.UpsertDocumentsResponse, error)
	}{
		{
			name:           "Invalid payload (missing ID in one event)",
			payload:        `{"events":[{"id":"abc","eventOwners":["123"],"name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364},{"eventOwners":["456"],"name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544}]}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Failed to extract event from payload: invalid body: Event at index 1 has no id`,
            expectMissingAuthHeader: false,
            mockUpsertFunc: nil,
		},
		{
			name:           "Valid payload, missing auth header",
			payload:        `{"events":[{"id":"abc","eventOwners":["123"],"name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364},{"id":"xyz","eventOwners":["456"],"name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544}]}`,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `Failed to get marqo event: error upserting documents: status code: 401`,
            expectMissingAuthHeader: true,
            mockUpsertFunc: nil,
		},
		{
			name:           "Valid payload",
			payload:        `{"events":[{"id":"abc","eventOwners":["123"],"name":"DC Bocce Ball Semifinals","description":"DC Bocce event description","startTime":"2099-02-15T18:30:00Z","address":"National Mall, Washington, DC","lat":38.8951,"long":-77.0364},{"id":"xyz","eventOwners":["456"],"name":"New York City Marathon","description":"NYC Marathon event description","startTime":"2099-11-02T08:00:00Z","address":"Fort Wadsworth, Staten Island, NY","lat":40.6075,"long":-74.0544}]}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `"errors":false`,
            expectMissingAuthHeader: false,
            mockUpsertFunc: nil,
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

			if !strings.Contains(strings.ToLower(rr.Body.String()), strings.ToLower(tt.expectedBody))  {
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
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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
            return
		}

		// Mock the response
		response := &marqo.UpsertDocumentsResponse{
			Errors: false,
			IndexName: "mock-events-search",
			Items: []marqo.Item{
				{
					ID: "123",
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
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
        t.Log("Started mock Marqo server")
    }
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

    tests := []struct {
        name string
        apiPath string
        requestBody string
        mockUpsertFunc    func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
        expectMissingAuthHeader bool
    }{
        {
            name:        "Valid event",
            apiPath:        `/test-id`,
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: func(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
                res, err := services.BulkUpsertEventToMarqo(client, events, false)
                if err != nil {
                    log.Printf("mocked request to upsert event failed: %v", err)
                }
                return &marqo.UpsertDocumentsResponse{}, fmt.Errorf("mocked request to upsert event res: %v", res)
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
            name:        "Valid payload, missing event path parameter",
            apiPath:        `/`,
            expectMissingAuthHeader: true,
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
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
            name:        "Valid payload, missing auth header",
            apiPath:        `/test-id`,
            expectMissingAuthHeader: true,
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
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
            name:           "Missing start time field",
            apiPath:        `/test-id`,
            requestBody:    `{"description":"A test event", "eventOwners":["123"],"lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "startTime is required") {
                    return fmt.Errorf("expected 'startTime is required', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:           "Missing name field",
            apiPath:        `/test-id`,
            requestBody:    `{"eventOwners":["123"],"startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278}`,
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
            name:        "Missing eventOwners field",
            apiPath:        `/test-id`,
            requestBody: `{"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, `Field validation for 'EventOwners' failed on the 'required' tag`) {
                    return fmt.Errorf("expected `Field validation for 'EventOwners' failed on the 'required' tag`, got '%s'", body)
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

            req, err := http.NewRequestWithContext(context.Background(), "PUT", "/events/" + tt.apiPath, bytes.NewBufferString(tt.requestBody))
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
