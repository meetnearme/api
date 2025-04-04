package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	// marqo-go is an unofficial Go client library for Marqo
	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestGetMarqoClient(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
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

	const eventId = "123"

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":            eventId,
					"eventOwners":    []interface{}{"789"},
					"eventOwnerName": "Event Host Test",
					"name":           "Test Event",
					"description":    "This is a test event",
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
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %f", err)
	}
	// we can't test the private variable `client.url`, so instead we make a mocked call
	// and if no error is thrown, we know the mock server responded on the configured URL
	_, err = GetMarqoEventByID(client, eventId, "0")
	if err != nil {
		t.Errorf("mocked endpoint not responding, error calling GetMarqoEventByID %f", err)
	}
}
func TestBulkUpsertEventToMarqo(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
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
		if authHeader == "" {
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
				{
					ID:     "789",
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

	loc, _ := time.LoadLocation("America/New_York")
	startTime1, err := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	if err != nil || startTime1 == 0 {
		t.Fatalf("failed to convert UTC to unix, %v", startTime1)
	}

	startTime2, err := helpers.UtcToUnix64("2099-06-01T14:00:00Z", loc)
	if err != nil || startTime2 == 0 {
		t.Fatalf("failed to convert UTC to unix, %v", startTime2)
	}

	startTime3, err := helpers.UtcToUnix64("2099-07-01T16:00:00Z", loc)
	if err != nil || startTime3 == 0 {
		t.Fatalf("failed to convert UTC to unix, %v", startTime3)
	}

	tests := []struct {
		name   string
		events []types.Event
	}{
		{
			name: "Multiple valid events",
			events: []types.Event{
				{
					EventOwners:    []string{"123"},
					EventOwnerName: "Event Host Test Name",
					Name:           "Test Event 1",
					Description:    "A test event 1",
					StartTime:      startTime1,
					Address:        "123 Test St",
					Lat:            51.5074,
					Long:           -0.1278,
					Timezone:       *loc,
				},
				{
					EventOwners:    []string{"456"},
					EventOwnerName: "Event Host Test Name",
					Name:           "Test Event 2",
					Description:    "A test event 2",
					StartTime:      startTime2,
					Address:        "456 Test Ave",
					Lat:            40.7128,
					Long:           -74.0060,
					Timezone:       *loc,
				},
				{
					EventOwners:    []string{"789"},
					EventOwnerName: "Event Host Test Name",
					Name:           "Test Event 3",
					Description:    "A test event 3",
					StartTime:      startTime3,
					Address:        "789 Test Blvd",
					Lat:            34.0522,
					Long:           -118.2437,
					Timezone:       *loc,
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

// NOTE: `calculateSearchBounds` is the function that actually calculates the bounds and
// this helper function should have no significant logic
func isPointInBounds(lat, long float64, minLat, maxLat, minLong, maxLong float64) bool {
	var inLatBounds bool = lat >= minLat && lat <= maxLat
	var inLongBounds bool = long >= minLong && long <= maxLong
	if inLatBounds && inLongBounds {
		return true
	}

	return false
}

func TestSearchMarqoEvents(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
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

	now := time.Now()
	testEvents := []map[string]interface{}{
		{
			"_id":            "1",
			"name":           "Today's Event",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"789"},
			"eventOwnerName": "Today Host",
			"description":    "Event happening today",
			"latitude":       51.5074, // Add coordinates
			"longitude":      -0.1278,
		},
		{
			"_id":            "2",
			"name":           "Next Week Event",
			"startTime":      now.AddDate(0, 0, 5).Unix(),
			"eventOwners":    []interface{}{"012"},
			"eventOwnerName": "Week Host",
			"description":    "Event happening next week",
			"latitude":       51.5074, // Add coordinates
			"longitude":      -0.1278,
		},
		{
			"_id":            "3",
			"name":           "Next Month Event",
			"startTime":      now.AddDate(0, 1, 0).Unix(),
			"eventOwners":    []interface{}{"345"},
			"eventOwnerName": "Month Host",
			"description":    "Event happening next month",
			"latitude":       51.5074, // Add coordinates
			"longitude":      -0.1278,
		},
		{
			"_id":            "4",
			"name":           "North Pole Event",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"678"},
			"eventOwnerName": "Polar Host",
			"description":    "Event near the North Pole",
			"latitude":       89.5,
			"longitude":      0.0,
		},
		{
			"_id":            "5",
			"name":           "International Date Line Event (East)",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"901"},
			"eventOwnerName": "Date Line Host",
			"description":    "Event East of date line",
			"latitude":       0.0,
			"longitude":      -179.9,
		},
		{
			"_id":            "6",
			"name":           "International Date Line Event (West)",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"389"},
			"eventOwnerName": "Date Line Host",
			"description":    "Event West of date line",
			"latitude":       0.0,
			"longitude":      179.9,
		},
		{
			"_id":            "7",
			"name":           "Prime Meridian Event (East)",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"251"},
			"eventOwnerName": "Date Line Host",
			"description":    "Event West of date line",
			"latitude":       10.0,
			"longitude":      -0.1,
		},
		{
			"_id":            "8",
			"name":           "Prime Meridian Event (West)",
			"startTime":      now.Unix(),
			"eventOwners":    []interface{}{"793"},
			"eventOwnerName": "Date Line Host",
			"description":    "Event East of date line",
			"latitude":       10.0,
			"longitude":      0.1,
		},
	}

	tests := []struct {
		name        string
		query       string
		startTime   int64
		endTime     int64
		location    []float64
		distance    float64
		expectedIds []string
	}{
		{
			name:        "Today's events",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{51.5074, -0.1278},
			distance:    10.0,
			expectedIds: []string{"1"},
		},
		{
			name:        "Near North Pole boundary",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{88.0, 0.0},
			distance:    200.0,
			expectedIds: []string{"4"},
		},
		{
			name:        "International Date Line wraparound, both events",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{0.0, -177.9},
			distance:    200.0,
			expectedIds: []string{"5", "6"},
		},
		{
			name:        "International Date Line wraparound, east event only",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{0.0, -179.9},
			distance:    5.0,
			expectedIds: []string{"5"},
		},
		{
			name:        "International Date Line wraparound, west event only",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{0.0, 179.9},
			distance:    5.0,
			expectedIds: []string{"6"},
		},
		{
			name:        "Prime Meridian wraparound, both events",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{10.0, -0.1},
			distance:    200.0,
			expectedIds: []string{"7", "8"},
		},
		{
			name:        "Prime Meridian wraparound, east event only",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{10.0, -0.1},
			distance:    5.0,
			expectedIds: []string{"7"},
		},
		{
			name:        "Prime Meridian wraparound, west event only",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{10.0, 0.1},
			distance:    5.0,
			expectedIds: []string{"8"},
		},
		{
			name:        "This week's events",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.AddDate(0, 0, 7).Unix(),
			location:    []float64{51.5074, -0.1278},
			distance:    100.0,
			expectedIds: []string{"1", "2"},
		},
		{
			name:        "This month's events",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.AddDate(0, 1, 0).Unix(),
			location:    []float64{51.5074, -0.1278},
			distance:    100.0,
			expectedIds: []string{"1", "2", "3"},
		},
		{
			name:        "Very large radius covers all longitudes",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{0.0, 0.0},
			distance:    12500.0,
			expectedIds: []string{"1", "4", "5", "6", "7", "8"},
		},
	}

	// Add a variable to track the current test case
	var currentTestIndex int

	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var searchRequest struct {
			Filter *string `json:"filter"`
		}
		if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
			t.Logf("Failed to decode request body: %v", err)
			http.Error(w, "Failed to decode request body", http.StatusBadRequest)
			return
		}

		filterStr := *searchRequest.Filter
		t.Logf("Received filter: %v", filterStr)

		// Extract time bounds
		startTimeStr := regexp.MustCompile(`startTime:\[(\d+) TO`).FindStringSubmatch(filterStr)
		endTimeStr := regexp.MustCompile(`TO (\d+)\]`).FindStringSubmatch(filterStr)

		var startTime, endTime int64
		if len(startTimeStr) > 1 {
			startTime, _ = strconv.ParseInt(startTimeStr[1], 10, 64)
		}
		if len(endTimeStr) > 1 {
			endTime, _ = strconv.ParseInt(endTimeStr[1], 10, 64)
		}

		// Get the current test case context
		currentTest := tests[currentTestIndex]

		// Calculate bounds using the same function as the main service
		minLat, maxLat, minLong1, maxLong1, minLong2, maxLong2, needsSplit := calculateSearchBounds(
			currentTest.location,
			currentTest.distance,
		)

		t.Logf("Calculated bounds: lat[%f TO %f], (long[%f TO %f] OR [%f TO %f]), needsSplit: %v", minLat, maxLat, minLong1, maxLong1, minLong2, maxLong2, needsSplit)

		// Filter events based on both time and location
		filteredEvents := []map[string]interface{}{}
		for _, event := range testEvents {
			eventTime := event["startTime"].(int64)
			eventLat := event["latitude"].(float64)
			eventLong := event["longitude"].(float64)

			inTimeRange := eventTime >= startTime && eventTime <= endTime
			inSpatialBound1 := isPointInBounds(eventLat, eventLong, minLat, maxLat, minLong1, maxLong1)
			inSpatialBound2 := isPointInBounds(eventLat, eventLong, minLat, maxLat, minLong2, maxLong2)

			log.Printf("event %s at [%f, %f] (time: %v, spatial bound 1: %v) || bounds: %v, %v, %v, %v", event["_id"], eventLat, eventLong, inTimeRange, inSpatialBound1, minLat, maxLat, minLong1, maxLong1)

			log.Printf("event %s at [%f, %f] (time: %v, spatial bound 2: %v) || bounds: %v, %v, %v, %v", event["_id"], eventLat, eventLong, inTimeRange, inSpatialBound2, minLat, maxLat, minLong2, maxLong2)

			if inTimeRange && (inSpatialBound1 || inSpatialBound2) {
				filteredEvents = append(filteredEvents, event)
				t.Logf("Including event %s at [%f, %f]", event["_id"], eventLat, eventLong)
			} else {
				t.Logf("Excluding event %s at [%f, %f] (time: %v, spatial: %v)",
					event["_id"], eventLat, eventLong, inTimeRange, inSpatialBound1)
			}
		}

		t.Logf("Returning %d filtered events", len(filteredEvents))

		response := map[string]interface{}{
			"hits": filteredEvents,
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			t.Logf("Failed to marshal response: %v", err)
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

	for i, tt := range tests {
		currentTestIndex = i // Update the current test index
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetMarqoClient()
			if err != nil {
				t.Fatalf("Failed to get Marqo client: %v", err)
			}

			result, err := SearchMarqoEvents(
				client,
				tt.query,
				tt.location, // Use location from test case
				tt.distance, // Use distance from test case
				tt.startTime,
				tt.endTime,
				[]string{},
				"",
				"",
				"0",
				[]string{helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[0]},
				[]string{},
			)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify the number of returned events
			if len(result.Events) != len(tt.expectedIds) {
				t.Errorf("Expected %d events, got %d", len(tt.expectedIds), len(result.Events))
			}

			// Verify the correct events were returned
			returnedIds := make([]string, len(result.Events))
			for i, event := range result.Events {
				returnedIds[i] = event.Id
			}
			for _, expectedId := range tt.expectedIds {
				found := false
				for _, returnedId := range returnedIds {
					if expectedId == returnedId {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected event with ID %s not found in results", expectedId)
					t.Errorf("Expected event with ID %s not found in results", expectedId)
				}
			}
		})
	}
}

// TODO: check for invalid / required fields, validate 404 response for non existent document
func TestGetMarqoEventByID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
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

	const (
		testEventID          = "123"
		testEventOwnerID     = "789"
		testEventName        = "Test Event"
		testEventDescription = "This is a test event"
	)

	loc, _ := time.LoadLocation("America/New_York")
	testEventStartTime, tmErr := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	if tmErr != nil || testEventStartTime == 0 {
		t.Logf("tmError converting UTC to unix: %v", tmErr)
	}
	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":            testEventID,
					"startTime":      testEventStartTime,
					"eventOwners":    []interface{}{testEventOwnerID},
					"eventOwnerName": "Event Host Test",
					"name":           testEventName,
					"description":    testEventDescription,
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
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %f", err)
	}

	event, err := GetMarqoEventByID(client, testEventID, "0")
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
		t.Errorf("expected event description %v, got %s", testEventStartTime, event.Description)
	}
}

func TestBulkGetMarqoEventByID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
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

	loc, _ := time.LoadLocation("America/New_York")
	testEventStartTime1, _err := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	if _err != nil {
		log.Printf("failed to convert UTC to unix 64, err: %v", _err)
	}

	testEventStartTime2, _err := helpers.UtcToUnix64("2099-06-01T14:00:00Z", loc)
	if _err != nil {
		log.Printf("failed to convert UTC to unix 64, err: %v", _err)
	}

	// NOTE: start times need to be generated from a helper to be human readable,
	// this is done above
	const (
		testEventID1          = "123"
		testEventID2          = "456"
		testEventOwnerName1   = "Test Owner 1"
		testEventOwnerName2   = "Test Owner 2"
		testEventOwnerID1     = "789"
		testEventOwnerID2     = "012"
		testEventName1        = "Test Event 1"
		testEventName2        = "Test Event 2"
		testEventDescription1 = "This is test event 1"
		testEventDescription2 = "This is test event 2"
	)

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response

		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":            testEventID1,
					"startTime":      testEventStartTime1,
					"eventOwners":    []interface{}{testEventOwnerID1},
					"eventOwnerName": testEventOwnerName1,
					"name":           testEventName1,
					"description":    testEventDescription1,
				},
				{
					"_id":            testEventID2,
					"startTime":      testEventStartTime2,
					"eventOwners":    []interface{}{testEventOwnerID2},
					"eventOwnerName": testEventOwnerName2,
					"name":           testEventName2,
					"description":    testEventDescription2,
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
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Call the handler
	client, err := GetMarqoClient()
	if err != nil {
		t.Errorf("error getting marqo client %v", err)
	}

	events, err := BulkGetMarqoEventByID(client, []string{testEventID1, testEventID2}, "0")
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
		ownerName   string
		name        string
		description string
		startTime   int64
	}{
		{testEventID1, testEventOwnerID1, testEventOwnerName1, testEventName1, testEventDescription1, testEventStartTime1},
		{testEventID2, testEventOwnerID2, testEventOwnerName2, testEventName2, testEventDescription2, testEventStartTime2},
	}

	for i, expectedEvent := range expectedEvents {
		event := events[i]
		if event.Id != expectedEvent.id {
			t.Errorf("expected event ID %s, got %s", expectedEvent.id, event.Id)
		}
		if len(event.EventOwners) != 1 || event.EventOwners[0] != expectedEvent.ownerID {
			t.Errorf("expected event owner %s, got %v", expectedEvent.ownerID, event.EventOwners)
		}
		if event.EventOwnerName != expectedEvent.ownerName {
			t.Errorf("expected event name %s, got %s", expectedEvent.name, event.Name)
		}
		if event.Name != expectedEvent.name {
			t.Errorf("expected event name %s, got %s", expectedEvent.name, event.Name)
		}
		if event.Description != expectedEvent.description {
			t.Errorf("expected event description %s, got %s", expectedEvent.description, event.Description)
		}
		if event.StartTime != expectedEvent.startTime {
			t.Errorf("expected event start time %v, got %v", expectedEvent.startTime, event.StartTime)
		}
	}
}
