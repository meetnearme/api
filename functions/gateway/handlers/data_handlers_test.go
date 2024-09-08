package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	testMarqoEndpoint := "http://localhost:8999"
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
		expectedAuthHeader := testMarqoApiKey
		if authHeader != expectedAuthHeader {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Mock the response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
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
        name string
        requestBody string
        mockUpsertFunc    func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
    }{
        {
            name:        "Valid event",
            requestBody: `{"name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error) {
                return &marqo.UpsertDocumentsResponse{
                    Errors: false,
                    IndexName: "mock-events-search",
                    Items: []marqo.Item{
                        {
                            ID: "998fa742-734c-4e6b-979d-1178f4806485",
                            Result: "",
                            Status: 200,
                        },
                    },
                    ProcessingTimeMS: 0.38569063499744516,
                }, nil
            },
            expectedStatus: http.StatusCreated,
            expectedBodyCheck: func(body string) error {
                var event map[string]interface{}
                if err := json.Unmarshal([]byte(body), &event); err != nil {
                    return fmt.Errorf("failed to unmarshal response body: %v", err)
                }
                if id, ok := event["id"].(string); !ok || id == "" {
                    return fmt.Errorf("expected non-empty id, got '%v'", id)
                }
                return nil
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
            requestBody: `{"name":"Test Event","description":"A test event","startTime":"2023-05-01T12:00:00Z","address":"123 Test St","zip_code":"12345","country":"Test Country","lat":51.5074,"long":-0.1278}`,
            mockUpsertFunc: func(client *marqo.Client, event services.Event) (*marqo.UpsertDocumentsResponse, error) {
                return &marqo.UpsertDocumentsResponse{
                    Errors: false,
                    Items: []marqo.Item{
                        {
                            ID: "998fa742-734c-4e6b-979d-1178f4806485",
                            Result: "200",
                        },
                    },
                    ProcessingTimeMS: 0.38569063499744516,
                }, nil
            },
            expectedStatus: http.StatusInternalServerError,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "Failed to add event") {
                    return fmt.Errorf("expected 'Failed to add event', got '%s'", body)
                }
                return nil
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {

            mockService := &services.MockEventService{
                UpsertEventToMarqoFunc: tt.mockUpsertFunc,
            }

            req, err := http.NewRequestWithContext(context.Background(), "POST", "/event", bytes.NewBufferString(tt.requestBody))
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }

            rr := httptest.NewRecorder()
            handler := NewEventHandler(mockService)

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

func TestSearchEvents(t *testing.T) {
    // TODO: implement test for event search
}
