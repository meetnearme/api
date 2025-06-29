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

	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	// ... add any other vars you need to save and restore

	// Get a unique port for THIS test
	port := test_helpers.GetNextPort()

	// Set the environment variables correctly
	os.Setenv("WEAVIATE_HOST", "localhost")
	os.Setenv("WEAVIATE_PORT", string(port))
	os.Setenv("WEAVIATE_SCHEME", "http")

	// Defer the cleanup to restore original env vars
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		// ... restore other vars
	}()

	// Create and start the mock server for this test
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... (Your mock response logic here) ...
	}))
	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
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

	_, err = SearchWeaviateEvents(context.Background(), client, "test", []float64{0, 0}, 0, 0, 0, []string{}, "", "", "", []string{}, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestBulkDeleteEventsFromWeaviate(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	// ... add any other vars you need to save and restore

	// Get a unique port for THIS test
	port := test_helpers.GetNextPort()

	// Set the environment variables correctly
	os.Setenv("WEAVIATE_HOST", "localhost")
	os.Setenv("WEAVIATE_PORT", string(port))
	os.Setenv("WEAVIATE_SCHEME", "http")

	// Defer the cleanup to restore original env vars
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		// ... restore other vars
	}()

	// Create and start the mock server for this test
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... (Your mock response logic here) ...
	}))
	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
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

	_, err = BulkDeleteEventsFromWeaviate(context.Background(), client, []string{"123"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetWeaviateEventByID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	// ... add any other vars you need to save and restore

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

	// Defer the cleanup to restore original env vars
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		// ... restore other vars
	}()

	// Create and start the mock server for this test
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... (Your mock response logic here) ...
	}))
	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
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

	_, err = GetWeaviateEventByID(context.Background(), client, "123", "0")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestBulkGetWeaviateEventByID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")

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

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		// ... restore other vars
	}()

	// Create and start the mock server for this test
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... (Your mock response logic here) ...
	}))
	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
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

	_, err = BulkGetWeaviateEventByID(context.Background(), client, []string{"123", "456"}, "0")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
