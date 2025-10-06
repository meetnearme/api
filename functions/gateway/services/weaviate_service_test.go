package services

import (
	"context"
	"encoding/json"
	"fmt"
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
		switch r.URL.Path {
		case "/v1/.well-known/ready":
			w.WriteHeader(http.StatusOK)
			return

		case "/v1/meta":
			if r.Method != "GET" {
				t.Errorf("expected method GET for /v1/meta, got %s", r.Method)
			}
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/graphql":
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						"Test": []interface{}{
							map[string]interface{}{
								"name": "Test Object",
								"_additional": map[string]interface{}{
									"id": "test-id-123",
								},
							},
						},
					},
				},
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
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

	// Update environment variables to match the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	defer mockWeaviateServer.Close()

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
		switch r.URL.Path {

		case "/v1/meta":
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/batch/objects":
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
		default:
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	loc, _ := time.LoadLocation("America/New_York")
	startTime1, err := helpers.UtcToUnix64WithTrimZ("2099-05-01T12:00:00Z", loc, false)
	if err != nil {
		t.Fatalf("failed to convert time: %v", err)
	}

	events := []types.Event{
		{
			Id:          "00000000-0000-0000-0000-000000000001",
			EventOwners: []string{"123"},
			Name:        "Test Event 1",
			StartTime:   startTime1,
			SourceUrl:   "https://example.com/event-source",
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

	className := helpers.WeaviateEventClassName

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

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						// The key here should match the Class name your code expects.
						helpers.WeaviateEventClassName: []interface{}{
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

func TestGetWeaviateEventByID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

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

	className := helpers.WeaviateEventClassName

	// Defer the cleanup to restore original env vars
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
	}()

	const expectedID = "00000000-0000-0000-0000-000000000123"
	const expectedName = "Concert at the Park"

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {

		case "/v1/meta":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version":"1.24.1"}`))

		case "/v1/graphql":
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						// The key is the Class Name your code is querying
						helpers.WeaviateEventClassName: []interface{}{
							// This map represents the single event object found
							map[string]interface{}{
								"name":        expectedName,
								"description": "An outdoor concert for testing.",
								"startTime":   time.Now().Unix(),
								"timezone":    "America/New_York",
								"_additional": map[string]interface{}{
									"id": strfmt.UUID(expectedID),
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

	// Call the function we want to test
	res, err := GetWeaviateEventByID(context.Background(), client, expectedID, "0")

	// --- Assertions ---
	if err != nil {
		t.Errorf("Unexpected error from GetWeaviateEventByID: %v", err)
	}
	if res == nil {
		t.Fatalf("Expected an event response but got nil")
	}

	// Check that the returned event has the correct data from our mock.
	if res.Id != expectedID {
		t.Errorf("Expected event ID '%s', but got '%s'", expectedID, res.Id)
	}
	if res.Name != expectedName {
		t.Errorf("Expected event name '%s', but got '%s'", expectedName, res.Name)
	}
}

func TestBulkGetWeaviateEventByID(t *testing.T) {
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
	if len(parts) != 2 {
		t.Fatalf("Expected GetNextPort to return 'host:port', but got: %s", hostAndPort)
	}
	host, port := parts[0], parts[1]

	os.Setenv("WEAVIATE_HOST", host)
	os.Setenv("WEAVIATE_PORT", port)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	className := helpers.WeaviateEventClassName

	// --- Mock Server Logic for Bulk Get By ID ---
	idsToFetch := []string{
		"00000000-0000-0000-0000-000000000123",
		"00000000-0000-0000-0000-000000000456",
	}

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A "Bulk Get by ID" uses a GraphQL query.
		switch r.URL.Path {
		case "/v1/meta":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version":"1.24.1"}`))

		case "/v1/graphql":
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Create a canned successful response containing multiple events.
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						// The key is the Class Name
						helpers.WeaviateEventClassName: []interface{}{
							// First event object
							map[string]interface{}{
								"name":        "First Mock Event",
								"description": "This is the first event.",
								"timezone":    "America/Denver",
								"sourceUrl":   "https://example.com/event/1",
								"_additional": map[string]interface{}{
									"id": strfmt.UUID(idsToFetch[0]),
								},
							},
							// Second event object
							map[string]interface{}{
								"name":        "Second Mock Event",
								"description": "This is the second event.",
								"timezone":    "America/Denver",
								"sourceUrl":   "https://example.com/event/2",
								"_additional": map[string]interface{}{
									"id": strfmt.UUID(idsToFetch[1]),
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

	listenAddress := fmt.Sprintf("localhost:%s", port)
	listener, err := test_helpers.BindToPort(t, listenAddress)
	if err != nil {
		t.Fatalf("Failed to start mock Weaviate server for test: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// --- Test Execution ---
	client, err := GetWeaviateClient()
	if err != nil {
		t.Fatalf("Failed to get Weaviate client: %v", err)
	}

	res, err := BulkGetWeaviateEventByID(context.Background(), client, idsToFetch, "0")

	// --- Assertions ---
	if err != nil {
		t.Errorf("Unexpected error from BulkGetWeaviateEventByID: %v", err)
	}
	if res == nil {
		t.Fatalf("Expected a slice of events but got nil")
	}

	// Check that the correct number of events were returned.
	if len(res) != 2 {
		t.Fatalf("Expected 2 events in the response, but got %d", len(res))
	}

	// Check the details of each returned event.
	if res[0].Id != idsToFetch[0] {
		t.Errorf("Expected first event ID to be %s, but got %s", idsToFetch[0], res[0].Id)
	}
	if res[0].Name != "First Mock Event" {
		t.Errorf("Expected first event name to be 'First Mock Event', but got '%s'", res[0].Name)
	}
	if res[0].SourceUrl != "https://example.com/event/1" {
		t.Errorf("Expected first event sourceUrl to be 'https://example.com/event/1', but got '%s'", res[0].SourceUrl)
	}

	if res[1].Id != idsToFetch[1] {
		t.Errorf("Expected second event ID to be %s, but got %s", idsToFetch[1], res[1].Id)
	}
	if res[1].Name != "Second Mock Event" {
		t.Errorf("Expected second event name to be 'Second Mock Event', but got '%s'", res[1].Name)
	}
	if res[1].SourceUrl != "https://example.com/event/2" {
		t.Errorf("Expected second event sourceUrl to be 'https://example.com/event/2', but got '%s'", res[1].SourceUrl)
	}
}
