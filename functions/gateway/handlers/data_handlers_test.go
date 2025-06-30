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
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	// internal_types "github.com/meetnearme/api/functions/gateway/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var searchUsersByIDs = helpers.SearchUsersByIDs

func init() {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
}

func TestPostEventHandler(t *testing.T) {
	t.Run("Valid event posts successfully", func(t *testing.T) {
		eventID := uuid.New().String()

		payloadStruct := rawEvent{
			rawEventData: rawEventData{
				Id:              eventID,
				EventOwnerName:  "Event Owner",
				EventOwners:     []string{"123"},
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Name:            "Test Event",
				Description:     "A test event",
				Address:         "123 Test St",
				Lat:             51.5074,
				Long:            -0.1278,
				Timezone:        "America/New_York",
			},
			StartTime: "2099-05-01T12:00:00Z",
		}

		requestBody, err := json.Marshal(payloadStruct)
		if err != nil {
			t.Fatalf("Failed to marshal test event to JSON: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/event", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handlerFunc := PostEventHandler(rr, req)
		handlerFunc(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		event, err := services.GetWeaviateEventByID(context.Background(), testClient, eventID, "0")
		if err != nil {
			t.Fatalf("Failed to get event from Weaviate for verification: %v", err)
		}
		if event == nil {
			t.Fatal("Event was not found in Weaviate after handler ran")
		}
		if event.Name != "Test Event" {
			t.Errorf("Expected event name to be 'Test Event', but got '%s'", event.Name)
		}
	})

	// The test table structure is excellent and we can keep it.
	failureTests := []struct {
		name              string
		requestBody       string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
		dbAssertionCheck  func(t *testing.T) // New function to check DB state
	}{
		{
			name:           "Invalid JSON",
			requestBody:    `{"name":"Test Event","description":}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(strings.ToLower(body), "invalid json payload") {
					t.Errorf("Expected body to contain 'invalid json payload', but got '%s'", body)
				}
			},
			dbAssertionCheck: nil, // No DB check needed for a validation failure
		},
		{
			name:           "Missing required name field",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"SINGLE_EVENT","startTime":"2099-05-01T12:00:00Z","description":"A test event","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					t.Errorf("Expected body to contain name validation error, but got '%s'", body)
				}
			},
			dbAssertionCheck: nil,
		},
		{
			name:           "Missing required startTime field",
			requestBody:    `{"description":"A test event", "eventOwnerName": "Event Owner", "eventOwners":["123"],"name":"Test Event","eventSourceType": "` + helpers.ES_SINGLE_EVENT + `","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'StartTime' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
			dbAssertionCheck: nil, // No DB interaction is expected
		},
		{
			name:           "Missing required name field",
			requestBody:    `{"eventOwnerName": "Event Owner", "eventOwners":["123"],"eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","startTime":"2099-05-01T12:00:00Z","description":"A test event","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'Name' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
			dbAssertionCheck: nil,
		},
		{
			name:           "Missing required eventOwners field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278,"timezone":"America/New_York"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'EventOwners' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
			dbAssertionCheck: nil,
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
			dbAssertionCheck: nil,
		},
		{
			name:           "Missing required timezone field",
			requestBody:    `{"eventOwnerName":"Event Owner","eventOwners":["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "Field validation for 'Timezone' failed on the 'required' tag"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
			dbAssertionCheck: nil,
		},
		{
			name:           "Invalid timezone field",
			requestBody:    `{"timezone":"Does_Not_Exist/Nowhere","eventOwnerName":"Event Owner","eventOwners":["123"], "eventSourceType":"` + helpers.ES_SINGLE_EVENT + `","name":"Test Event","description":"A test event","startTime":"2099-05-01T12:00:00Z","address":"123 Test St","lat":51.5074,"long":-0.1278}`,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				expectedSubstring := "invalid timezone: unknown time zone Does_Not_Exist/Nowhere"
				if !strings.Contains(body, expectedSubstring) {
					t.Errorf("Expected response body to contain '%s', but got '%s'", expectedSubstring, body)
				}
			},
			dbAssertionCheck: nil,
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE: Create a new HTTP request with the test case body.
			req := httptest.NewRequest("POST", "/api/event", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Create a ResponseRecorder to capture the handler's response.
			rr := httptest.NewRecorder()

			// ACT: Call your handler factory directly. This simulates the router calling it.
			// This directly tests your current application pattern.
			handlerFunc := PostEventHandler(rr, req)
			handlerFunc(rr, req) // Execute the handler that the factory returns

			// ASSERT HTTP RESPONSE
			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}

			// ASSERT DATABASE STATE
			if tt.dbAssertionCheck != nil {
				tt.dbAssertionCheck(t)
			}
		})
	}
}

// Need to move these to Weaviate

func TestPostBatchEvents(t *testing.T) {
	validEventID1 := uuid.New().String()
	validEventID2 := uuid.New().String()

	validPayload := struct {
		Events []rawEvent `json:"events"`
	}{
		Events: []rawEvent{
			createValidRawEvent(validEventID1, "Valid Batch Event 1"),
			createValidRawEvent(validEventID2, "Valid Batch Event 2"),
		},
	}
	validRequestBody, err := json.Marshal(validPayload)
	if err != nil {
		t.Fatalf("Setup failed: Could not marshal valid request body: %v", err)
	}

	invalidPayloadEvent1 := createValidRawEvent(uuid.New().String(), "This event is valid")
	invalidPayloadEvent2 := createValidRawEvent(uuid.New().String(), "This event has no name")
	invalidPayloadEvent2.Name = ""

	partiallyInvalidPayload := struct {
		Events []rawEvent `json:"events"`
	}{
		Events: []rawEvent{invalidPayloadEvent1, invalidPayloadEvent2},
	}
	partiallyInvalidRequestBody, err := json.Marshal(partiallyInvalidPayload)
	if err != nil {
		t.Fatalf("Setup failed: Could not marshal partially invalid request body: %v", err)
	}

	tests := []struct {
		name              string
		requestBody       string
		idsToCleanup      []string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
		dbAssertionCheck  func(t *testing.T)
	}{
		{
			name:           "Valid batch of events posts successfully",
			requestBody:    string(validRequestBody), // Use the body we just created
			idsToCleanup:   []string{validEventID1, validEventID2},
			expectedStatus: http.StatusCreated,
			dbAssertionCheck: func(t *testing.T) {
				for _, id := range []string{validEventID1, validEventID2} {
					event, err := services.GetWeaviateEventByID(context.Background(), testClient, id, "0")
					if err != nil {
						t.Errorf("Failed to get event '%s' for verification: %v", id, err)
						continue
					}
					if event == nil {
						t.Errorf("Event '%s' was not found after handler ran", id)
					}
				}
			},
		},
		{
			name:           "Batch with one invalid event fails validation",
			requestBody:    string(partiallyInvalidRequestBody), // Use the body we just created
			idsToCleanup:   []string{invalidPayloadEvent1.Id},   // The valid one might get created if logic is wrong
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				// Now we can be sure the error is for the 'Name' field on the second event.
				expectedErr := "invalid event at index 1: Field validation for 'Name' failed on the 'required' tag"
				if !strings.Contains(body, expectedErr) {
					t.Errorf("Expected validation error '%s', but got '%s'", expectedErr, body)
				}
			},
			dbAssertionCheck: func(t *testing.T) {
				// Verify that NO events from the failed batch were committed
				event, err := services.GetWeaviateEventByID(context.Background(), testClient, invalidPayloadEvent1.Id, "0")
				if err == nil {
					t.Fatalf("Expected an error when checking for non-existent event, but received no error and found event: %v", err)
				}
				if event != nil {
					t.Error("An event from a failed batch was incorrectly saved to the database")
				}
				if !strings.Contains(err.Error(), "no event found") {
					t.Errorf("Expected a 'not found' error, but got a different database error: %v", err)
				}
			},
		},
		{
			name:           "Invalid JSON payload",
			requestBody:    `{"events":[{"name":"Test Event","description":}]}`,
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if len(tt.idsToCleanup) > 0 {
					_, err := services.BulkDeleteEventsFromWeaviate(context.Background(), testClient, tt.idsToCleanup)
					if err != nil {
						t.Errorf("ERROR in test cleanup: %v", err)
					}
				}
			}()

			req := httptest.NewRequest("POST", "/api/events/batch", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handlerFunc := PostBatchEventsHandler(rr, req)
			handlerFunc(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
			if tt.dbAssertionCheck != nil {
				tt.dbAssertionCheck(t)
			}
		})
	}
}

func createValidRawEvent(id, name string) rawEvent {
	return rawEvent{
		rawEventData: rawEventData{
			Id:              id,
			EventOwners:     []string{"owner-123"},
			EventOwnerName:  "Test Owner",
			EventSourceType: helpers.ES_SINGLE_EVENT,
			Name:            name,
			Description:     "A valid test event description.",
			Address:         "123 Test St, Testville",
			Lat:             40.1,
			Long:            -74.1,
			Timezone:        "America/New_York",
		},
		StartTime: "2099-10-10T10:00:00Z",
	}
}

func TestSearchEvents(t *testing.T) {
	// === ARRANGE: SETUP FOR ALL TEST CASES ===
	// 1. Generate the dynamic UUIDs for our test cases BEFORE the test table.
	event1ID := uuid.New().String()
	event2ID := uuid.New().String()

	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("SETUP FAILED: could not load timezone: %v", err)
	}

	// 2. Define the test data using the generated UUIDs.
	eventsToCreate := []types.Event{
		{
			Id:              event1ID, // Using the pre-generated UUID
			EventOwners:     []string{"owner-123"},
			EventOwnerName:  "Tech Org",
			EventSourceType: helpers.ES_SINGLE_EVENT,
			Name:            "Conference on Go Programming",
			Description:     "A deep dive into the Go language and its powerful ecosystem.",
			StartTime:       time.Now().Add(48 * time.Hour).Unix(),
			Address:         "123 Tech Way, Silicon Valley, CA",
			Lat:             37.3861,
			Long:            -122.0839,
			Timezone:        *tz,
		},
		{
			Id:              event2ID, // Using the pre-generated UUID
			EventOwners:     []string{"owner-456"},
			EventOwnerName:  "Music Planners Inc.",
			EventSourceType: helpers.ES_SINGLE_EVENT,
			Name:            "Local Music Festival",
			Description:     "Enjoy live bands and great food by the scenic waterfront.",
			StartTime:       time.Now().Add(72 * time.Hour).Unix(),
			Address:         "456 Melody Lane, Austin, TX",
			Lat:             30.2672,
			Long:            -97.7431,
			Timezone:        *tz,
		},
	}

	// --- Define the Test Table ---
	tests := []struct {
		name              string
		path              string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
	}{
		{
			name:           "Search with specific term finds correct event",
			path:           "/events?q=programming", // This should only match the tech conference
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				var res types.EventSearchResponse
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(res.Events) != 1 {
					t.Fatalf("Expected to find 1 event, but got %d", len(res.Events))
				}

				// ASSERT: The assertion check now uses the pre-defined variable.
				if res.Events[0].Id != event1ID {
					t.Errorf("Expected to find event '%s', but got '%s'", event1ID, res.Events[0].Id)
				}
			},
		},
		{
			name:           "Search for term with no matches returns empty list",
			path:           "/events?q=nonexistenttermxyz",
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				// This test doesn't need to check against dynamic data.
				var res types.EventSearchResponse
				if err := json.Unmarshal([]byte(body), &res); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if len(res.Events) != 0 {
					t.Errorf("Expected 0 events for a nonexistent term, but got %d", len(res.Events))
				}
			},
		},
	}

	// --- The Test Runner ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE & SEED: Use your service function to insert the data.
			// This happens before every test to ensure a clean state.
			_, err := services.BulkUpsertEventsToWeaviate(context.Background(), testClient, eventsToCreate)
			if err != nil {
				t.Fatalf("DB setup failed for test '%s': %v", tt.name, err)
			}

			// CLEANUP: Defer the deletion of test data.
			defer func() {
				var idsToCleanup []string
				for _, event := range eventsToCreate {
					idsToCleanup = append(idsToCleanup, event.Id)
				}
				_, err := services.BulkDeleteEventsFromWeaviate(context.Background(), testClient, idsToCleanup)
				if err != nil {
					t.Errorf("ERROR in test cleanup for '%s': %v", tt.name, err)
				}
			}()

			// Give Weaviate a moment to index after seeding.
			time.Sleep(2 * time.Second)

			// ACT: Perform the HTTP request.
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler := SearchEventsHandler(rr, req)
			handler(rr, req)

			// ASSERT: Check the results.
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
		})
	}
}

// func TestSearchEvents(t *testing.T) {
// 	tz, err := time.LoadLocation("America/Los_Angeles")
// 	if err != nil {
// 		t.Fatalf("SETUP FAILED: could not load timezone: %v", err)
// 	}
//
// 	tests := []struct {
// 		name              string
// 		path              string
// 		dbSeeder          func(t *testing.T) // Function to seed data for this specific test.
// 		idsToCleanup      []string           // IDs to delete after this specific test.
// 		expectedStatus    int
// 		expectedBodyCheck func(t *testing.T, body string)
// 	}{
// 		{
// 			name: "Search with specific term finds correct event",
// 			path: "/events?q=programming", // This should only match the tech conference
// 			dbSeeder: func(t *testing.T) {
// 				// This test needs specific, valid data in the DB to find.
// 				eventsToCreate := []types.Event{
// 					{
// 						Id:              "search-1",
// 						EventOwners:     []string{"owner-123"},
// 						EventOwnerName:  "Tech Org",
// 						EventSourceType: helpers.ES_SINGLE_EVENT,
// 						Name:            "Conference on Go Programming",
// 						Description:     "A deep dive into the Go language and its powerful ecosystem.",
// 						StartTime:       time.Now().Add(48 * time.Hour).Unix(),
// 						Address:         "123 Tech Way, Silicon Valley, CA",
// 						Lat:             37.3861,
// 						Long:            -122.0839,
// 						Timezone:        *tz,
// 					},
// 					{
// 						Id:              "search-2",
// 						EventOwners:     []string{"owner-456"},
// 						EventOwnerName:  "Music Planners Inc.",
// 						EventSourceType: helpers.ES_SINGLE_EVENT,
// 						Name:            "Local Music Festival",
// 						Description:     "Enjoy live bands and great food by the scenic waterfront.",
// 						StartTime:       time.Now().Add(72 * time.Hour).Unix(),
// 						Address:         "456 Melody Lane, Austin, TX",
// 						Lat:             30.2672,
// 						Long:            -97.7431,
// 						Timezone:        *tz,
// 					},
// 				}
// 				_, err := services.BulkUpsertEventsToWeaviate(context.Background(), testClient, eventsToCreate)
// 				if err != nil {
// 					t.Fatalf("DB seeder failed: %v", err)
// 				}
// 			},
// 			idsToCleanup:   []string{"search-1", "search-2"},
// 			expectedStatus: http.StatusOK,
// 			expectedBodyCheck: func(t *testing.T, body string) {
// 				var res types.EventSearchResponse
// 				if err := json.Unmarshal([]byte(body), &res); err != nil {
// 					t.Fatalf("Failed to unmarshal response body: %v", err)
// 				}
// 				if len(res.Events) != 1 {
// 					t.Fatalf("Expected to find 1 event, but got %d", len(res.Events))
// 				}
// 				if res.Events[0].Id != "search-1" {
// 					t.Errorf("Expected to find event 'search-1', but got '%s'", res.Events[0].Id)
// 				}
// 			},
// 		},
// 		{
// 			name:           "Search for term with no matches returns empty list",
// 			path:           "/events?q=nonexistenttermxyz",
// 			dbSeeder:       nil, // No data needs to exist for this test.
// 			idsToCleanup:   nil, // Nothing to clean up.
// 			expectedStatus: http.StatusOK,
// 			expectedBodyCheck: func(t *testing.T, body string) {
// 				var res types.EventSearchResponse
// 				if err := json.Unmarshal([]byte(body), &res); err != nil {
// 					t.Fatalf("Failed to unmarshal response body: %v", err)
// 				}
// 				if len(res.Events) != 0 {
// 					t.Errorf("Expected 0 events for a nonexistent term, but got %d", len(res.Events))
// 				}
// 			},
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// CLEANUP: Defer the deletion of test data for this specific run.
// 			defer func() {
// 				if len(tt.idsToCleanup) > 0 {
// 					_, err := services.BulkDeleteEventsFromWeaviate(context.Background(), testClient, tt.idsToCleanup)
// 					if err != nil {
// 						t.Errorf("ERROR in test cleanup: %v", err)
// 					}
// 				}
// 			}()
//
// 			// ARRANGE: Seed the database if a seeder function is provided.
// 			if tt.dbSeeder != nil {
// 				tt.dbSeeder(t)
// 			}
//
// 			// ACT: Perform the HTTP request against the real handler.
// 			req := httptest.NewRequest("GET", tt.path, nil)
// 			rr := httptest.NewRecorder()
//
// 			// Replace `YourSearchEventsHandler` with your actual handler function.
// 			handler := SearchEventsHandler(rr, req)
// 			handler(rr, req)
//
// 			// ASSERT: Check the results.
// 			if status := rr.Code; status != tt.expectedStatus {
// 				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
// 			}
// 			if tt.expectedBodyCheck != nil {
// 				tt.expectedBodyCheck(t, rr.Body.String())
// 			}
// 		})
// 	}
// }

func TestBulkUpdateEvents(t *testing.T) {
	// Instantiate the real Weaviate service once for all tests to use.

	// SETUP: Load timezone location needed for creating valid Event objects.
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("SETUP FAILED: could not load timezone: %v", err)
	}

	// Define test cases.
	tests := []struct {
		name              string
		dbSeeder          func(t *testing.T) // Sets up the initial state in the DB.
		requestBody       string             // The PUT request payload with the updates.
		idsToCleanup      []string           // IDs to delete after the test.
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
		dbAssertionCheck  func(t *testing.T) // Verifies the final state in the DB.
	}{
		{
			name: "Successful bulk update modifies existing events",
			dbSeeder: func(t *testing.T) {
				// ARRANGE Part 1: Create the initial versions of events in the DB.
				initialEvents := []types.Event{
					{
						Id:             "update-test-1",
						EventOwners:    []string{"owner-123"},
						EventOwnerName: "Original Owner",
						Name:           "Original Event Name", // This is what we will check for changes.
						Description:    "Initial description.",
						StartTime:      time.Now().Add(10 * 24 * time.Hour).Unix(),
						Address:        "1 First St, Washington, DC",
						Lat:            38.8951,
						Long:           -77.0364,
						Timezone:       *tz,
					},
				}
				_, err := services.BulkUpsertEventsToWeaviate(context.Background(), testClient, initialEvents)
				if err != nil {
					t.Fatalf("DB seeder failed for update test: %v", err)
				}
			},
			// ARRANGE Part 2: This is the payload for the PUT request with the updated data.
			requestBody: `{ "events": [
				{"id": "update-test-1", "eventOwners":["owner-123"],"eventOwnerName":"Updated Owner","name":"Updated Event Name","description":"This description has been updated.","startTime":` + fmt.Sprintf("%d", time.Now().Add(11*24*time.Hour).Unix()) + `,"address":"1 First St, Washington, DC","lat":38.8951,"long":-77.0364,"timezone":"America/New_York"}
			]}`,
			idsToCleanup:   []string{"update-test-1"},
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, `"errors":false`) {
					t.Errorf("Expected response body to indicate no errors, but got: %s", body)
				}
			},
			dbAssertionCheck: func(t *testing.T) {
				// ASSERT Part 2: Verify that the event in the DB was actually updated.
				event, err := services.GetWeaviateEventByID(context.Background(), testClient, "update-test-1", "0")
				if err != nil {
					t.Fatalf("Failed to get event from Weaviate for verification: %v", err)
				}
				if event == nil {
					t.Fatal("Event 'update-test-1' was not found in Weaviate after update handler ran")
				}
				// Check if the name was successfully changed.
				if event.Name != "Updated Event Name" {
					t.Errorf("Expected event name to be 'Updated Event Name', but got '%s'", event.Name)
				}
				if event.EventOwnerName != "Updated Owner" {
					t.Errorf("Expected event owner name to be 'Updated Owner', but got '%s'", event.EventOwnerName)
				}
			},
		},
		{
			name:           "Bulk update with an event missing an ID fails validation",
			dbSeeder:       nil,
			requestBody:    `{ "events": [{"name": "Event missing an ID"}]}`,
			idsToCleanup:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				if !strings.Contains(body, "event has no id") {
					t.Errorf("Expected body to contain 'event has no id', but got '%s'", body)
				}
			},
			dbAssertionCheck: nil, // No DB changes are expected.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// CLEANUP: Defer the deletion of test data for this specific run.
			defer func() {
				if len(tt.idsToCleanup) > 0 {
					_, err := services.BulkDeleteEventsFromWeaviate(context.Background(), testClient, tt.idsToCleanup)
					if err != nil {
						t.Errorf("ERROR in test cleanup: %v", err)
					}
				}
			}()

			// ARRANGE: Seed the database if a seeder function is provided for the test case.
			if tt.dbSeeder != nil {
				tt.dbSeeder(t)
			}

			// ACT: Perform the HTTP request against the real handler.
			req := httptest.NewRequest("PUT", "/api/events", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			// Replace `YourBulkUpdateEventsHandler` with your actual handler function.
			handler := BulkUpdateEventsHandler(rr, req)
			handler(rr, req)

			// ASSERT: Check the HTTP response and the final database state.
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
			if tt.dbAssertionCheck != nil {
				tt.dbAssertionCheck(t)
			}
		})
	}
}

func TestUpdateOneEvent(t *testing.T) {
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("SETUP FAILED: could not load timezone: %v", err)
	}

	// Define test cases.
	tests := []struct {
		name              string
		eventID           string // The ID of the event to update
		dbSeeder          func(t *testing.T)
		requestBody       string
		idsToCleanup      []string
		expectedStatus    int
		expectedBodyCheck func(t *testing.T, body string)
		dbAssertionCheck  func(t *testing.T)
	}{
		{
			name:    "Successful update of a single event",
			eventID: "update-single-1",
			dbSeeder: func(t *testing.T) {
				// ARRANGE Part 1: Create the initial version of the event in the DB.
				initialEvent := []types.Event{
					{
						Id:              "update-single-1",
						EventOwners:     []string{"owner-abc"},
						EventOwnerName:  "The Original Organizer",
						EventSourceType: helpers.ES_SINGLE_EVENT,
						Name:            "Pre-Update Concert",
						Description:     "An event that is about to be updated.",
						StartTime:       time.Now().Add(24 * time.Hour).Unix(),
						Address:         "123 Old Street, New York, NY",
						Lat:             40.7128,
						Long:            -74.0060,
						Timezone:        *tz,
					},
				}
				_, err := services.BulkUpsertEventsToWeaviate(context.Background(), testClient, initialEvent)
				if err != nil {
					t.Fatalf("DB seeder failed for update test: %v", err)
				}
			},
			// ARRANGE Part 2: The updated event data in the request body.
			requestBody: `{
				"id": "update-single-1",
				"eventOwners": ["owner-abc"],
				"eventOwnerName": "The New Organizer",
				"eventSourceType": "` + helpers.ES_SINGLE_EVENT + `",
				"name": "Post-Update Rock Show",
				"description": "This event has been successfully updated.",
				"startTime": ` + fmt.Sprintf("%d", time.Now().Add(25*time.Hour).Unix()) + `,
				"address": "456 New Avenue, New York, NY",
				"lat": 40.7129,
				"long": -74.0061,
				"timezone": "America/New_York"
			}`,
			idsToCleanup:   []string{"update-single-1"},
			expectedStatus: http.StatusOK,
			expectedBodyCheck: func(t *testing.T, body string) {
				// Body check can be minimal, as the DB check is the source of truth.
				var event types.Event
				if err := json.Unmarshal([]byte(body), &event); err != nil {
					t.Fatalf("Failed to unmarshal response body: %v", err)
				}
				if event.Id != "update-single-1" {
					t.Errorf("Expected response ID to be 'update-single-1', got '%s'", event.Id)
				}
			},
			dbAssertionCheck: func(t *testing.T) {
				// ASSERT: Verify that the event in the DB was actually updated.
				event, err := services.GetWeaviateEventByID(context.Background(), testClient, "update-single-1", "0")
				if err != nil {
					t.Fatalf("Failed to get event from Weaviate for verification: %v", err)
				}
				if event == nil {
					t.Fatal("Event 'update-single-1' was not found in Weaviate after update")
				}
				// Check if the fields were changed.
				if event.Name != "Post-Update Rock Show" {
					t.Errorf("Expected event name to be 'Post-Update Rock Show', but got '%s'", event.Name)
				}
				if event.EventOwnerName != "The New Organizer" {
					t.Errorf("Expected owner name to be 'The New Organizer', but got '%s'", event.EventOwnerName)
				}
			},
		},
		{
			name:             "Update with invalid JSON fails",
			eventID:          "any-id",
			dbSeeder:         nil, // No DB state needed.
			requestBody:      `{"name": "Invalid JSON",}`,
			idsToCleanup:     nil,
			expectedStatus:   http.StatusUnprocessableEntity,
			dbAssertionCheck: nil,
		},
		{
			name:           "Update with missing required field fails validation",
			eventID:        "any-id",
			dbSeeder:       nil,
			requestBody:    `{"id": "any-id", "description": "This event is missing a name"}`,
			idsToCleanup:   nil,
			expectedStatus: http.StatusBadRequest,
			expectedBodyCheck: func(t *testing.T, body string) {
				// Check for the specific validation error.
				if !strings.Contains(body, "Field validation for 'Name' failed on the 'required' tag") {
					t.Errorf("Expected error about missing 'Name' field, but got: %s", body)
				}
			},
			dbAssertionCheck: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// CLEANUP: Defer the deletion of test data for this specific run.
			defer func() {
				if len(tt.idsToCleanup) > 0 {
					_, err := services.BulkDeleteEventsFromWeaviate(context.Background(), testClient, tt.idsToCleanup)
					if err != nil {
						t.Errorf("ERROR in test cleanup: %v", err)
					}
				}
			}()

			// ARRANGE: Seed the database if a seeder function is provided.
			if tt.dbSeeder != nil {
				tt.dbSeeder(t)
			}

			// ACT: Perform the HTTP request against the real handler.
			path := fmt.Sprintf("/events/%s", tt.eventID)
			req := httptest.NewRequest("PUT", path, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// This is crucial for testing handlers that use gorilla/mux for URL parameters.
			req = mux.SetURLVars(req, map[string]string{
				"eventId": tt.eventID,
			})

			rr := httptest.NewRecorder()

			// Replace `YourUpdateOneEventHandler` with your actual handler function.
			handler := UpdateOneEventHandler(rr, req)
			handler(rr, req)

			// ASSERT: Check the HTTP response and the final database state.
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
			if tt.expectedBodyCheck != nil {
				tt.expectedBodyCheck(t, rr.Body.String())
			}
			if tt.dbAssertionCheck != nil {
				tt.dbAssertionCheck(t)
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

	// Set the mock Zitadel server URL
	mockZitadelServer.Listener.Close()
	var err error
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
	var err error
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
