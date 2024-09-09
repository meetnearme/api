package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
)

func init() {
    os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
}

func TestCreateEvent(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := helpers.MOCK_MARQO_URL
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
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
        mockUpsertFunc    func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
        expectMissingAuthHeader bool
    }{
        {
            name:        "Valid event",
            requestBody: `{"eventOwners":["123"],"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error) {
                res, err := services.UpsertEventToMarqo(client, event)
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
            mockUpsertFunc: func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error) {
                res, err := services.UpsertEventToMarqo(client, event)
                if err != nil {
                    log.Printf("mocked request to upsert event failed: %v", err)
                }
                return res, nil
            },
            expectedStatus: http.StatusInternalServerError,
            expectedBodyCheck: func(body string) error {
                if strings.Contains(body, "ERR: Failed to upsert event to marqo") {
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
                if !strings.Contains(body, "Invalid JSON payload") {
                    return fmt.Errorf("expected 'Invalid JSON payload', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:           "Missing required field",
            requestBody:    `{"description":"A test event"}`,
            mockUpsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "Invalid body") {
                    return fmt.Errorf("expected 'Invalid body', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:        "Service error",
            requestBody: `{"name":"Test Event","description":"A test event","startTime":"2023-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
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
                UpsertEventToMarqoFunc: func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error) {
                    return tt.mockUpsertFunc(marqoClient, event)
                },
            }

            req, err := http.NewRequestWithContext(context.Background(), "POST", "/event", bytes.NewBufferString(tt.requestBody))
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }

            rr := httptest.NewRecorder()
            handler := NewMarqoHandler(mockService)

            handler.PostEvents(rr, req)

            if status := rr.Code; status != tt.expectedStatus {
                t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
            }

            if err := tt.expectedBodyCheck(rr.Body.String()); err != nil {
                t.Errorf("Body check failed: %v", err)
            }
        })
    }
}

// func TestUpsertEventToMarqo(t *testing.T) {
// 	// Save original environment variables
// 	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
// 	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

// 	// Set test environment variables
// 	testMarqoApiKey := "test-marqo-api-key"
// 	testMarqoEndpoint := MOCK_MARQO_URL
// 	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
// 	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

// 	// Defer resetting environment variables
// 	defer func() {
// 		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
// 		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
// 	}()

// 	// Create a mock HTTP server for Marqo
// 	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		authHeader := r.Header.Get("x-api-key")
// 		expectedAuthHeader := testMarqoApiKey
// 		if authHeader != expectedAuthHeader {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		// Mock the response
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(`{"success": true}`))
// 	}))

// 	// Set the mock Marqo server URL
// 	mockMarqoServer.Listener.Close()
// 	var err error
// 	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
// 	if err != nil {
// 		t.Fatalf("Failed to start mock Marqo server: %v", err)
// 	}
// 	mockMarqoServer.Start()
// 	defer mockMarqoServer.Close()

// 	// Test data
// 	eventTime := time.Date(2030, 5, 1, 12, 0, 0, 0, time.UTC)
// 	createEvent := services.Event{
// 		Name:        "New Event",
// 		Description: "New Description",
// 		StartTime:   eventTime.Format(time.RFC3339),
// 		Address:     "New Address",
// 		Lat:         float64(51.5074),
// 		Long:        float64(-0.1278),
// 	}

// 	newEvent, err := services.UpsertEventToMarqo(client, createEvent)
// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}
// 	if newEvent == nil {
// 		t.Fatal("Expected newEvent to be non-nil")
// 	}
// }


func TestSearchEvents(t *testing.T) {
    // TODO: implement test for event search
}

