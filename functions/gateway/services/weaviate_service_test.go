package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/weaviate/weaviate/entities/models"
)

func TestGetWeaviateClient(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalWeaviateApiKey := os.Getenv("WEAVIATE_API_KEY_ALLOWED_KEYS")

	// Set test environment variables
	hostAndPort := test_helpers.GetNextPort()

	parts := strings.Split(hostAndPort, ":")
	if len(parts) != 2 {
		t.Fatalf("Expected GetNextPort to return 'host:port', but got: %s", hostAndPort)
	}

	host := parts[0]
	port := parts[1]

	os.Setenv("WEAVIATE_HOST", host)
	os.Setenv("WEAVIATE_PORT", port)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	// Defer resetting environment variables
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", originalWeaviateApiKey)
	}()

	// Create a mock HTTP server for Weaviate
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response for the /.well-known/ready endpoint
		if r.URL.Path == "/v1/.well-known/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Mock the response for other endpoints
		response := &models.GraphQLResponse{}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Weaviate server URL
	mockWeaviateServer.Listener.Close()
	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
	if err != nil {
		t.Fatalf("Failed to start mock Weaviate server after retries: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// --- ADD THESE TWO DEBUG LINES ---
	log.Printf("DEBUG: WEAVIATE_HOST is set to: '%s'", os.Getenv("WEAVIATE_HOST"))
	log.Printf("DEBUG: WEAVIATE_PORT is set to: '%s'", os.Getenv("WEAVIATE_PORT"))
	// --- END OF DEBUG LINES ---

	// Call the handler
	client, err := GetWeaviateClient()
	if err != nil {
		t.Errorf("error getting weaviate client %v", err)
	}

	// we can't test the private variable `client.url`, so instead we make a mocked call
	// and if no error is thrown, we know the mock server responded on the configured URL
	_, err = client.GraphQL().Get().WithClassName("Test").Do(context.Background())
	if err != nil {
		t.Errorf("mocked endpoint not responding, error calling Get %v", err)
	}
}

func TestBulkUpsertEventsToWeaviate(t *testing.T) {
	// --- Standard Test Setup ---
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
	}()

	hostAndPort := test_helpers.GetNextPort()
	parts := strings.Split(hostAndPort, ":")
	host, port := parts[0], parts[1]

	os.Setenv("WEAVIATE_HOST", host)
	os.Setenv("WEAVIATE_PORT", port)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batch/objects" {
			t.Errorf("expected path /v1/batch/objects, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}

		// The client sends an array of 'models.Object'
		var batchObjects []*models.Object
		if err := json.NewDecoder(r.Body).Decode(&batchObjects); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// The client expects a response of type '[]*bmodels.ObjectsGetResponse'
		// We will build this response structure correctly.
		response := make([]*models.ObjectsGetResponse, len(batchObjects))
		for i, obj := range batchObjects {

			// The status is a pointer to a string.
			status := "SUCCESS"

			// This is the correct structure for each item in the response array.
			response[i] = &models.ObjectsGetResponse{
				Object: models.Object{
					ID:    obj.ID,
					Class: obj.Class,
				},
				Result: &models.ObjectsGetResponseAO2Result{
					Status: &status, // Status is inside the Result object
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
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// --- Test Execution ---
	loc, _ := time.LoadLocation("America/New_York")
	startTime1, err := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	if err != nil {
		t.Fatalf("failed to convert time: %v", err)
	}

	events := []types.Event{
		{
			Id:          "00000000-0000-0000-0000-000000000001",
			EventOwners: []string{"123"},
			Name:        "Test Event 1",
			StartTime:   startTime1,
		},
	}

	client, err := GetWeaviateClient()
	if err != nil {
		t.Fatalf("GetWeaviateClient failed: %v", err)
	}

	res, err := BulkUpsertEventsToWeaviate(context.Background(), client, events)
	if err != nil {
		t.Errorf("BulkUpsertEventsToWeaviate returned an unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected a non-nil response, but got nil")
	}
	if len(res) != 1 {
		t.Errorf("expected response length 1, got %d", len(res))
	}
	if *(res)[0].Result.Status != models.ObjectsGetResponseAO2ResultStatusSUCCESS {
		t.Errorf("expected status SUCCESS, got %s", *(res)[0].Result.Status)
	}
}

func TestSearchWeaviateEvents(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
	}()

	hostAndPort := test_helpers.GetNextPort()
	parts := strings.Split(hostAndPort, ":")
	host, port := parts[0], parts[1]

	os.Setenv("WEAVIATE_HOST", host)
	os.Setenv("WEAVIATE_PORT", port)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// We use a switch to handle different endpoints the client might call.
		switch r.URL.Path {

		// FIX 1: Handle the initial "handshake" call from the client.
		case "/v1/meta":
			if r.Method != "GET" {
				t.Errorf("expected method GET for /v1/meta, got %s", r.Method)
			}
			// Send back a minimal, successful meta response.
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		// Handle the actual search query.
		case "/v1/graphql":
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// FIX 2: Make the mock response more realistic.
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						// The key here should match the Class name your code expects.
						"EventStrict": []interface{}{
							map[string]interface{}{
								"ClassName":   "Event", // The field your code was looking for.
								"name":        "Rock Concert",
								"description": "A loud rock concert.",
								"startTime":   time.Now().Unix(),
								"_additional": map[string]interface{}{
									"id": strfmt.UUID("00000000-0000-0000-0000-000000000123"),
								},
							},
						},
					},
				},
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("Failed to start mock Weaviate server for test: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	client, err := GetWeaviateClient()
	if err != nil {
		t.Fatalf("Failed to get Weaviate client: %v", err)
	}
	// Call the function we want to test
	res, err := SearchWeaviateEvents(context.Background(), client, "concert", []float64{0, 0}, 1000, 0, 0, []string{}, "", "", "10", []string{}, []string{})

	// --- Assertions ---
	if err != nil {
		t.Errorf("Unexpected error from SearchWeaviateEvents: %v", err)
	}

	if len(res.Events) != 1 {
		t.Errorf("Expected 1 event in the response, but got %d", len(res.Events))
	} else {
		expectedID := "00000000-0000-0000-0000-000000000123"
		if res.Events[0].Id != expectedID {
			t.Errorf("Expected event ID %s, but got %s", expectedID, res.Events[0].Id)
		}

		expectedName := "Rock Concert"
		if res.Events[0].Name != expectedName {
			t.Errorf("Expected event name '%s', but got '%s'", expectedName, res.Events[0].Name)
		}
	}
}

func TestBulkDeleteEventsFromWeaviate(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
	}()

	hostAndPort := test_helpers.GetNextPort()
	parts := strings.Split(hostAndPort, ":")
	if len(parts) != 2 {
		t.Fatalf("GetNextPort should return host:port, got: %s", hostAndPort)
	}
	host, port := parts[0], parts[1]

	os.Setenv("WEAVIATE_HOST", host)
	os.Setenv("WEAVIATE_PORT", port)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {

		// FIX 1: Handle the initial "handshake" call from the client.
		case "/v1/meta":
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
			if r.Method != "DELETE" {
				t.Errorf("expected method DELETE, got %s", r.Method)
			}

			var requestBody map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}

			// ~~ Delete me
			prettyJSON, _ := json.MarshalIndent(requestBody, "", "  ")
			t.Logf("RECEIVED REQUEST BODY:\n%s", string(prettyJSON))
			/// ~~
			matchObject, ok := requestBody["match"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected delete request body to contain a 'match' object")
			}

			if _, ok := matchObject["where"]; !ok {
				t.Error("expected delete request body to contain a 'where' filter")
			}

			mockResponse := models.BatchDeleteResponse{
				Results: &models.BatchDeleteResponseResults{
					Matches:    1,
					Successful: 1,
					Failed:     0,
				}}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)
		default:
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
		}
	}))

	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
	if err != nil {
		t.Fatalf("Failed to start mock Weaviate server for test: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Actual test section

	client, err := GetWeaviateClient()
	if err != nil {
		t.Fatalf("Failed to get Weaviate client: %v", err)
	}

	res, err := BulkDeleteEventsFromWeaviate(context.Background(), client, []string{"123"})

	if err != nil {
		t.Errorf("Unexpected error from BulkDeleteEventsFromWeaviate: %v", err)
	}

	if res == nil {
		t.Fatalf("Expected a response but got nil")
	}

	if res.Results == nil {
		t.Fatalf("expected response to have a non-nil Results fields")
	}

	if res.Results.Matches != 1 {
		t.Errorf("expected Matches to be 1, got %d", res.Results.Matches)
	}

	if res.Results.Successful != 1 {
		t.Errorf("expected Successful to be 1, got %d", res.Results.Successful)
	}
}

//
// func TestGetWeaviateEventByID(t *testing.T) {
// 	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
// 	defer os.Unsetenv("GO_ENV")
//
// 	// Save original environment variables
// 	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
// 	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
// 	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
// 	// ... add any other vars you need to save and restore
//
// 	hostAndPort := test_helpers.GetNextPort()
//
// 	parts := strings.Split(hostAndPort, ":")
// 	if len(parts) != 2 {
// 		t.Fatalf("Expected GetNextPort to return 'host:port', but got: %s", hostAndPort)
// 	}
//
// 	host := parts[0]
// 	port := parts[1]
//
// 	os.Setenv("WEAVIATE_HOST", host)
// 	os.Setenv("WEAVIATE_PORT", port)
// 	os.Setenv("WEAVIATE_SCHEME", "http")
// 	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")
//
// 	// Defer the cleanup to restore original env vars
// 	defer func() {
// 		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
// 		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
// 		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
// 		// ... restore other vars
// 	}()
//
// 	// Create and start the mock server for this test
// 	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// ... (Your mock response logic here) ...
// 	}))
// 	listenAddress := fmt.Sprintf("localhost:%s", port)
// 	listener, err := test_helpers.BindToPort(t, listenAddress)
// 	if err != nil {
// 		t.Fatalf("Failed to start mock Weaviate server for test: %v", err)
// 	}
// 	mockWeaviateServer.Listener = listener
// 	mockWeaviateServer.Start()
// 	defer mockWeaviateServer.Close()
//
// 	client, err := GetWeaviateClient()
// 	if err != nil {
// 		t.Fatalf("Failed to get Weaviate client: %v", err)
// 	}
//
// 	_, err = GetWeaviateEventByID(context.Background(), client, "123", "0")
// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}
// }
//
// func TestBulkGetWeaviateEventByID(t *testing.T) {
// 	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
// 	defer os.Unsetenv("GO_ENV")
// 	// Save original environment variables
// 	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
// 	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
// 	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
//
// 	hostAndPort := test_helpers.GetNextPort()
//
// 	parts := strings.Split(hostAndPort, ":")
// 	if len(parts) != 2 {
// 		t.Fatalf("Expected GetNextPort to return 'host:port', but got: %s", hostAndPort)
// 	}
//
// 	host := parts[0]
// 	port := parts[1]
//
// 	os.Setenv("WEAVIATE_HOST", host)
// 	os.Setenv("WEAVIATE_PORT", port)
// 	os.Setenv("WEAVIATE_SCHEME", "http")
// 	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")
//
// 	defer func() {
// 		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
// 		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
// 		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
// 		// ... restore other vars
// 	}()
//
// 	// Create and start the mock server for this test
// 	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// ... (Your mock response logic here) ...
// 	}))
// 	listenAddress := fmt.Sprintf("localhost:%s", port)
// 	listener, err := test_helpers.BindToPort(t, listenAddress)
// 	if err != nil {
// 		t.Fatalf("Failed to start mock Weaviate server for test: %v", err)
// 	}
// 	mockWeaviateServer.Listener = listener
// 	mockWeaviateServer.Start()
// 	defer mockWeaviateServer.Close()
//
// 	client, err := GetWeaviateClient()
// 	if err != nil {
// 		t.Fatalf("Failed to get Weaviate client: %v", err)
// 	}
//
// 	_, err = BulkGetWeaviateEventByID(context.Background(), client, []string{"123", "456"}, "0")
// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}
// }
