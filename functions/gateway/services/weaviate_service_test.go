package services

import (
	"context"
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

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestWeaviateCRUDAndSearch(t *testing.T) {
	ctx := context.Background()
	loc, _ := time.LoadLocation("America/New_York")
	now := time.Now()

	testEvents := []types.Event{
		{
			Id:          "00000000-0000-0000-0000-000000000001",
			Name:        "Music Concert in Denver",
			Description: "A fun live music event with local bands",
			Address:     "Denver, CO",
			Lat:         39.7392, Long: -104.9903,
			StartTime: now.Add(1 * time.Hour).Unix(),
			Timezone:  *loc,
		},
		{
			Id:          "00000000-0000-0000-0000-000000000002",
			Name:        "Art Festival in Boulder",
			Description: "An outdoor show with paintings and sculptures",
			Address:     "Boulder, CO",
			Lat:         40.0150, Long: -105.2705,
			StartTime: now.Add(24 * time.Hour).Unix(),
			Timezone:  *loc,
		},
		{
			Id:          "00000000-0000-0000-0000-000000000003",
			Name:        "Tech Conference about Art",
			Description: "A conference about computers and digital art",
			Address:     "Convention Center, Denver, CO",
			Lat:         39.7424, Long: -104.9942,
			StartTime: now.Add(48 * time.Hour).Unix(),
			Timezone:  *loc,
		},
	}

	//  Test BulkUpsert and BulkUpdate
	t.Run("Bulk Upsert Events", func(t *testing.T) {
		resp, err := BulkUpsertEventsToWeaviate(ctx, testClient, testEvents)

		if err != nil {
			t.Fatalf("BulkUpsertEventToWeaviate should not return an error, but got: %v", err)
		}
		if resp == nil {
			t.Fatal("Response from BulkUpsert should not be nil")
		}
		if len(resp) != 3 {
			t.Errorf("Expected response length of 3, but got %d", len(resp))
		}
		for _, itemResult := range resp {
			if itemResult.Result == nil || itemResult.Result.Status == nil || *itemResult.Result.Status != "SUCCESS" {
				t.Errorf("Expected item status to be SUCCESS, but got: %v", itemResult.Result)
			}
		}

		// Now, test BulkUpdate by re-inserting the same events.
		_, err = BulkUpdateWeaviateEventsByID(ctx, testClient, testEvents)
		if err != nil {
			t.Fatalf("BulkUpdateWeaviateEventsByID should not return an error, but got: %v", err)
		}
	})

	//  Test BulkGet and Single Get by ID
	t.Run("Get Events by ID", func(t *testing.T) {
		idsToGet := []string{"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000003"}
		retrievedEvents, err := BulkGetWeaviateEventByID(ctx, testClient, idsToGet, "0")

		if err != nil {
			t.Fatalf("BulkGetWeaviateEventsByID should not return an error, but got: %v", err)
		}
		if len(retrievedEvents) != 2 {
			t.Fatalf("Expected to retrieve 2 events, but got %d", len(retrievedEvents))
		}

		// Verify a single event
		singleEvent, err := GetWeaviateEventByID(ctx, testClient, "00000000-0000-0000-0000-000000000002", "0")
		if err != nil {
			t.Fatalf("GetWeaviateEventByID should not return an error, but got: %v", err)
		}
		if singleEvent == nil {
			t.Fatal("Single event should not be nil")
		}
		if singleEvent.Name != "Art Festival in Boulder" {
			t.Errorf("Expected event name 'Art Festival in Boulder', but got '%s'", singleEvent.Name)
		}
	})

	// Test Search Functionality
	t.Run("Search for Events", func(t *testing.T) {
		// Give Weaviate a moment to finish indexing before searching.
		time.Sleep(1 * time.Second)

		searchResp, err := SearchWeaviateEvents(ctx, testClient, "live music show", []float64{39.7, -105.0}, 50000, now.Unix(), now.Add(7*24*time.Hour).Unix(), nil, "", "", "", nil, nil)

		if err != nil {
			t.Fatalf("SearchWeaviateEvents should not return an error, but got: %v", err)
		}
		if len(searchResp.Events) == 0 {
			t.Fatal("Search for 'live music show' should return at least one result, but got none")
		}

		// The most relevant result should be the concert.
		expectedID := "00000000-0000-0000-0000-000000000001"
		actualID := searchResp.Events[0].Id
		if actualID != expectedID {
			t.Errorf("Expected the most relevant event ID to be '%s', but got '%s'", expectedID, actualID)
		}
	})
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

func TestSearchWeaviateEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	loc, _ := time.LoadLocation("America/New_York")

	// brought over from the marqo test version
	searchTestEvents := []types.Event{
		{
			Id:             "1", // Was "_id"
			Name:           "Today's Event",
			StartTime:      now.Unix(),
			EventOwners:    []string{"789"}, // Is now []string
			EventOwnerName: "Today Host",
			Description:    "Event happening today",
			Lat:            51.5074, // Was "latitude"
			Long:           -0.1278, // Was "longitude"
			Timezone:       *loc,
		},
		{
			Id:             "2",
			Name:           "Next Week Event",
			StartTime:      now.AddDate(0, 0, 5).Unix(),
			EventOwners:    []string{"012"},
			EventOwnerName: "Week Host",
			Description:    "Event happening next week",
			Lat:            51.5074,
			Long:           -0.1278,
			Timezone:       *loc,
		},
		{
			Id:             "3",
			Name:           "Next Month Event",
			StartTime:      now.AddDate(0, 1, 0).Unix(),
			EventOwners:    []string{"345"},
			EventOwnerName: "Month Host",
			Description:    "Event happening next month",
			Lat:            51.5074,
			Long:           -0.1278,
			Timezone:       *loc,
		},
		{
			Id:             "4",
			Name:           "North Pole Event",
			StartTime:      now.Unix(),
			EventOwners:    []string{"678"},
			EventOwnerName: "Polar Host",
			Description:    "Event near the North Pole",
			Lat:            89.5,
			Long:           0.0,
			Timezone:       *loc,
		},
		{
			Id:             "5",
			Name:           "International Date Line Event (East)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"901"},
			EventOwnerName: "Date Line Host",
			Description:    "Event East of date line",
			Lat:            0.0,
			Long:           -179.9,
			Timezone:       *loc,
		},
		{
			Id:             "6",
			Name:           "International Date Line Event (West)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"389"},
			EventOwnerName: "Date Line Host",
			Description:    "Event West of date line",
			Lat:            0.0,
			Long:           179.9,
			Timezone:       *loc,
		},
		{
			Id:             "7",
			Name:           "Prime Meridian Event (East)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"251"},
			EventOwnerName: "Date Line Host",
			Description:    "Event West of date line",
			Lat:            10.0,
			Long:           -0.1,
			Timezone:       *loc,
		},
		{
			Id:             "8",
			Name:           "Prime Meridian Event (West)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"793"},
			EventOwnerName: "Date Line Host",
			Description:    "Event East of date line",
			Lat:            10.0,
			Long:           0.1,
			Timezone:       *loc,
		},
		{
			Id:             "9",
			Name:           "East Event (100 miles apart)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"239"},
			EventOwnerName: "Pair Event Host",
			Description:    "Paired Event East side",
			Lat:            35.685837,
			Long:           -105.945083,
			Timezone:       *loc,
		},
		{
			Id:             "10",
			Name:           "West Event (100 miles apart)",
			StartTime:      now.Unix(),
			EventOwners:    []string{"239"},
			EventOwnerName: "Pair Event Host",
			Description:    "Paired Event West side",
			Lat:            35.685837,
			Long:           -106.945083,
			Timezone:       *loc,
		},
	}

	_, err := BulkUpsertEventsToWeaviate(ctx, testClient, searchTestEvents)
	if err != nil {
		t.Fatalf("Arrange failed: could not insert test data for search tests: %v", err)
	}

	time.Sleep(1 * time.Second)

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
			name:        "Pair of Events, 56.25 miles apart",
			query:       "",
			startTime:   now.Unix(),
			endTime:     now.Add(24 * time.Hour).Unix(),
			location:    []float64{35.685837, -106.945083},
			distance:    56.25,
			expectedIds: []string{"9", "10"},
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
			expectedIds: []string{"1", "4", "5", "6", "7", "8", "9", "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchResp, err := SearchWeaviateEvents(
				ctx,
				testClient,
				tt.query,
				tt.location,
				tt.distance,
				tt.startTime,
				tt.endTime,
				[]string{}, "", "", "0", nil, nil,
			)

			if err != nil {
				t.Fatalf("SearchWeaviateEvents returned an unexpected error: %v", err)
			}

			if len(searchResp.Events) != len(tt.expectedIds) {
				t.Errorf("Expected %d events, but got %d. Results: %v", len(tt.expectedIds), len(searchResp.Events), eventIdsFromSlice(searchResp.Events))
			}

			returnedIds := make(map[string]bool)
			for _, event := range searchResp.Events {
				returnedIds[event.Id] = true
			}

			for _, expectedId := range tt.expectedIds {
				if _, found := returnedIds[expectedId]; !found {
					t.Errorf("Expected event with ID %s was not found in results", expectedId)
				}
			}
		})
	}
}

func eventIdsFromSlice(events []types.Event) []string {
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.Id
	}
	return ids
}

func TestGetWeaviateEventByID(t *testing.T) {
	ctx := context.Background()

	loc, _ := time.LoadLocation("America/New_York")
	startTime, _ := helpers.UtcToUnix64("2099-08-15T10:00:00Z", loc)

	testEvent := types.Event{
		Id:              "test-id-123", // Use a known, unique ID
		Name:            "Single Event Test",
		Description:     "This is a test event for GetByID.",
		EventOwners:     []string{"owner-789"},
		EventOwnerName:  "GetByID Test Host",
		EventSourceType: "TEST",
		StartTime:       startTime,
		Address:         "123 Weaviate Way",
		Lat:             40.7128,
		Long:            -74.0060,
		Timezone:        *loc,
	}

	_, err := BulkUpsertEventsToWeaviate(ctx, testClient, []types.Event{testEvent})
	if err != nil {
		t.Fatalf("ARRANGE failed: could not insert test event: %v", err)
	}

	// Give Weaviate a moment to index the new object.
	time.Sleep(200 * time.Millisecond)

	retrievedEvent, err := GetWeaviateEventByID(ctx, testClient, "test-id-123", "0")

	// Check that the retrieved event has the correct data.
	if err != nil {
		t.Fatalf("GetWeaviateEventByID returned an unexpected error: %v", err)
	}
	if retrievedEvent == nil {
		t.Fatal("Expected to retrieve an event, but got nil")
	}

	// Check each property of the returned event.
	if retrievedEvent.Id != testEvent.Id {
		t.Errorf("expected event ID '%s', got '%s'", testEvent.Id, retrievedEvent.Id)
	}
	if retrievedEvent.Name != testEvent.Name {
		t.Errorf("expected event name '%s', got '%s'", testEvent.Name, retrievedEvent.Name)
	}
	if retrievedEvent.Description != testEvent.Description {
		t.Errorf("expected event description '%s', got '%s'", testEvent.Description, retrievedEvent.Description)
	}
	if len(retrievedEvent.EventOwners) != 1 || retrievedEvent.EventOwners[0] != "owner-789" {
		t.Errorf("expected event owner '%s', got %v", "owner-789", retrievedEvent.EventOwners)
	}
	if retrievedEvent.StartTime != testEvent.StartTime {
		t.Errorf("expected event start time '%v', got '%v'", testEvent.StartTime, retrievedEvent.StartTime)
	}
}

func TestBulkGetWeaviateEventsByID(t *testing.T) {
	ctx := context.Background()

	loc, _ := time.LoadLocation("America/New_York")
	startTime1, _ := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	startTime2, _ := helpers.UtcToUnix64("2099-06-01T12:00:00Z", loc)
	startTime3, _ := helpers.UtcToUnix64("2099-07-01T12:00:00Z", loc)

	testEventsToInsert := []types.Event{
		{Id: "bulk-get-id-123", Name: "Test Event 1", Description: "This is test event 1", StartTime: startTime1, EventOwners: []string{"owner-789"}, EventOwnerName: "Test Owner 1", Timezone: *loc},
		{Id: "bulk-get-id-456", Name: "Test Event 2", Description: "This is test event 2", StartTime: startTime2, EventOwners: []string{"owner-012"}, EventOwnerName: "Test Owner 2", Timezone: *loc},
		{Id: "bulk-get-id-789", Name: "Test Event 3", Description: "This event won't be fetched", StartTime: startTime3, EventOwners: []string{"owner-345"}, EventOwnerName: "Test Owner 3", Timezone: *loc},
	}

	_, err := BulkUpsertEventsToWeaviate(ctx, testClient, testEventsToInsert)
	if err != nil {
		t.Fatalf("Insert for test setup failed: could not insert test events: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	idsToFetch := []string{"bulk-get-id-123", "bulk-get-id-456"}
	retrievedEvents, err := BulkGetWeaviateEventByID(ctx, testClient, idsToFetch, "0")

	if err != nil {
		t.Fatalf("BulkGetWeaviateEventsByID returned an unexpected error: %v", err)
	}
	if len(retrievedEvents) != 2 {
		t.Fatalf("Expected to retrieve 2 events but got %d", len(retrievedEvents))
	}

	resultsMap := make(map[string]*types.Event)
	for _, event := range retrievedEvents {
		resultsMap[event.Id] = event
	}

	event1, ok := resultsMap["bulk-get-id-123"]
	if !ok {
		t.Error("Expected to find event with ID 'bulk-get-id-123', but it was not returned.")
	} else {
		if event1.Name != "Test Event 1" {
			t.Errorf("For event 1, expected name 'Test Event 1', got '%s'", event1.Name)
		}
		if len(event1.EventOwners) != 1 || event1.EventOwners[0] != "owner-789" {
			t.Errorf("For event 1, expected owner 'owner-789', got '%v'", event1.EventOwners)
		}
	}

	event2, ok := resultsMap["bulk-get-id-456"]
	if !ok {
		t.Error("Expected to find event with ID 'bulk-get-id-456', but it was not returned.")
	} else {
		if event2.Name != "Test Event 2" {
			t.Errorf("For event 2, expected name 'Test Event 2', got '%s'", event2.Name)
		}
		if len(event2.EventOwners) != 1 || event2.EventOwners[0] != "owner-012" {
			t.Errorf("For event 2, expected owner 'owner-012', got '%v'", event2.EventOwners)
		}
	}
}
