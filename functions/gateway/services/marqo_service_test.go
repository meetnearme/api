package services

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	// marqo-go is an unofficial Go client library for Marqo
	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
)

func TestGetMarqoClient(t *testing.T){
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

	const eventId = "123"

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":          eventId,
					"eventOwners": []interface{}{"789"},
					"name":        "Test Event",
					"description": "This is a test event",
				},
			},
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

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %f", err)
	}
	// we can't test the private variable `client.url`, so instead we make a mocked call
	// and if no error is thrown, we know the mock server responded on the configured URL
	_, err = GetMarqoEventByID(client, eventId)
	if err != nil {
		t.Errorf("mocked endpoint not responding, error calling GetMarqoEventByID %f", err)
	}
}

func TestUpsertEventToMarqo(t *testing.T) {
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
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	tests := []struct {
		name string
		event Event
	}{
		{
			name: "Valid event",
			event: Event{
				EventOwners: []string{"123"},
				Name:        "Test Event",
				Description: "A test event",
				StartTime:   "2099-05-01T12:00:00Z",
				Address:     "123 Test St",
				Lat:         51.5074,
				Long:        -0.1278,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetMarqoClient()
			if err != nil {
				t.Fatalf("Failed to get Marqo client: %v", err)
			}

			res, err := UpsertEventToMarqo(client, tt.event)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if res == nil {
				t.Errorf("Expected non-nil response, got nil")
			} else {
				if res.Errors {
					t.Errorf("Expected no errors, but got errors in response")
				}
				if len(res.Items) == 0 {
					t.Errorf("Expected non-empty Items array, got empty array")
				}
				if res.Items[0].ID != "123" {
					t.Errorf("Expected ID '123', got '%s'", res.Items[0].ID)
				}
			}
		})
	}
}

func TestBulkUpsertEventToMarqo(t *testing.T) {
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
				{
					ID: "456",
					Result: "",
					Status: 200,
				},
				{
					ID: "789",
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
		name   string
		events []Event
	}{
		{
			name: "Multiple valid events",
			events: []Event{
				{
					EventOwners: []string{"123"},
					Name:        "Test Event 1",
					Description: "A test event 1",
					StartTime:   "2099-05-01T12:00:00Z",
					Address:     "123 Test St",
					Lat:         51.5074,
					Long:        -0.1278,
				},
				{
					EventOwners: []string{"456"},
					Name:        "Test Event 2",
					Description: "A test event 2",
					StartTime:   "2099-06-01T14:00:00Z",
					Address:     "456 Test Ave",
					Lat:         40.7128,
					Long:        -74.0060,
				},
				{
					EventOwners: []string{"789"},
					Name:        "Test Event 3",
					Description: "A test event 3",
					StartTime:   "2099-07-01T16:00:00Z",
					Address:     "789 Test Blvd",
					Lat:         34.0522,
					Long:        -118.2437,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetMarqoClient()
			if err != nil {
				t.Fatalf("Failed to get Marqo client: %v", err)
			}

			res, err := BulkUpsertEventToMarqo(client, tt.events)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if res == nil {
				t.Errorf("Expected non-nil response, got nil")
			} else {
				if res.Errors {
					t.Errorf("Expected no errors, but got errors in response")
				}
				if len(res.Items) != len(tt.events) {
					t.Errorf("Expected %d items, got %d", len(tt.events), len(res.Items))
				}
				expectedIDs := []string{"123", "456", "789"}
				for i, item := range res.Items {
					if item.ID != expectedIDs[i] {
						t.Errorf("Expected ID '%s', got '%s'", expectedIDs[i], item.ID)
					}
				}
			}
		})
	}
}

func TestSearchMarqoEvents(t *testing.T) {
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
		query := r.URL.Query().Get("q")
		decodedQuery, err := url.QueryUnescape(query)
		if err != nil {
			http.Error(w, "Failed to decode query", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"hits": []map[string]interface{}{
				{
					"_id":         "123",
					"eventOwners": []interface{}{"789"},
					"name":        "First Test Event",
					"description": "Description of the first event",
				},
				{
					"_id":         "456",
					"eventOwners": []interface{}{"012"},
					"name":        "Second Test Event",
					"description": "Description of the second event",
				},
			},
			"query": decodedQuery,
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
		query          string
		userLocation   []float64
		maxDistance    float64
		ownerIds       []string
		expectedEvents int
		expectedError  bool
	}{
		{
			name:           "Valid search",
			query:          "test search",
			userLocation:   []float64{51.5074, -0.1278},
			maxDistance:    10000,
			ownerIds:       []string{},
			expectedEvents: 2,
			expectedError:  false,
		},
		{
			name:           "Empty query",
			query:          "",
			userLocation:   []float64{51.5074, -0.1278},
			maxDistance:    10000,
			ownerIds:       []string{},
			expectedEvents: 2,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetMarqoClient()
			if err != nil {
				t.Fatalf("Failed to get Marqo client: %v", err)
			}

			result, err := SearchMarqoEvents(client, tt.query, tt.userLocation, tt.maxDistance, tt.ownerIds)

			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}

			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(result.Events) != tt.expectedEvents {
				t.Errorf("Expected %d events, but got %d", tt.expectedEvents, len(result.Events))
			}

			if result.Query != tt.query {
				t.Errorf("Expected query to be '%s', but got '%s'", tt.query, result.Query)
			}

			// Add more specific checks for the returned events if needed
		})
	}
}

func TestGetMarqoEventByID(t *testing.T) {
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

	const (
		testEventID          = "123"
		testEventOwnerID     = "789"
		testEventName        = "Test Event"
		testEventDescription = "This is a test event"
		testEventStartTime        = "2099-05-01T12:00:00Z"
	)

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":          testEventID,
					"startTime":   testEventStartTime,
					"eventOwners": []interface{}{testEventOwnerID},
					"name":        testEventName,
					"description": testEventDescription,
				},
			},
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

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %f", err)
	}

	event, err := GetMarqoEventByID(client, testEventID)
	if err != nil {
		t.Errorf("mocked endpoint not responding, error calling GetMarqoEventByID %v", err)
	}

	// Check event properties
	if event.Id != testEventID {
		t.Errorf("expected event ID %s, got %s", testEventID, event.Id)
	}
	if len(event.EventOwners) != 1 || event.EventOwners[0] != testEventOwnerID {
		t.Errorf("expected event owner %s, got %v", testEventOwnerID, event.EventOwners)
	}
	if event.Name != testEventName {
		t.Errorf("expected event name %s, got %s", testEventName, event.Name)
	}
	if event.Description != testEventDescription {
		t.Errorf("expected event description %s, got %s", testEventDescription, event.Description)
	}
	if event.StartTime != testEventStartTime {
		t.Errorf("expected event description %s, got %s", testEventStartTime, event.Description)
	}
}

func TestBulkGetMarqoEventByID(t *testing.T) {
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

	const (
		testEventID1          = "123"
		testEventID2          = "456"
		testEventOwnerID1     = "789"
		testEventOwnerID2     = "012"
		testEventName1        = "Test Event 1"
		testEventName2        = "Test Event 2"
		testEventDescription1 = "This is test event 1"
		testEventDescription2 = "This is test event 2"
		testEventStartTime1   = "2099-05-01T12:00:00Z"
		testEventStartTime2   = "2099-06-01T14:00:00Z"
	)

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":          testEventID1,
					"startTime":    testEventStartTime1,
					"eventOwners":  []interface{}{testEventOwnerID1},
					"name":         testEventName1,
					"description":  testEventDescription1,
				},
				{
					"_id":          testEventID2,
					"startTime":    testEventStartTime2,
					"eventOwners":  []interface{}{testEventOwnerID2},
					"name":         testEventName2,
					"description":  testEventDescription2,
				},
			},
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

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %v", err)
	}

	events, err := BulkGetMarqoEventByID(client, []string{testEventID1, testEventID2})
	if err != nil {
		t.Errorf("mocked endpoint not responding, error calling BulkGetMarqoEventByID %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Check properties for both events
	expectedEvents := []struct {
		id          string
		ownerID     string
		name        string
		description string
		startTime   string
	}{
		{testEventID1, testEventOwnerID1, testEventName1, testEventDescription1, testEventStartTime1},
		{testEventID2, testEventOwnerID2, testEventName2, testEventDescription2, testEventStartTime2},
	}

	for i, expectedEvent := range expectedEvents {
		event := events[i]
		if event.Id != expectedEvent.id {
			t.Errorf("expected event ID %s, got %s", expectedEvent.id, event.Id)
		}
		if len(event.EventOwners) != 1 || event.EventOwners[0] != expectedEvent.ownerID {
			t.Errorf("expected event owner %s, got %v", expectedEvent.ownerID, event.EventOwners)
		}
		if event.Name != expectedEvent.name {
			t.Errorf("expected event name %s, got %s", expectedEvent.name, event.Name)
		}
		if event.Description != expectedEvent.description {
			t.Errorf("expected event description %s, got %s", expectedEvent.description, event.Description)
		}
		if event.StartTime != expectedEvent.startTime {
			t.Errorf("expected event start time %s, got %s", expectedEvent.startTime, event.StartTime)
		}
	}
}