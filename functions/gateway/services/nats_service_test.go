package services

import (
	"context"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestPublishMsg(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey:         "example-event-1",
		LocationLatitude:         1.352222231,
		LocationLongitude:        103.8198,
		LocationAddress:          "123 Orchard Road, Singapore",
		ScheduledHour:            15,
		TargetNameCSSPath:        ".event-title",
		TargetLocationCSSPath:    ".event-location",
		TargetStartTimeCSSPath:   ".start-time",
		TargetEndTimeCSSPath:     ".end-time",
		TargetDescriptionCSSPath: ".description",
		TargetHrefCSSPath:        "a.more-info",
		Status:                   "HEALTHY",
		LastScrapeSuccess:        1727385600,
		LastScrapeFailure:        0,
		LastScrapeFailureCount:   0,
		OwnerID:                  "user_abc123",
		KnownScrapeSource:        "MEETUP",
	}

	err := mockQueue.PublishMsg(ctx, payload)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockQueue.PublishedMsgs) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(mockQueue.PublishedMsgs))
	}
}

func TestPeekTopOfQueue(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "example-event-1",
	}

	_ = mockQueue.PublishMsg(ctx, payload)

	top, err := mockQueue.PeekTopOfQueue(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if top == nil {
		t.Error("Expected non-nil top message, got nil")
	}
}

func TestPeekFromEmptyQueue(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	top, err := mockQueue.PeekTopOfQueue(ctx)
	if err != nil {
		t.Fatalf("expected no error when queue empty, got %v", err)
	}
	if top != nil {
		t.Errorf("expected nil message when queue is empty")
	}
}

func TestConsumeMsg(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "example-event-1",
	}

	_ = mockQueue.PublishMsg(ctx, payload)

	err := mockQueue.ConsumeMsg(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	top, _ := mockQueue.PeekTopOfQueue(ctx)
	if top != nil {
		t.Error("Expected queue to be empty after ConsumeMsg, but it's not")
	}
}

func TestPublishMsgMarshalError(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	err := mockQueue.PublishMsg(ctx, func() {})
	if err == nil {
		t.Fatalf("expected marshal error for unsupported payload")
	}
	if len(mockQueue.PublishedMsgs) != 0 {
		t.Errorf("expected no messages stored on marshal failure")
	}
}

func TestConsumeMsgEmptyQueue(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	if err := mockQueue.ConsumeMsg(ctx); err != nil {
		t.Fatalf("expected no error consuming empty queue, got %v", err)
	}
}

// ========================================
// Event Comparison Tests
// ========================================

func TestEventComparison_ExactMatch(t *testing.T) {
	// Setup: Create timezone
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	// Existing event in DB
	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Friday Night Magic",
		Address:   "123 Main St, Austin, TX",
		StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
		Timezone:  *chicagoTz,
	}

	// New event from scrape (same event)
	newEvent := internal_types.EventInfo{
		EventTitle:     "Friday Night Magic",
		EventLocation:  "123 Main St, Austin, TX",
		EventStartTime: "2025-11-14T17:00:00", // Same time in local timezone
	}

	// Test comparison logic
	nameMatch := existingEvent.Name == newEvent.EventTitle
	locationMatch := existingEvent.Address == newEvent.EventLocation

	newEventTime, err := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}

	timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
	timeMatch := timeDiff == 0

	// Assertions
	if !nameMatch {
		t.Errorf("Expected name match, got nameMatch=%v", nameMatch)
	}
	if !locationMatch {
		t.Errorf("Expected location match, got locationMatch=%v", locationMatch)
	}
	if !timeMatch {
		t.Errorf("Expected time match, got timeMatch=%v, timeDiff=%d seconds", timeMatch, timeDiff)
	}

	isMatch := nameMatch && locationMatch && timeMatch
	if !isMatch {
		t.Errorf("Expected exact match for identical events")
	}
}

func TestEventComparison_DifferentTime(t *testing.T) {
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Friday Night Magic",
		Address:   "123 Main St, Austin, TX",
		StartTime: time.Date(2025, 11, 21, 17, 0, 0, 0, chicagoTz).Unix(), // Nov 21
		Timezone:  *chicagoTz,
	}

	newEvent := internal_types.EventInfo{
		EventTitle:     "Friday Night Magic",
		EventLocation:  "123 Main St, Austin, TX",
		EventStartTime: "2025-11-14T17:00:00", // Nov 14 - different date
	}

	nameMatch := existingEvent.Name == newEvent.EventTitle
	locationMatch := existingEvent.Address == newEvent.EventLocation

	newEventTime, _ := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
	timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
	timeMatch := timeDiff == 0

	if !nameMatch || !locationMatch {
		t.Errorf("Expected name and location to match")
	}
	if timeMatch {
		t.Errorf("Expected time NOT to match (different dates), got timeMatch=%v, timeDiff=%d", timeMatch, timeDiff)
	}

	isMatch := nameMatch && locationMatch && timeMatch
	if isMatch {
		t.Errorf("Expected NO match for different times (recurring event on different date)")
	}
}

func TestEventComparison_DifferentName(t *testing.T) {
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Friday Night Magic",
		Address:   "123 Main St, Austin, TX",
		StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
		Timezone:  *chicagoTz,
	}

	newEvent := internal_types.EventInfo{
		EventTitle:     "Saturday Morning Magic", // Different name
		EventLocation:  "123 Main St, Austin, TX",
		EventStartTime: "2025-11-14T17:00:00",
	}

	nameMatch := existingEvent.Name == newEvent.EventTitle
	locationMatch := existingEvent.Address == newEvent.EventLocation

	if nameMatch {
		t.Errorf("Expected name NOT to match, got nameMatch=%v", nameMatch)
	}

	isMatch := nameMatch && locationMatch
	if isMatch {
		t.Errorf("Expected NO match for different event names")
	}
}

func TestEventComparison_DifferentLocation(t *testing.T) {
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Friday Night Magic",
		Address:   "123 Main St, Austin, TX",
		StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
		Timezone:  *chicagoTz,
	}

	newEvent := internal_types.EventInfo{
		EventTitle:     "Friday Night Magic",
		EventLocation:  "456 Oak Ave, Austin, TX", // Different location
		EventStartTime: "2025-11-14T17:00:00",
	}

	nameMatch := existingEvent.Name == newEvent.EventTitle
	locationMatch := existingEvent.Address == newEvent.EventLocation

	if !nameMatch {
		t.Errorf("Expected name to match")
	}
	if locationMatch {
		t.Errorf("Expected location NOT to match, got locationMatch=%v", locationMatch)
	}

	isMatch := nameMatch && locationMatch
	if isMatch {
		t.Errorf("Expected NO match for different locations")
	}
}

func TestEventComparison_TimezoneCorrectness(t *testing.T) {
	// Test that parsing in the correct timezone prevents UTC/local mismatch
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	// Event at 5 PM Chicago time on Nov 14
	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Test Event",
		Address:   "123 Main St",
		StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
		Timezone:  *chicagoTz,
	}

	newEvent := internal_types.EventInfo{
		EventTitle:     "Test Event",
		EventLocation:  "123 Main St",
		EventStartTime: "2025-11-14T17:00:00", // 5 PM local time
	}

	// Parse in the CORRECT timezone (existing event's timezone)
	newEventTimeCorrect, _ := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
	correctTimeDiff := abs(existingEvent.StartTime - newEventTimeCorrect.Unix())

	// Parse INCORRECTLY as UTC (this would be the bug)
	newEventTimeWrong, _ := time.Parse("2006-01-02T15:04:05", newEvent.EventStartTime)
	wrongTimeDiff := abs(existingEvent.StartTime - newEventTimeWrong.Unix())

	// Assertions
	if correctTimeDiff != 0 {
		t.Errorf("Expected 0 time diff when parsing in correct timezone, got %d seconds", correctTimeDiff)
	}

	// Chicago is UTC-6 in November (CST), so parsing as UTC would be off by 6 hours = 21,600 seconds
	expectedWrongDiff := int64(6 * 3600) // 6 hours
	if wrongTimeDiff != expectedWrongDiff {
		t.Errorf("Expected wrong parsing to be off by %d seconds, got %d seconds", expectedWrongDiff, wrongTimeDiff)
	}

	t.Logf("Correct timezone parsing: %d second difference (should be 0)", correctTimeDiff)
	t.Logf("Wrong UTC parsing: %d second difference (should be ~21600)", wrongTimeDiff)
}

func TestEventComparison_RecurringEvents(t *testing.T) {
	// Test that recurring events (same name/location, different dates) are treated as separate events
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	// Existing events in DB: Friday Night Magic on Nov 14, 21, and 28
	existingEvents := []constants.Event{
		{
			Id:        "event-1",
			Name:      "Friday Night Magic",
			Address:   "123 Main St, Austin, TX",
			StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
		{
			Id:        "event-2",
			Name:      "Friday Night Magic",
			Address:   "123 Main St, Austin, TX",
			StartTime: time.Date(2025, 11, 21, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
		{
			Id:        "event-3",
			Name:      "Friday Night Magic",
			Address:   "123 Main St, Austin, TX",
			StartTime: time.Date(2025, 11, 28, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
	}

	// New event from scrape: Friday Night Magic on Nov 14
	newEvent := internal_types.EventInfo{
		EventTitle:     "Friday Night Magic",
		EventLocation:  "123 Main St, Austin, TX",
		EventStartTime: "2025-11-14T17:00:00",
	}

	// Test comparison logic
	matchCount := 0
	for _, existingEvent := range existingEvents {
		nameMatch := existingEvent.Name == newEvent.EventTitle
		locationMatch := existingEvent.Address == newEvent.EventLocation

		newEventTime, _ := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
		timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
		timeMatch := timeDiff == 0

		if nameMatch && locationMatch && timeMatch {
			matchCount++
		}
	}

	// Should only match ONE existing event (Nov 14), not all three
	if matchCount != 1 {
		t.Errorf("Expected new event to match exactly 1 existing event, got %d matches", matchCount)
	}
}

func TestEventComparison_SkipOptimization(t *testing.T) {
	// Test that skip check prevents redundant comparisons
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvents := []constants.Event{
		{
			Id:        "event-1",
			Name:      "Test Event",
			Address:   "123 Main St",
			StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
		{
			Id:        "event-2",
			Name:      "Test Event",
			Address:   "123 Main St",
			StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
	}

	newEvents := []internal_types.EventInfo{
		{
			EventTitle:     "Test Event",
			EventLocation:  "123 Main St",
			EventStartTime: "2025-11-14T17:00:00",
		},
	}

	// Simulate comparison with skip check
	newEventIndicesToSkip := make(map[int]bool)
	preservedEventIds := make(map[string]bool)
	comparisonCount := 0

	for _, existingEvent := range existingEvents {
		for j, newEvent := range newEvents {
			// Skip check optimization
			if newEventIndicesToSkip[j] {
				continue // Should skip on second iteration
			}

			comparisonCount++

			nameMatch := existingEvent.Name == newEvent.EventTitle
			locationMatch := existingEvent.Address == newEvent.EventLocation

			newEventTime, _ := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
			timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
			timeMatch := timeDiff == 0

			if nameMatch && locationMatch && timeMatch {
				preservedEventIds[existingEvent.Id] = true
				newEventIndicesToSkip[j] = true
				break
			}
		}
	}

	// Should only compare once (first existing event), then skip on second
	if comparisonCount != 1 {
		t.Errorf("Expected 1 comparison (skip optimization should prevent second), got %d", comparisonCount)
	}

	// Should only preserve first matching event
	if len(preservedEventIds) != 1 {
		t.Errorf("Expected 1 preserved event, got %d", len(preservedEventIds))
	}

	// Should mark new event as skipped
	if !newEventIndicesToSkip[0] {
		t.Errorf("Expected new event to be marked as skipped")
	}
}

func TestEventComparison_EmptyStartTime(t *testing.T) {
	// Test handling of empty start time
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvent := constants.Event{
		Id:        "event-123",
		Name:      "Test Event",
		Address:   "123 Main St",
		StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
		Timezone:  *chicagoTz,
	}

	newEvent := internal_types.EventInfo{
		EventTitle:     "Test Event",
		EventLocation:  "123 Main St",
		EventStartTime: "", // Empty time
	}

	nameMatch := existingEvent.Name == newEvent.EventTitle
	locationMatch := existingEvent.Address == newEvent.EventLocation

	// Time match should be false when EventStartTime is empty
	timeMatch := false
	if newEvent.EventStartTime != "" {
		newEventTime, err := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
		if err == nil {
			timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
			timeMatch = timeDiff == 0
		}
	}

	if timeMatch {
		t.Errorf("Expected time NOT to match when start time is empty")
	}

	isMatch := nameMatch && locationMatch && timeMatch
	if isMatch {
		t.Errorf("Expected NO match when start time is empty")
	}
}

func TestEventComparison_MultipleNewEvents(t *testing.T) {
	// Test matching with multiple new events
	chicagoTz, _ := time.LoadLocation("America/Chicago")

	existingEvents := []constants.Event{
		{
			Id:        "event-1",
			Name:      "Event A",
			Address:   "Location A",
			StartTime: time.Date(2025, 11, 14, 17, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
		{
			Id:        "event-2",
			Name:      "Event B",
			Address:   "Location B",
			StartTime: time.Date(2025, 11, 15, 18, 0, 0, 0, chicagoTz).Unix(),
			Timezone:  *chicagoTz,
		},
	}

	newEvents := []internal_types.EventInfo{
		{
			EventTitle:     "Event A",
			EventLocation:  "Location A",
			EventStartTime: "2025-11-14T17:00:00",
		},
		{
			EventTitle:     "Event B",
			EventLocation:  "Location B",
			EventStartTime: "2025-11-15T18:00:00",
		},
		{
			EventTitle:     "Event C",
			EventLocation:  "Location C",
			EventStartTime: "2025-11-16T19:00:00",
		},
	}

	// Simulate comparison logic
	preservedEventIds := make(map[string]bool)
	newEventIndicesToSkip := make(map[int]bool)

	for _, existingEvent := range existingEvents {
		for j, newEvent := range newEvents {
			if newEventIndicesToSkip[j] {
				continue
			}

			nameMatch := existingEvent.Name == newEvent.EventTitle
			locationMatch := existingEvent.Address == newEvent.EventLocation

			newEventTime, _ := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
			timeDiff := abs(existingEvent.StartTime - newEventTime.Unix())
			timeMatch := timeDiff == 0

			if nameMatch && locationMatch && timeMatch {
				preservedEventIds[existingEvent.Id] = true
				newEventIndicesToSkip[j] = true
				break
			}
		}
	}

	// Should preserve 2 existing events (Event A and B match)
	if len(preservedEventIds) != 2 {
		t.Errorf("Expected 2 preserved events, got %d", len(preservedEventIds))
	}

	// Should skip 2 new events (Event A and B)
	if len(newEventIndicesToSkip) != 2 {
		t.Errorf("Expected 2 skipped new events, got %d", len(newEventIndicesToSkip))
	}

	// Event C should NOT be skipped (new event)
	if newEventIndicesToSkip[2] {
		t.Errorf("Expected Event C to NOT be skipped (it's a new event)")
	}

	// Build eventsToInsert (only Event C)
	eventsToInsert := []internal_types.EventInfo{}
	for i, event := range newEvents {
		if !newEventIndicesToSkip[i] {
			eventsToInsert = append(eventsToInsert, event)
		}
	}

	if len(eventsToInsert) != 1 {
		t.Errorf("Expected 1 event to insert (Event C), got %d", len(eventsToInsert))
	}

	if len(eventsToInsert) > 0 && eventsToInsert[0].EventTitle != "Event C" {
		t.Errorf("Expected Event C to be inserted, got %s", eventsToInsert[0].EventTitle)
	}
}
