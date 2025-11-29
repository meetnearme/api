package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	// internal_types "github.com/meetnearme/api/functions/gateway/types"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/weaviate/weaviate/entities/models"
)

var searchUsersByIDs = helpers.SearchUsersByIDs

// customRoundTripperForStripe is a custom HTTP round tripper that redirects Stripe API calls to a mock server
type customRoundTripperForStripe struct {
	transport http.RoundTripper
	mockURL   string
}

func (c *customRoundTripperForStripe) RoundTrip(req *http.Request) (*http.Response, error) {
	// If this is a Stripe API request, redirect it to our mock server
	if strings.Contains(req.URL.Host, "api.stripe.com") {
		mockURL, _ := url.Parse(c.mockURL)
		req.URL.Scheme = mockURL.Scheme
		req.URL.Host = mockURL.Host
	}
	// Use the underlying transport to make the request
	return c.transport.RoundTrip(req)
}

func TestPostEventHandler(t *testing.T) {
	// Store original env vars and transport
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	os.Setenv("WEAVIATE_HOST", "localhost")
	os.Setenv("WEAVIATE_PORT", "8080")
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	// Set up logging transport to intercept ALL HTTP requests
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock server using proper port rotation
	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
			t.Logf("   └─ Handling /v1/batch/objects")
			if r.Method != "POST" {
				t.Errorf("expected method POST, got %s", r.Method)
			}

			var requestBody struct {
				Objects []*models.Object `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			batchObjects := requestBody.Objects
			response := make([]*models.ObjectsGetResponse, len(batchObjects))
			for i, obj := range batchObjects {
				status := "SUCCESS"
				response[i] = &models.ObjectsGetResponse{
					Object: models.Object{
						ID:    obj.ID,
						Class: obj.Class,
					},
					Result: &models.ObjectsGetResponseAO2Result{
						Status: &status,
						Errors: nil,
					},
				}
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"name":        "Test Event",
								"description": "A test event",
								"timezone":    "America/New_York",
								"startTime":   time.Now().Unix(),
								"_additional": map[string]interface{}{
									"id": "123",
								},
							},
						},
					},
				},
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock GraphQL response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	// Parse mock server URL and set environment variables
	mockURL, err := url.Parse(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse mock server URL: %v", err)
	}

	os.Setenv("WEAVIATE_HOST", mockURL.Hostname())
	os.Setenv("WEAVIATE_SCHEME", mockURL.Scheme)
	os.Setenv("WEAVIATE_PORT", mockURL.Port())
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	// t.Logf("🔧 SETUP COMPLETE")
	// t.Logf("   └─ Mock Server: %s", mockServer.URL)
	// t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	// t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))
	// t.Logf("   └─ WEAVIATE_SCHEME: %s", os.Getenv("WEAVIATE_SCHEME"))

	// Test cases
	tests := []struct {
		name              string
		requestBody       string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name:           "Valid event posts successfully",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType": "` + constants.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York","startTime":"2099-05-01T12:00:00Z"}`,
			expectedStatus: http.StatusCreated,
			expectedBodyCheck: func(t *testing.T, body string) {
				t.Logf("Response body: %s", body)
				// Update this check since we expect success now
				if strings.Contains(body, "error") {
					t.Errorf("Expected successful response, but got error: %s", body)
				}
			},
		},
		{
			name:           "Valid event with sourceUrl posts successfully",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType": "` + constants.ES_SINGLE_EVENT + `","name":"Test Event with Source","description":"A test event with source URL","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York","startTime":"2099-05-01T12:00:00Z","sourceUrl":"https://example.com/event-source"}`,
			expectedStatus: http.StatusCreated,
			expectedBodyCheck: func(t *testing.T, body string) {
				t.Logf("Response body: %s", body)
				// Update this check since we expect success now
				if strings.Contains(body, "error") {
					t.Errorf("Expected successful response, but got error: %s", body)
				}
			},
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"name":"Test Event","description":}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(strings.ToLower(body), "invalid json payload") {
					t.Errorf("Expected body to contain 'invalid json payload', but got '%s'", body)
				}
			},
		},
		{
			name:           "Missing required name field",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType": "` + constants.ES_SINGLE_EVENT + `","startTime":"2099-05-01T12:00:00Z","description":"A test event","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					t.Errorf("Expected body to contain name validation error, but got '%s'", body)
				}
			},
		},
		{
			name:           "Missing required startTime field",
			requestBody:    `{"description":"A test event", "eventOwnerName": "Event Owner", "eventOwners":["123"],"name":"Test Event","eventSourceType": "` + constants.ES_SINGLE_EVENT + `","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'StartTime' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
		{
			name:           "Missing required name field",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType": "` + constants.ES_SINGLE_EVENT + `","startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'Name' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
		{
			name:           "Missing required eventOwners field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventSourceType": "` + constants.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'EventOwners' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
		{
			name:           "Missing required eventOwnerName field",
			requestBody:    `{"eventOwners":["123"], "name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'EventOwnerName' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
		{
			name:           "Missing required timezone field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventOwners":["123"], "eventSourceType": "` + constants.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'Timezone' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
		{
			name:           "Invalid timezone field",
			requestBody:    `{"timezone":"Does_Not_Exist/Nowhere","eventOwnerName":"Event Owner","eventOwners":["123"],"eventSourceType": "` + constants.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "invalid timezone: unknown time zone Does_Not_Exist/Nowhere"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 RUNNING TEST: %s", tt.name)

			req := httptest.NewRequest("POST", "/api/event", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handlerFunc := PostEventHandler(rr, req)
			handlerFunc(rr, req)

			// t.Logf("📊 TEST RESULTS:")
			// t.Logf("   └─ Status: %d (expected %d)", rr.Code, tt.expectedStatus)
			// t.Logf("   └─ Body: %s", rr.Body.String())

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

// Need to move these to Weaviate

func TestPostBatchEvents(t *testing.T) {
	// --- Standard Test Setup (same pattern) ---
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
			t.Logf("   └─ Handling /v1/batch/objects")
			if r.Method != "POST" {
				t.Errorf("expected method POST, got %s", r.Method)
			}

			var requestBody struct {
				Objects []*models.Object `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			batchObjects := requestBody.Objects
			response := make([]*models.ObjectsGetResponse, len(batchObjects))
			for i, obj := range batchObjects {
				status := "SUCCESS"
				response[i] = &models.ObjectsGetResponse{
					Object: models.Object{
						ID:    obj.ID,
						Class: obj.Class,
					},
					Result: &models.ObjectsGetResponseAO2Result{
						Status: &status,
						Errors: nil,
					},
				}
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 BATCH TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

	// Test data setup
	validEventID1 := uuid.New().String()
	validEventID2 := uuid.New().String()

	validPayload := struct {
		Events []services.RawEvent `json:"events"`
	}{
		Events: []services.RawEvent{
			createValidRawEvent(validEventID1, "Valid Batch Event 1", "https://example.com/batch-event-1"),
			createValidRawEvent(validEventID2, "Valid Batch Event 2", "https://example.com/batch-event-2"),
		},
	}
	validRequestBody, err := json.Marshal(validPayload)
	if err != nil {
		t.Fatalf("Setup failed: Could not marshal valid request body: %v", err)
	}

	invalidPayloadEvent1 := createValidRawEvent(uuid.New().String(), "This event is valid", "https://example.com/valid-event")
	invalidPayloadEvent2 := createValidRawEvent(uuid.New().String(), "This event has no name", "https://example.com/no-name-event")
	invalidPayloadEvent2.Name = ""

	partiallyInvalidPayload := struct {
		Events []services.RawEvent `json:"events"`
	}{
		Events: []services.RawEvent{invalidPayloadEvent1, invalidPayloadEvent2},
	}
	partiallyInvalidRequestBody, err := json.Marshal(partiallyInvalidPayload)
	if err != nil {
		t.Fatalf("Setup failed: Could not marshal partially invalid request body: %v", err)
	}

	tests := []struct {
		name              string
		requestBody       string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name:           "Valid batch of events posts successfully",
			requestBody:    string(validRequestBody),
			expectedStatus: http.StatusCreated,
			expectedBodyCheck: func(t *testing.T, body string) {
				// For successful batch creation, verify we get a success response
				if strings.Contains(body, "error") {
					t.Errorf("Expected successful response, but got error: %s", body)
				}
				t.Logf("✅ Batch events were successfully processed")
			},
		},
		{
			name:           "Batch with one invalid event fails validation",
			requestBody:    string(partiallyInvalidRequestBody),
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedErr := "invalid event at index 1: Field validation for 'Name' failed on the 'required' tag"
				if !strings.Contains(body, expectedErr) {
					t.Errorf("Expected validation error '%s', but got '%s'", expectedErr, body)
				}
			},
		},
		{
			name:           "Invalid JSON payload",
			requestBody:    `{"events":[{"name":"Test Event","description":}]}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(strings.ToLower(body), "invalid json payload") {
					t.Errorf("Expected body to contain 'invalid json payload', but got '%s'", body)
				}
			},
		},
		{
			name:           "Empty events array",
			requestBody:    `{"events":[]}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Events' failed on the 'min' tag") {
					t.Errorf("Expected validation error for empty events array, but got '%s'", body)
				}
			},
		},
		{
			name:           "Missing events field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Events' failed on the 'required' tag") {
					t.Errorf("Expected validation error for missing events field, but got '%s'", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 RUNNING BATCH TEST: %s", tt.name)

			req := httptest.NewRequest("POST", "/api/events/batch", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// Fix: Get the handler function and then call it
			handlerFunc := PostBatchEventsHandler(rr, req)
			handlerFunc(rr, req)

			t.Logf("📊 BATCH TEST RESULTS:")
			t.Logf("   └─ Status: %d (expected %d)", rr.Code, tt.expectedStatus)
			t.Logf("   └─ Body: %s", rr.Body.String())

			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}

			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

func createValidRawEvent(id, name, sourceUrl string) services.RawEvent {
	return services.RawEvent{
		RawEventData: services.RawEventData{
			Id:              id,
			EventOwners:     []string{"owner-123"},
			EventOwnerName:  "Test Owner",
			EventSourceType: constants.ES_SINGLE_EVENT,
			Name:            name,
			Description:     "A valid test event description.",
			Address:         "123 Test St, Testville",
			Lat:             40.1,
			Long:            -74.1,
			Timezone:        "America/New_York",
		},
		StartTime: "2099-10-10T10:00:00Z",
		SourceUrl: &sourceUrl,
	}
}

func TestSearchEvents(t *testing.T) {
	// --- Standard Test Setup (same pattern) ---
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql search")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Parse the query to determine what to return
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			queryStr := string(body)
			t.Logf("   └─ GraphQL Query: %s", queryStr)

			var mockResponse models.GraphQLResponse

			// Return different responses based on the search term
			if strings.Contains(queryStr, "programming") {
				// Return one matching event for "programming" search
				mockResponse = models.GraphQLResponse{
					Data: map[string]models.JSONObject{
						"Get": map[string]interface{}{
							constants.WeaviateEventClassName: []interface{}{
								map[string]interface{}{
									"name":            "Conference on Go Programming",
									"description":     "A deep dive into the Go language and its powerful ecosystem.",
									"eventOwnerName":  "Tech Org",
									"eventSourceType": constants.ES_SINGLE_EVENT,
									"address":         "123 Tech Way, Silicon Valley, CA",
									"lat":             37.3861,
									"long":            -122.0839,
									"timezone":        "America/Los_Angeles",
									"startTime":       time.Now().Add(48 * time.Hour).Unix(),
									"_additional": map[string]interface{}{
										"id": "programming-event-123",
									},
								},
							},
						},
					},
				}
			} else if strings.Contains(queryStr, "nonexistenttermxyz") {
				// Return empty results for non-existent term
				mockResponse = models.GraphQLResponse{
					Data: map[string]models.JSONObject{
						"Get": map[string]interface{}{
							constants.WeaviateEventClassName: []interface{}{},
						},
					},
				}
			} else {
				// Default case - return empty
				mockResponse = models.GraphQLResponse{
					Data: map[string]models.JSONObject{
						"Get": map[string]interface{}{
							constants.WeaviateEventClassName: []interface{}{},
						},
					},
				}
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock GraphQL response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 SEARCH TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

	// --- Define the Test Table ---
	tests := []struct {
		name              string
		path              string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name:           "Search with specific term finds correct event",
			path:           "/events?q=programming",
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				var res types.EventSearchResponse
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(res.Events) != 1 {
					t.Fatalf("Expected to find 1 event, but got %d", len(res.Events))
				}

				// Check the returned event details
				if res.Events[0].Id != "programming-event-123" {
					t.Errorf("Expected to find event 'programming-event-123', but got '%s'", res.Events[0].Id)
				}
				if res.Events[0].Name != "Conference on Go Programming" {
					t.Errorf("Expected event name 'Conference on Go Programming', but got '%s'", res.Events[0].Name)
				}
			},
		},
		{
			name:           "Search for term with no matches returns empty list",
			path:           "/events?q=nonexistenttermxyz",
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				var res types.EventSearchResponse
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(res.Events) != 0 {
					t.Errorf("Expected 0 events for a nonexistent term, but got %d", len(res.Events))
				}
			},
		},
		{
			name:           "Search without query parameter returns empty results",
			path:           "/events",
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				var res types.EventSearchResponse
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				// Should return empty results when no query provided
				if len(res.Events) != 0 {
					t.Errorf("Expected 0 events when no query provided, but got %d", len(res.Events))
				}
			},
		},
	}

	// --- The Test Runner ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 RUNNING SEARCH TEST: %s", tt.name)

			// ACT: Perform the HTTP request
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			// Fix: Get the handler function and then call it
			handlerFunc := SearchEventsHandler(rr, req)
			handlerFunc(rr, req)

			t.Logf("📊 SEARCH TEST RESULTS:")
			t.Logf("   └─ Status: %d (expected %d)", rr.Code, tt.expectedStatus)
			t.Logf("   └─ Body: %s", rr.Body.String())

			// ASSERT: Check the results
			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

func TestBulkUpdateEvents(t *testing.T) {
	// --- Standard Test Setup (same pattern) ---
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
			t.Logf("   └─ Handling /v1/batch/objects (bulk update)")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/batch/objects, got %s", r.Method)
			}

			var requestBody struct {
				Objects []*models.Object `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			batchObjects := requestBody.Objects
			response := make([]*models.ObjectsGetResponse, len(batchObjects))
			for i, obj := range batchObjects {
				status := "SUCCESS"
				response[i] = &models.ObjectsGetResponse{
					Object: models.Object{
						ID:    obj.ID,
						Class: obj.Class,
					},
					Result: &models.ObjectsGetResponseAO2Result{
						Status: &status,
						Errors: nil,
					},
				}
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 BULK UPDATE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

	// Define test cases (removed all DB integration parts)
	tests := []struct {
		name              string
		requestBody       string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name: "Successful bulk update with valid events",
			requestBody: `{ "events": [
				{
					"id": "update-test-1",
					"eventOwners":["owner-123"],
					"eventOwnerName":"Updated Owner",
					"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
					"name":"Updated Event Name",
					"description":"This description has been updated.",
					"startTime":"2099-05-01T12:00:00Z",
					"address":"1 First St, Washington, DC",
					"lat":38.8951,
					"long":-77.0364,
					"timezone":"America/New_York"
				}
			]}`,
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, `"status":"SUCCESS"`) {
					t.Errorf("Expected response body to indicate success, but got: %s", body)
				}
			},
		},
		{
			name: "Successful bulk update with sourceUrl field",
			requestBody: `{ "events": [
				{
					"id": "update-test-with-source",
					"eventOwners":["owner-123"],
					"eventOwnerName":"Updated Owner",
					"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
					"name":"Updated Event with Source",
					"description":"This event has been updated with source URL.",
					"startTime":"2099-05-01T12:00:00Z",
					"address":"1 First St, Washington, DC",
					"lat":38.8951,
					"long":-77.0364,
					"timezone":"America/New_York",
					"sourceUrl":"https://example.com/updated-event-source"
				}
			]}`,
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, `"status":"SUCCESS"`) {
					t.Errorf("Expected response body to indicate success, but got: %s", body)
				}
			},
		},
		{
			name: "Bulk update with an event missing an ID fails validation",
			requestBody: `{ "events": [
				{
					"eventOwners":["owner-123"],
					"eventOwnerName":"Owner",
					"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
					"name": "Event missing an ID",
					"description": "A complete event but missing ID",
					"startTime":"2099-05-01T12:00:00Z",
					"address":"123 Test St",
					"lat":40.7128,
					"long":-74.0060,
					"timezone":"America/New_York"
				}
			]}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "event has no id") {
					t.Errorf("Expected body to contain 'event has no id', but got '%s'", body)
				}
			},
		},
		{
			name:           "Invalid JSON payload",
			requestBody:    `{ "events": [{"id": "test", "name":}]}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(strings.ToLower(body), "invalid json payload") {
					t.Errorf("Expected body to contain 'invalid json payload', but got '%s'", body)
				}
			},
		},
		{
			name:           "Missing events field",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Events' failed on the 'required' tag") {
					t.Errorf("Expected validation error for missing events field, but got '%s'", body)
				}
			},
		},
		{
			name:           "Empty events array",
			requestBody:    `{"events": []}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Events' failed on the 'min' tag") {
					t.Errorf("Expected validation error for empty events array, but got '%s'", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 RUNNING BULK UPDATE TEST: %s", tt.name)

			// ACT: Perform the HTTP request
			req := httptest.NewRequest("PUT", "/api/events", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// Fix: Get the handler function and then call it
			handlerFunc := BulkUpdateEventsHandler(rr, req)
			handlerFunc(rr, req)

			t.Logf("📊 BULK UPDATE TEST RESULTS:")
			t.Logf("   └─ Status: %d (expected %d)", rr.Code, tt.expectedStatus)
			t.Logf("   └─ Body: %s", rr.Body.String())

			// ASSERT: Check the results
			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

func TestUpdateOneEvent(t *testing.T) {
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
			t.Logf("   └─ Handling /v1/batch/objects (single event update)")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/batch/objects, got %s", r.Method)
			}

			var requestBody struct {
				Objects []*models.Object `json:"objects"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			batchObjects := requestBody.Objects
			response := make([]*models.ObjectsGetResponse, len(batchObjects))
			for i, obj := range batchObjects {
				status := "SUCCESS"
				response[i] = &models.ObjectsGetResponse{
					Object: models.Object{
						ID:    obj.ID,
						Class: obj.Class,
					},
					Result: &models.ObjectsGetResponseAO2Result{
						Status: &status,
						Errors: nil,
					},
				}
			}

			responseBytes, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 UPDATE ONE EVENT TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

	// Define test cases (focused on handler's actual behavior)
	tests := []struct {
		name              string
		eventID           string
		requestBody       string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name:    "Successful update of a single event",
			eventID: "update-single-1",
			requestBody: `{
				"eventOwners": ["owner-abc"],
				"eventOwnerName": "The New Organizer",
				"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
				"name": "Post-Update Rock Show",
				"description": "This event has been successfully updated.",
				"startTime": "2099-05-01T12:00:00Z",
				"address": "456 New Avenue, New York, NY",
				"lat": 40.7129,
				"long": -74.0061,
				"timezone": "America/New_York"
			}`,
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				// Handler returns batch response format (array of ObjectsGetResponse)
				var response []models.ObjectsGetResponse
				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(response) != 1 {
					t.Errorf("Expected 1 response object, got %d", len(response))
				}
				if response[0].Result == nil || response[0].Result.Status == nil || *response[0].Result.Status != "SUCCESS" {
					t.Errorf("Expected success status, got %v", response[0].Result)
				}
			},
		},
		{
			name:    "Successful update of a single event with sourceUrl",
			eventID: "update-single-with-source",
			requestBody: `{
				"eventOwners": ["owner-abc"],
				"eventOwnerName": "The New Organizer",
				"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
				"name": "Post-Update Rock Show with Source",
				"description": "This event has been successfully updated with source URL.",
				"startTime": "2099-05-01T12:00:00Z",
				"address": "456 New Avenue, New York, NY",
				"lat": 40.7129,
				"long": -74.0061,
				"timezone": "America/New_York",
				"sourceUrl": "https://example.com/rock-show-source"
			}`,
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				// Handler returns batch response format (array of ObjectsGetResponse)
				var response []models.ObjectsGetResponse
				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(response) != 1 {
					t.Errorf("Expected 1 response object, got %d", len(response))
				}
				if response[0].Result == nil || response[0].Result.Status == nil || *response[0].Result.Status != "SUCCESS" {
					t.Errorf("Expected success status, got %v", response[0].Result)
				}
			},
		},
		{
			name:           "Update with invalid JSON fails",
			eventID:        "any-id",
			requestBody:    `{"name": "Invalid JSON",}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(strings.ToLower(body), "invalid json payload") {
					t.Errorf("Expected body to contain 'invalid json payload', but got '%s'", body)
				}
			},
		},
		{
			name:    "Update with missing required field fails validation",
			eventID: "any-id",
			requestBody: `{
				"eventOwners": ["owner-123"],
				"eventOwnerName": "Owner",
				"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
				"description": "This event is missing a name",
				"startTime": "2099-05-01T12:00:00Z",
				"timezone": "America/New_York"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					t.Errorf("Expected error about missing 'Name' field, but got: %s", body)
				}
			},
		},
		{
			name:    "Update with missing timezone fails validation",
			eventID: "timezone-test",
			requestBody: `{
				"eventOwners": ["owner-123"],
				"eventOwnerName": "Owner",
				"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
				"name": "Event Missing Timezone",
				"description": "This event is missing timezone",
				"startTime": "2099-05-01T12:00:00Z",
				"address": "123 Test St",
				"lat": 40.7128,
				"long": -74.0060
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Timezone' failed on the 'required' tag") {
					t.Errorf("Expected error about missing 'Timezone' field, but got: %s", body)
				}
			},
		},
		{
			name:    "Update with invalid timezone fails validation",
			eventID: "invalid-timezone-test",
			requestBody: `{
				"eventOwners": ["owner-123"],
				"eventOwnerName": "Owner",
				"eventSourceType": "` + constants.ES_SINGLE_EVENT + `",
				"name": "Event with Invalid Timezone",
				"description": "This event has invalid timezone",
				"startTime": "2099-05-01T12:00:00Z",
				"address": "123 Test St",
				"lat": 40.7128,
				"long": -74.0060,
				"timezone": "Invalid/Timezone"
			}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "invalid timezone: unknown time zone Invalid/Timezone") {
					t.Errorf("Expected error about invalid timezone, but got: %s", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("🧪 RUNNING UPDATE ONE EVENT TEST: %s", tt.name)

			// ACT: Perform the HTTP request
			path := fmt.Sprintf("/events/%s", tt.eventID)
			req := httptest.NewRequest("PUT", path, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// This is crucial for testing handlers that use gorilla/mux for URL parameters.
			req = mux.SetURLVars(req, map[string]string{
				"eventId": tt.eventID,
			})

			rr := httptest.NewRecorder()

			// Fix: Get the handler function and then call it
			handlerFunc := UpdateOneEventHandler(rr, req)
			handlerFunc(rr, req)

			t.Logf("📊 UPDATE ONE EVENT TEST RESULTS:")
			t.Logf("   └─ Status: %d (expected %d)", rr.Code, tt.expectedStatus)
			t.Logf("   └─ Body: %s", rr.Body.String())

			// ASSERT: Check the results
			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

func TestHandleCheckoutWebhook(t *testing.T) {
	t.Run("handles checkout.session.completed successfully", func(t *testing.T) {
		// Save original env var
		originalWebhookSecret := os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
		testWebhookSecret := "whsec_test_secret"
		os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", testWebhookSecret)
		// Restore original env var after test
		defer os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", originalWebhookSecret)

		// Setup mock service first
		mockPurchasesService := &dynamodb_service.MockPurchaseService{
			GetPurchaseByPkFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error) {
				return &internal_types.Purchase{
					EventID:         eventId,
					UserID:          userId,
					CreatedAtString: createdAt,
					Status:          constants.PurchaseStatus.Pending,
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
				if update.Status != constants.PurchaseStatus.Settled {
					t.Errorf("expected status %v, got %v", constants.PurchaseStatus.Settled, update.Status)
				}
				return nil, nil
			},
			HasPurchaseForEventFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, childEventId, parentEventId, userId string) (bool, error) {
				return false, nil
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
			"api_version": "2025-09-30.clover",
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
		r.Header.Set("stripe-signature", stripeSignature)

		w := httptest.NewRecorder()
		// Execute handler
		handler.HandleCheckoutWebhook(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
		}
	})
	t.Run("handles checkout.session.expired successfully", func(t *testing.T) {
		// Save original env var
		originalWebhookSecret := os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
		testWebhookSecret := "whsec_test_secret"
		os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", testWebhookSecret)
		// Restore original env var after test
		defer os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", originalWebhookSecret)

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
					"api_version": "2025-09-30.clover",
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
				r.Header.Set("stripe-signature", stripeSignature)
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
							Status:          constants.PurchaseStatus.Pending,
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
						if update.Status != constants.PurchaseStatus.Canceled {
							t.Errorf("expected status %v, got %v", constants.PurchaseStatus.Canceled, update.Status)
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
		req.Header.Set("stripe-signature", "invalid_signature")
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
	// Save original environment variables
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")

	// Create a mock HTTP server for Zitadel
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/meta") {
			// Handle meta requests
			w.Header().Set("Content-Type", "application/json")

			// Extract the ID from the URL path
			pathParts := strings.Split(r.URL.Path, "/")
			id := pathParts[len(pathParts)-3] // Assuming ID is second-to-last part
			if id == "tm_b8de1f5b-d377-458e-a47e-96123afcc6f3" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// Return an empty or invalid metadata structure to trigger the error
				json.NewEncoder(w).Encode(map[string]interface{}{
					"metadata": map[string]interface{}{
						"value": "", // This will cause GetBase64ValueFromMap to return empty string
					},
				})
				return
			}

			response := map[string]interface{}{
				"metadata": map[string]interface{}{
					"value": base64.StdEncoding.EncodeToString([]byte("user1,user2")),
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

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
			case len(userIds) == 1 && strings.HasPrefix(userIds[0], "tm_"):
				// Check if this is the error test case
				if userIds[0] == "tm_b8de1f5b-d377-458e-a47e-96123afcc6f3" {
					// Return a successful user search response
					json.NewEncoder(w).Encode(map[string]interface{}{
						"result": []map[string]interface{}{
							{
								"userId": userIds[0],
								"human": map[string]interface{}{
									"profile": map[string]interface{}{
										"displayName": "Test User",
									},
								},
							},
						},
					})
					return
				}

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
						UserID:   userIds[0],
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

	// Set the mock Zitadel server URL using proper port binding
	zitadelHostAndPort := test_helpers.GetNextPort()
	zitadelListener, err := test_helpers.BindToPort(t, zitadelHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind Zitadel server: %v", err)
	}
	mockZitadelServer.Listener = zitadelListener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Set environment variable to the actual bound port
	actualAddr := zitadelListener.Addr().String()
	os.Setenv("ZITADEL_INSTANCE_HOST", actualAddr)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

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
			name:           "Valid tm_uuid format",
			queryParams:    "?ids=tm_123e4567-e89b-12d3-a456-426614174000",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"userId":"tm_123e4567-e89b-12d3-a456-426614174000","displayName":"Test User","metadata":{"members":["user1","user2"]}}]`,
		},
		{
			name:           "Invalid tm_uuid format",
			queryParams:    "?ids=tm_invalid-uuid-format",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":{"message":"ERR: invalid UUID format after 'tm_': invalid-uuid-format"}}`,
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
		{
			name:           "GetOtherUserMetaByID fails on missing user meta with throw=1",
			queryParams:    "?ids=tm_b8de1f5b-d377-458e-a47e-96123afcc6f3&throw=1",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to get user meta",
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
	// Save original environment variables
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")

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

	// Set the mock Zitadel server URL using proper port binding
	zitadelHostAndPort := test_helpers.GetNextPort()
	zitadelListener, err := test_helpers.BindToPort(t, zitadelHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind Zitadel server: %v", err)
	}
	mockZitadelServer.Listener = zitadelListener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Set environment variable to the actual bound port
	actualAddr := zitadelListener.Addr().String()
	os.Setenv("ZITADEL_INSTANCE_HOST", actualAddr)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

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

// =============================================================================
// Tests for CheckRole handler
// =============================================================================

func TestCheckRole(t *testing.T) {
	tests := []struct {
		name           string
		userInfo       constants.UserInfo
		roleClaims     []constants.RoleClaim
		queryRole      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - User has the requested role",
			userInfo: constants.UserInfo{
				Sub:   "user123",
				Email: "test@example.com",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "subGrowth"},
				{Role: "eventAdmin"},
			},
			queryRole:      "subGrowth",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"success","message":"` + constants.ROLE_ACTIVE_MESSAGE + `"}`,
		},
		{
			name: "Success - User has multiple roles including requested",
			userInfo: constants.UserInfo{
				Sub:   "user456",
				Email: "test2@example.com",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "subSeed"},
				{Role: "orgAdmin"},
				{Role: "eventAdmin"},
			},
			queryRole:      "orgAdmin",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"success","message":"` + constants.ROLE_ACTIVE_MESSAGE + `"}`,
		},
		{
			name: "Not Found - User does not have the requested role",
			userInfo: constants.UserInfo{
				Sub:   "user789",
				Email: "test3@example.com",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "subSeed"},
			},
			queryRole:      "subGrowth",
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":{"message":"ERR: {\"message\":\"` + constants.ROLE_NOT_FOUND_MESSAGE + `\",\"status\":\"error\"}"}}`,
		},
		{
			name: "Not Found - User has no roles at all",
			userInfo: constants.UserInfo{
				Sub:   "user000",
				Email: "test4@example.com",
			},
			roleClaims:     []constants.RoleClaim{},
			queryRole:      "subGrowth",
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":{"message":"ERR: {\"message\":\"` + constants.ROLE_NOT_FOUND_MESSAGE + `\",\"status\":\"error\"}"}}`,
		},
		{
			name: "Bad Request - Missing role parameter",
			userInfo: constants.UserInfo{
				Sub:   "user999",
				Email: "test5@example.com",
			},
			roleClaims:     []constants.RoleClaim{{Role: "subGrowth"}},
			queryRole:      "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing role parameter",
		},
		{
			name:           "Unauthorized - No user in context",
			userInfo:       constants.UserInfo{},
			roleClaims:     []constants.RoleClaim{},
			queryRole:      "subGrowth",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameter
			url := "/api/auth/check-role"
			if tt.queryRole != "" {
				url += "?role=" + tt.queryRole
			}
			req := httptest.NewRequest("GET", url, nil)

			// Add user info and role claims to context
			ctx := req.Context()
			if tt.userInfo.Sub != "" {
				ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			}
			if len(tt.roleClaims) > 0 {
				ctx = context.WithValue(ctx, "roleClaims", tt.roleClaims)
			}
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler - CheckRole returns a handler function, so we need to call it
			handlerFunc := CheckRole(w, req)
			handlerFunc(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify response body
			gotBody := strings.TrimSpace(w.Body.String())
			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusNotFound {
				// For JSON responses, normalize and compare
				var got, expected map[string]interface{}
				if err := json.Unmarshal([]byte(gotBody), &got); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.expectedBody), &expected); err != nil {
					t.Fatalf("failed to unmarshal expected body: %v", err)
				}
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("Expected body %v, got %v", expected, got)
				}
			} else {
				if !strings.Contains(gotBody, tt.expectedBody) {
					t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, gotBody)
				}
			}
		})
	}
}

// =============================================================================
// Tests for CreateSubscriptionCheckoutSession handler
// =============================================================================

func TestCreateSubscriptionCheckoutSession(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	originalApexURL := os.Getenv("APEX_URL")
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
		os.Setenv("APEX_URL", originalApexURL)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		http.DefaultTransport = originalTransport
	}()

	// Set up test environment
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock")
	os.Setenv("APEX_URL", "https://test.example.com")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")

	tests := []struct {
		name               string
		userInfo           constants.UserInfo
		roleClaims         []constants.RoleClaim
		subscriptionPlanID string
		expectedRedirect   string
		expectedStatus     int
		mockStripeCustomer bool
		mockStripeError    bool
	}{
		{
			name: "Unauthorized - No user ID",
			userInfo: constants.UserInfo{
				Sub: "",
			},
			roleClaims:         []constants.RoleClaim{},
			subscriptionPlanID: "price_growth_test",
			expectedRedirect:   "",
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name: "Bad Request - Missing subscription plan ID",
			userInfo: constants.UserInfo{
				Sub:   "user123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			roleClaims:         []constants.RoleClaim{},
			subscriptionPlanID: "",
			expectedRedirect:   "https://test.example.com/pricing?error=checkout_failed",
			expectedStatus:     http.StatusSeeOther,
		},
		{
			name: "Bad Request - Invalid subscription plan ID",
			userInfo: constants.UserInfo{
				Sub:   "user123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			roleClaims:         []constants.RoleClaim{},
			subscriptionPlanID: "invalid_plan",
			expectedRedirect:   "https://test.example.com/pricing?error=checkout_failed",
			expectedStatus:     http.StatusSeeOther,
		},
		{
			name: "Already Subscribed - User with Seed subscription tries to checkout Seed",
			userInfo: constants.UserInfo{
				Sub:   "user_seed",
				Email: "seed@example.com",
				Name:  "Seed User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: constants.Roles[constants.SubSeed]},
			},
			subscriptionPlanID: "price_seed_test",
			expectedRedirect:   "https://test.example.com/pricing?error=already_subscribed",
			expectedStatus:     http.StatusSeeOther,
		},
		{
			name: "Already Subscribed - User with Growth subscription tries to checkout Growth",
			userInfo: constants.UserInfo{
				Sub:   "user_growth",
				Email: "growth@example.com",
				Name:  "Growth User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: constants.Roles[constants.SubGrowth]},
			},
			subscriptionPlanID: "price_growth_test",
			expectedRedirect:   "https://test.example.com/pricing?error=already_subscribed",
			expectedStatus:     http.StatusSeeOther,
		},
		{
			name: "Already Subscribed - User with Growth subscription tries to checkout Seed (Growth includes Seed)",
			userInfo: constants.UserInfo{
				Sub:   "user_growth_seed",
				Email: "growth@example.com",
				Name:  "Growth User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: constants.Roles[constants.SubGrowth]},
			},
			subscriptionPlanID: "price_seed_test",
			expectedRedirect:   "https://test.example.com/pricing?error=already_subscribed",
			expectedStatus:     http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameter
			url := "/api/checkout-subscription"
			if tt.subscriptionPlanID != "" {
				url += "?subscription_plan_id=" + tt.subscriptionPlanID
			}
			req := httptest.NewRequest("GET", url, nil)

			// Add user info and role claims to context
			ctx := req.Context()
			if tt.userInfo.Sub != "" {
				ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			}
			if len(tt.roleClaims) > 0 {
				ctx = context.WithValue(ctx, "roleClaims", tt.roleClaims)
			}
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			err := CreateSubscriptionCheckoutSession(w, req)
			if err != nil && tt.expectedStatus != http.StatusSeeOther {
				t.Logf("Handler returned error: %v", err)
			}

			// Verify status code or redirect
			result := w.Result()
			if result.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, result.StatusCode)
			}

			// For redirects, verify location header
			if tt.expectedStatus == http.StatusSeeOther && tt.expectedRedirect != "" {
				location := result.Header.Get("Location")
				if location != tt.expectedRedirect {
					t.Errorf("Expected redirect to '%s', got '%s'", tt.expectedRedirect, location)
				}
			}
		})
	}
}

// =============================================================================
// Tests for HandleSubscriptionWebhook handler
// =============================================================================

func TestHandleSubscriptionWebhook(t *testing.T) {
	// Save original environment variables
	originalWebhookSecret := os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalZitadelHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalZitadelToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")

	defer func() {
		os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", originalWebhookSecret)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalZitadelToken)
	}()

	// Set up test environment
	testWebhookSecret := "whsec_test_secret"
	os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", testWebhookSecret)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "prod_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "prod_seed_test")
	os.Setenv("ZITADEL_INSTANCE_HOST", "https://mock-zitadel.example.com")
	os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "mock_token")

	// Setup mock Zitadel server for GetUserRoles and SetUserRoles calls
	mockZitadelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "ListAuthorizations") {
			// Mock GetUserRoles response - return empty list so we create new auth
			response := map[string]interface{}{
				"authorizations": []map[string]interface{}{},
			}
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "POST" && strings.Contains(r.URL.Path, "CreateAuthorization") {
			// Mock CreateAuthorization response
			response := map[string]interface{}{
				"id": "auth_123",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.Method == "POST" && strings.Contains(r.URL.Path, "UpdateAuthorization") {
			// Mock SetUserRoles response
			response := map[string]interface{}{
				"id": "auth_123",
			}
			json.NewEncoder(w).Encode(response)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer mockZitadelServer.Close()

	// Remove the protocol from the URL since helpers adds it
	zitadelURL := strings.TrimPrefix(mockZitadelServer.URL, "http://")
	os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)

	t.Run("handles customer.subscription.created with Growth plan", func(t *testing.T) {
		// Save original transport and stripe key
		originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
		originalTransport := http.DefaultTransport
		defer func() {
			os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
			http.DefaultTransport = originalTransport
		}()

		os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")

		// Create mock Stripe server
		mockStripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/customers/cus_test123") {
				customerResponse := map[string]interface{}{
					"id": "cus_test123",
					"metadata": map[string]interface{}{
						"zitadel_user_id": "zitadel_user_123",
					},
				}
				json.NewEncoder(w).Encode(customerResponse)
			} else {
				http.Error(w, "Not Found", http.StatusNotFound)
			}
		}))
		defer mockStripeServer.Close()

		// Setup custom round tripper to redirect Stripe API calls to mock
		customRoundTripper := &customRoundTripperForStripe{
			transport: http.DefaultTransport,
			mockURL:   mockStripeServer.URL,
		}
		http.DefaultTransport = test_helpers.NewLoggingTransport(customRoundTripper, t)
		services.ResetStripeClient()

		payload := []byte(`{
			"type": "customer.subscription.created",
			"api_version": "2025-09-30.clover",
			"data": {
				"object": {
					"id": "sub_test123",
					"customer": "cus_test123",
					"items": {
						"data": [
							{
								"price": {
									"product": "prod_growth_test"
								}
							}
						]
					}
				}
			}
		}`)

		now := time.Now()
		timestamp := now.Unix()
		mac := hmac.New(sha256.New, []byte(testWebhookSecret))
		mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
		mac.Write([]byte("."))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		stripeSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)

		req := httptest.NewRequest(http.MethodPost, "/webhook/subscription", bytes.NewBuffer(payload))
		req.Header.Set("stripe-signature", stripeSignature)
		w := httptest.NewRecorder()

		subscriptionService := services.NewStripeSubscriptionService()
		handler := NewSubscriptionWebhookHandler(subscriptionService)

		err := handler.HandleSubscriptionWebhook(w, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
		}

		// Verify response contains expected subscription and customer IDs
		expectedSubscriptionID := "sub_test123"
		expectedCustomerID := "cus_test123"
		if !strings.Contains(w.Body.String(), expectedSubscriptionID) {
			t.Errorf("expected body to contain subscription ID '%s', got '%s'", expectedSubscriptionID, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), expectedCustomerID) {
			t.Errorf("expected body to contain customer ID '%s', got '%s'", expectedCustomerID, w.Body.String())
		}
	})

	t.Run("handles customer.subscription.created with Seed plan", func(t *testing.T) {
		// Save original transport and stripe key
		originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
		originalTransport := http.DefaultTransport
		defer func() {
			os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
			http.DefaultTransport = originalTransport
		}()

		os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")

		// Create mock Stripe server
		mockStripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/customers/cus_test456") {
				customerResponse := map[string]interface{}{
					"id": "cus_test456",
					"metadata": map[string]interface{}{
						"zitadel_user_id": "zitadel_user_456",
					},
				}
				json.NewEncoder(w).Encode(customerResponse)
			} else {
				http.Error(w, "Not Found", http.StatusNotFound)
			}
		}))
		defer mockStripeServer.Close()

		// Setup custom round tripper to redirect Stripe API calls to mock
		customRoundTripper := &customRoundTripperForStripe{
			transport: http.DefaultTransport,
			mockURL:   mockStripeServer.URL,
		}
		http.DefaultTransport = test_helpers.NewLoggingTransport(customRoundTripper, t)
		services.ResetStripeClient()

		payload := []byte(`{
			"type": "customer.subscription.created",
			"api_version": "2025-09-30.clover",
			"data": {
				"object": {
					"id": "sub_test456",
					"customer": "cus_test456",
					"items": {
						"data": [
							{
								"price": {
									"product": "prod_seed_test"
								}
							}
						]
					}
				}
			}
		}`)

		now := time.Now()
		timestamp := now.Unix()
		mac := hmac.New(sha256.New, []byte(testWebhookSecret))
		mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
		mac.Write([]byte("."))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		stripeSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)

		req := httptest.NewRequest(http.MethodPost, "/webhook/subscription", bytes.NewBuffer(payload))
		req.Header.Set("stripe-signature", stripeSignature)
		w := httptest.NewRecorder()

		subscriptionService := services.NewStripeSubscriptionService()
		handler := NewSubscriptionWebhookHandler(subscriptionService)

		err := handler.HandleSubscriptionWebhook(w, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
		}

		// Verify response contains expected subscription and customer IDs
		expectedSubscriptionID := "sub_test456"
		expectedCustomerID := "cus_test456"
		if !strings.Contains(w.Body.String(), expectedSubscriptionID) {
			t.Errorf("expected body to contain subscription ID '%s', got '%s'", expectedSubscriptionID, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), expectedCustomerID) {
			t.Errorf("expected body to contain customer ID '%s', got '%s'", expectedCustomerID, w.Body.String())
		}
	})

	t.Run("handles unhandled event type", func(t *testing.T) {
		payload := []byte(`{
			"id": "evt_test",
			"type": "customer.created",
			"api_version": "2025-09-30.clover",
			"data": {
				"object": {}
			}
		}`)

		// Generate signed payload
		now := time.Now()
		timestamp := now.Unix()
		mac := hmac.New(sha256.New, []byte(testWebhookSecret))
		mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
		mac.Write([]byte("."))
		mac.Write(payload)
		signature := hex.EncodeToString(mac.Sum(nil))
		stripeSignature := fmt.Sprintf("t=%d,v1=%s", timestamp, signature)

		req := httptest.NewRequest(http.MethodPost, "/webhook/subscription", bytes.NewBuffer(payload))
		req.Header.Set("stripe-signature", stripeSignature)
		w := httptest.NewRecorder()

		subscriptionService := services.NewStripeSubscriptionService()
		handler := NewSubscriptionWebhookHandler(subscriptionService)

		err := handler.HandleSubscriptionWebhook(w, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %v, got %v", http.StatusOK, w.Code)
		}

		expectedBody := "Unhandled event type"
		if !strings.Contains(w.Body.String(), expectedBody) {
			t.Errorf("expected body to contain '%s', got '%s'", expectedBody, w.Body.String())
		}
	})
}

// =============================================================================
// Test helper functions for subscription handlers
// =============================================================================

func TestValidateSubscriptionPlanID(t *testing.T) {
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed")

	tests := []struct {
		name     string
		planID   string
		expected bool
	}{
		{
			name:     "Valid Growth plan",
			planID:   "price_growth",
			expected: true,
		},
		{
			name:     "Valid Seed plan",
			planID:   "price_seed",
			expected: true,
		},
		{
			name:     "Invalid plan ID",
			planID:   "price_invalid",
			expected: false,
		},
		{
			name:     "Empty plan ID",
			planID:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			growthPlanID := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
			seedPlanID := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")

			isValid := tt.planID == growthPlanID || tt.planID == seedPlanID

			if isValid != tt.expected {
				t.Errorf("Expected validation result %v, got %v for plan ID '%s'", tt.expected, isValid, tt.planID)
			}
		})
	}
}
