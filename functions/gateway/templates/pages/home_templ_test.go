package pages

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestHomePage(t *testing.T) {
	// Mock data
	loc, _ := time.LoadLocation("America/New_York")
	events := []types.Event{
		{
			Id:              "123",
			Name:            "Test Event 1",
			Description:     "Description for Test Event 1",
			Address:         "123 Test St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "456",
			Name:            "Test Event 2",
			Description:     "Description for Test Event 2",
			Address:         "456 Test St",
			Lat:             40.7580,
			Long:            -73.9855,
			StartTime:       1704153600,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}

	origLatStr := ""
	origLonStr := ""
	// Test cases
	tests := []struct {
		name              string
		pageUser          *types.UserSearchResult
		cfLocation        constants.CdnLocation
		cityStr           string
		latStr            string
		lonStr            string
		origQueryLocation string
		expectedItems     []string
	}{
		{
			name:              "Without page user",
			pageUser:          nil,
			cityStr:           "",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
			},
		},
		{
			name: "With page user",
			// pageUser: &types.UserSearchResult{ID: "1234567890", DisplayName: "Test User", Meta: map[string]string{"about": "Welcome to Brian's Pub"}},
			pageUser: &types.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian Feister",
			},
			cityStr:           "",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"data-page-user-id=\"1234567890\"",
			},
		},
		{
			name: "With page user and `about` section",
			pageUser: &types.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian's Pub",
				Metadata: map[string]string{
					constants.META_ABOUT_KEY: "Welcome to Brian's Pub",
				},
			},
			cityStr:           "",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"data-page-user-id=\"1234567890\"",
				"Welcome to Brian's Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
		{
			name: "With non-default city / lat / lon / location",
			pageUser: &types.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian's Pub",
				Metadata: map[string]string{
					constants.META_ABOUT_KEY: "Welcome to Brian's Pub",
				},
			},
			cityStr:           "", // this is only for city from metadata
			latStr:            "29.760427",
			lonStr:            "-95.369803",
			origQueryLocation: "Houston, TX",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"data-city-label-initial=\"Houston, TX\"",
				"Houston, TX",
				"data-page-user-id=\"1234567890\"",
				"data-city-latitude-initial=\"29.760427\"",
				"data-city-longitude-initial=\"-95.369803\"",
				"Welcome to Brian's Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
		{
			name: "With page user and location from metadata",
			pageUser: &types.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian's Pub",
				Metadata: map[string]string{
					constants.META_ABOUT_KEY: "Welcome to Brian's Pub",
				},
			},
			cityStr:           "Georgetown, Texas",
			latStr:            "30.0",
			lonStr:            "45.0",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"Georgetown, Texas",
				"data-page-user-id=\"1234567890\"",
				"Welcome to Brian's Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
		{
			name: "With page user and cloudflare location",
			pageUser: &types.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian's Pub",
				Metadata: map[string]string{
					constants.META_ABOUT_KEY: "Welcome to Brian's Pub",
				},
			},
			cfLocation: constants.CdnLocation{
				City: "Salt Lake City",
				Lat:  40.760779,
				Lon:  -111.891047,
				CCA2: "US",
			},
			cityStr:           "",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"Salt Lake City, US",
				"data-page-user-id=\"1234567890\"",
				"Welcome to Brian's Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the HomePage function
			component := HomePage(context.Background(), events, tt.pageUser, tt.cfLocation, tt.cityStr, tt.latStr, tt.lonStr, origLatStr, origLonStr, tt.origQueryLocation)

			// Render the component
			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering HomePage: %v", err)
			}

			// Check if the rendered content contains expected elements
			renderedContent := buf.String()
			for _, element := range tt.expectedItems {
				if !strings.Contains(renderedContent, element) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
					t.Errorf("rendered content \n'%s'", renderedContent)
				}
			}
		})
	}
}

func TestReShareButton(t *testing.T) {
	component := ReShareButton(types.Event{
		Id:          "123",
		Name:        "Test Event 1",
		Description: "Description for Test Event 1",
	})

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering ReShareButton: %v", err)
	}

	renderedContent := buf.String()
	if !strings.Contains(renderedContent, "Re share") {
		t.Errorf("Expected rendered content to contain 'Re share', but it didn't")
	}
	if !strings.Contains(renderedContent, "re-share-123") {
		t.Errorf("Expected rendered content to contain button ID 're-share-123', but it didn't")
	}
}

func TestHomeWithReShareButton(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	events := []types.Event{
		{
			Id:              "123",
			Name:            "Test Event 1",
			Description:     "Description for Test Event 1",
			Address:         "123 Test St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}
	pageUser := &types.UserSearchResult{
		UserID:      "1234567890",
		DisplayName: "Brian Feister",
	}
	cfLocation := constants.CdnLocation{
		City: "New York",
		CCA2: "US",
	}
	component := HomePage(context.Background(), events, pageUser, cfLocation, "New York", "40.7128", "-74.0060", "", "", "")

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering HomePage: %v", err)
	}

	renderedContent := buf.String()
	if !strings.Contains(renderedContent, "Test Event 1") {
		t.Errorf("Expected rendered content to contain 'Test Event 1', but it didn't")
	}
	if !strings.Contains(renderedContent, "data-page-user-id=\"1234567890\"") {
		t.Errorf("Expected rendered content to contain 'data-page-user-id=\"1234567890\"', but it didn't")
	}
	if !strings.Contains(renderedContent, "data-city-label-initial=\"New York\"") {
		t.Errorf("Expected rendered content to contain 'data-city-label-initial=\"New York\"', but it didn't")
	}
	if !strings.Contains(renderedContent, "data-city-latitude-initial=\"40.7128\"") {
		t.Errorf("Expected rendered content to contain 'data-city-latitude-initial=\"40.7128\"', but it didn't")
	}
	if !strings.Contains(renderedContent, "data-city-longitude-initial=\"-74.0060\"") {
		t.Errorf("Expected rendered content to contain 'data-city-longitude-initial=\"-74.0060\"', but it didn't")
	}
}

func TestEventsInnerWithGroupedEvents(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	// Create grouped events - same name, same location, different dates
	groupedEvents := []types.Event{
		{
			Id:              "event-1",
			Name:            "Weekly Meetup",
			Description:     "A weekly meetup event",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200, // 2024-01-01
			Timezone:        *loc,
			EventOwners:     []string{"owner-1"},
			EventOwnerName:  "Test Owner",
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-2",
			Name:            "Weekly Meetup",
			Description:     "A weekly meetup event",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704672000, // 2024-01-08
			Timezone:        *loc,
			EventOwners:     []string{"owner-1"},
			EventOwnerName:  "Test Owner",
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-3",
			Name:            "Weekly Meetup",
			Description:     "A weekly meetup event",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1705276800, // 2024-01-15
			Timezone:        *loc,
			EventOwners:     []string{"owner-1"},
			EventOwnerName:  "Test Owner",
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}

	// Create ungrouped event - different location
	ungroupedEvent := types.Event{
		Id:              "event-4",
		Name:            "Different Event",
		Description:     "An event at a different location",
		Address:         "456 Other St",
		Lat:             34.0522,
		Long:            -118.2437,
		StartTime:       1704067200,
		Timezone:        *loc,
		EventOwners:     []string{"owner-2"},
		EventOwnerName:  "Other Owner",
		EventSourceType: constants.ES_SINGLE_EVENT,
	}

	pageUser := &types.UserSearchResult{
		UserID: "user-1",
	}

	tests := []struct {
		name          string
		mode          string
		events        []types.Event
		expectedItems []string
		notExpected   []string
	}{
		{
			name:   "EV_MODE_UPCOMING with grouped events",
			mode:   constants.EV_MODE_UPCOMING,
			events: groupedEvents,
			expectedItems: []string{
				"Weekly Meetup",
				"123 Main St",
				"carousel-container",
				"/event/event-1",
				"/event/event-2",
				"/event/event-3",
			},
		},
		{
			name:   "EV_MODE_CAROUSEL with grouped events",
			mode:   constants.EV_MODE_CAROUSEL,
			events: groupedEvents,
			expectedItems: []string{
				"carousel-container",
				"carousel-item",
			},
		},
		{
			name:   "EV_MODE_LIST with grouped events",
			mode:   constants.EV_MODE_LIST,
			events: groupedEvents,
			expectedItems: []string{
				"Weekly Meetup",
				"carousel-container",
			},
		},
		{
			name:   "EV_MODE_ADMIN_LIST with grouped events",
			mode:   constants.EV_MODE_ADMIN_LIST,
			events: groupedEvents,
			expectedItems: []string{
				"Weekly Meetup",
				"3 occurrences",
				"Event Admin",
			},
		},
		{
			name:   "EV_MODE_UPCOMING with single event (no grouping)",
			mode:   constants.EV_MODE_UPCOMING,
			events: []types.Event{groupedEvents[0]},
			expectedItems: []string{
				"Weekly Meetup",
			},
			notExpected: []string{
				"carousel-container",
			},
		},
		{
			name:   "EV_MODE_ADMIN_LIST with single event (no grouping indicator)",
			mode:   constants.EV_MODE_ADMIN_LIST,
			events: []types.Event{groupedEvents[0]},
			expectedItems: []string{
				"Weekly Meetup",
			},
			notExpected: []string{
				"occurrences",
			},
		},
		{
			name:   "EV_MODE_LIST with mixed grouped and ungrouped events",
			mode:   constants.EV_MODE_LIST,
			events: append(groupedEvents, ungroupedEvent),
			expectedItems: []string{
				"Weekly Meetup",
				"Different Event",
			},
		},
	}

	// Test EventSourceId icon display in ADMIN_LIST mode
	eventWithSourceId := types.Event{
		Id:              "event-with-source",
		Name:            "Re-Shared Event",
		Description:     "An event that was re-shared",
		Address:         "789 Source St",
		Lat:             40.7128,
		Long:            -74.0060,
		StartTime:       1704067200,
		Timezone:        *loc,
		EventOwners:     []string{"owner-1"},
		EventOwnerName:  "Test Owner",
		EventSourceType: constants.ES_SINGLE_EVENT,
		EventSourceId:   "source-123", // This should trigger the icon display
	}

	eventWithoutSourceId := types.Event{
		Id:              "event-without-source",
		Name:            "Regular Event",
		Description:     "An event without source ID",
		Address:         "321 Regular St",
		Lat:             34.0522,
		Long:            -118.2437,
		StartTime:       1704067200,
		Timezone:        *loc,
		EventOwners:     []string{"owner-2"},
		EventOwnerName:  "Other Owner",
		EventSourceType: constants.ES_SINGLE_EVENT,
		EventSourceId:   "", // This should NOT trigger the icon display
	}

	eventSourceIdTests := []struct {
		name          string
		mode          string
		events        []types.Event
		expectedItems []string
		notExpected   []string
	}{
		{
			name:   "EV_MODE_ADMIN_LIST with EventSourceId (should show icon)",
			mode:   constants.EV_MODE_ADMIN_LIST,
			events: []types.Event{eventWithSourceId},
			expectedItems: []string{
				"Re-Shared Event",
				"viewBox=\"0 -960 960 960\"", // SVG viewBox attribute
				"M482-160q-134 0-228-93t-94-227v-7l-64 64-56-56 160-160 160 160-56 56-64-64v7q0 100 70.5 170T482-240q26 0 51-6t49-18l60 60q-38 22-78 33t-82 11Zm278-161L600-481l56-56 64 64v-7q0-100-70.5-170T478-720q-26 0-51 6t-49 18l-60-60q38-22 78-33t82-11q134 0 228 93t94 227v7l64-64 56 56-160 160Z", // SVG path
			},
		},
		{
			name:   "EV_MODE_ADMIN_LIST without EventSourceId (should not show icon)",
			mode:   constants.EV_MODE_ADMIN_LIST,
			events: []types.Event{eventWithoutSourceId},
			expectedItems: []string{
				"Regular Event",
			},
			notExpected: []string{
				"viewBox=\"0 -960 960 960\"", // SVG should not appear
			},
		},
		{
			name:   "EV_MODE_ADMIN_LIST with mixed events (one with source, one without)",
			mode:   constants.EV_MODE_ADMIN_LIST,
			events: []types.Event{eventWithSourceId, eventWithoutSourceId},
			expectedItems: []string{
				"Re-Shared Event",
				"Regular Event",
				"viewBox=\"0 -960 960 960\"", // Should appear for the first event only
			},
		},
	}

	for _, tt := range eventSourceIdTests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventsInner(tt.events, tt.mode, []constants.RoleClaim{}, "", pageUser, false, "")

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering EventsInner: %v", err)
			}

			renderedContent := buf.String()

			// Check for expected items
			for _, item := range tt.expectedItems {
				if !strings.Contains(renderedContent, item) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", item)
				}
			}

			// Check that not expected items are absent
			for _, item := range tt.notExpected {
				if strings.Contains(renderedContent, item) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", item)
				}
			}
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventsInner(tt.events, tt.mode, []constants.RoleClaim{}, "", pageUser, false, "")

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering EventsInner: %v", err)
			}

			renderedContent := buf.String()

			for _, expected := range tt.expectedItems {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
					t.Logf("Rendered content:\n%s", renderedContent)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(renderedContent, notExpected) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", notExpected)
					t.Logf("Rendered content:\n%s", renderedContent)
				}
			}
		})
	}
}

func TestGetGroupedEventsPreservesOrder(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	// Create events in a specific order: Group A (first), Group B (second), Group A again (should be grouped with first)
	events := []types.Event{
		{
			Id:              "event-a1",
			Name:            "Event A",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-b1",
			Name:            "Event B",
			Address:         "456 Other St",
			Lat:             34.0522,
			Long:            -118.2437,
			StartTime:       1704153600,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-a2",
			Name:            "Event A",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704240000,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-c1",
			Name:            "Event C",
			Address:         "789 Third St",
			Lat:             41.8781,
			Long:            -87.6298,
			StartTime:       1704326400,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}

	// We can't directly test getGroupedEvents since it's a templ function,
	// but we can test that EventsInner preserves order by checking the rendered output
	pageUser := &types.UserSearchResult{
		UserID: "user-1",
	}

	component := EventsInner(events, constants.EV_MODE_ADMIN_LIST, []constants.RoleClaim{}, "", pageUser, false, "")

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering EventsInner: %v", err)
	}

	renderedContent := buf.String()

	// Find the positions of each event name in the rendered output
	// Event A should appear first (even though it has 2 occurrences)
	// Event B should appear second
	// Event C should appear third
	posA := strings.Index(renderedContent, "Event A")
	posB := strings.Index(renderedContent, "Event B")
	posC := strings.Index(renderedContent, "Event C")

	if posA == -1 || posB == -1 || posC == -1 {
		t.Errorf("Not all events found in rendered content. A: %d, B: %d, C: %d", posA, posB, posC)
		return
	}

	// Verify order: A should come before B, B should come before C
	if posA > posB {
		t.Errorf("Event A should appear before Event B, but A at position %d, B at position %d", posA, posB)
	}
	if posB > posC {
		t.Errorf("Event B should appear before Event C, but B at position %d, C at position %d", posB, posC)
	}

	// Verify that Event A shows "2 occurrences" since it's grouped
	if !strings.Contains(renderedContent, "2 occurrences") {
		t.Errorf("Expected '2 occurrences' to appear for grouped Event A")
	}

	// Verify that Event B and C don't show occurrences (they're single events)
	// Count occurrences of "occurrences" - should be exactly 1 (for Event A)
	occurrencesCount := strings.Count(renderedContent, "occurrences")
	if occurrencesCount != 1 {
		t.Errorf("Expected exactly 1 'occurrences' text (for Event A), but found %d", occurrencesCount)
	}
}

func TestGetGroupedEventsPreservesOrderWithManyGroups(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	// Create 15 different event groups to test padding (values 0-14, including 10+)
	events := []types.Event{}
	for i := 0; i < 15; i++ {
		events = append(events, types.Event{
			Id:              fmt.Sprintf("event-%d", i),
			Name:            fmt.Sprintf("Event %d", i),
			Address:         fmt.Sprintf("%d Main St", i),
			Lat:             40.7128 + float64(i)*0.01, // Different lat for each group
			Long:            -74.0060 - float64(i)*0.01,
			StartTime:       1704067200 + int64(i)*86400,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		})
	}

	pageUser := &types.UserSearchResult{
		UserID: "user-1",
	}

	component := EventsInner(events, constants.EV_MODE_ADMIN_LIST, []constants.RoleClaim{}, "", pageUser, false, "")

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering EventsInner: %v", err)
	}

	renderedContent := buf.String()

	// Verify all events appear in order (0, 1, 2, ..., 9, 10, 11, 12, 13, 14)
	// Check that Event 0 comes before Event 10 (critical test for padding)
	pos0 := strings.Index(renderedContent, "Event 0")
	pos10 := strings.Index(renderedContent, "Event 10")
	pos14 := strings.Index(renderedContent, "Event 14")

	if pos0 == -1 || pos10 == -1 || pos14 == -1 {
		t.Errorf("Not all events found. Event 0: %d, Event 10: %d, Event 14: %d", pos0, pos10, pos14)
		return
	}

	// Critical: Event 0 should come before Event 10 (tests padding)
	if pos0 > pos10 {
		t.Errorf("Event 0 should appear before Event 10 (padding test), but Event 0 at %d, Event 10 at %d", pos0, pos10)
	}

	// Event 10 should come before Event 14
	if pos10 > pos14 {
		t.Errorf("Event 10 should appear before Event 14, but Event 10 at %d, Event 14 at %d", pos10, pos14)
	}

	// Verify sequential ordering for a few more events
	for i := 1; i < 10; i++ {
		posI := strings.Index(renderedContent, fmt.Sprintf("Event %d", i))
		posI1 := strings.Index(renderedContent, fmt.Sprintf("Event %d", i+1))
		if posI > posI1 {
			t.Errorf("Event %d should appear before Event %d, but Event %d at %d, Event %d at %d", i, i+1, i, posI, i+1, posI1)
		}
	}

	// Verify events 10-14 are in order
	for i := 10; i < 14; i++ {
		posI := strings.Index(renderedContent, fmt.Sprintf("Event %d", i))
		posI1 := strings.Index(renderedContent, fmt.Sprintf("Event %d", i+1))
		if posI > posI1 {
			t.Errorf("Event %d should appear before Event %d, but Event %d at %d, Event %d at %d", i, i+1, i, posI, i+1, posI1)
		}
	}
}

func TestGetGroupedEventsSortsByStartTime(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	// Create events with the same name and location but different start times (out of order)
	events := []types.Event{
		{
			Id:              "event-3",
			Name:            "Weekly Meetup",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704240000, // Latest date (2024-01-03)
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-1",
			Name:            "Weekly Meetup",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200, // Earliest date (2024-01-01)
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "event-2",
			Name:            "Weekly Meetup",
			Address:         "123 Main St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704153600, // Middle date (2024-01-02)
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}

	pageUser := &types.UserSearchResult{
		UserID: "user-1",
	}

	component := EventsInner(events, constants.EV_MODE_CAROUSEL, []constants.RoleClaim{}, "", pageUser, false, "")

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering EventsInner: %v", err)
	}

	renderedContent := buf.String()

	// Find positions of event IDs in the rendered output
	// Events should be ordered by StartTime: event-1, event-2, event-3
	pos1 := strings.Index(renderedContent, "/event/event-1")
	pos2 := strings.Index(renderedContent, "/event/event-2")
	pos3 := strings.Index(renderedContent, "/event/event-3")

	if pos1 == -1 || pos2 == -1 || pos3 == -1 {
		t.Errorf("Not all events found in rendered content. event-1: %d, event-2: %d, event-3: %d", pos1, pos2, pos3)
		return
	}

	// Verify chronological order: event-1 (earliest) should come before event-2, which should come before event-3
	if pos1 > pos2 {
		t.Errorf("event-1 (earliest) should appear before event-2, but event-1 at %d, event-2 at %d", pos1, pos2)
	}
	if pos2 > pos3 {
		t.Errorf("event-2 should appear before event-3 (latest), but event-2 at %d, event-3 at %d", pos2, pos3)
	}
}

func TestEventsInnerEdgeCases(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	pageUser := &types.UserSearchResult{
		UserID: "user-1",
	}

	tests := []struct {
		name          string
		mode          string
		events        []types.Event
		expectedItems []string
	}{
		{
			name:   "EV_MODE_UPCOMING with empty events",
			mode:   constants.EV_MODE_UPCOMING,
			events: []types.Event{},
			expectedItems: []string{
				"No events found",
			},
		},
		{
			name:   "EV_MODE_CAROUSEL with empty events",
			mode:   constants.EV_MODE_CAROUSEL,
			events: []types.Event{},
			expectedItems: []string{
				"This event series has no events",
			},
		},
		{
			name:   "EV_MODE_LIST with empty events",
			mode:   constants.EV_MODE_LIST,
			events: []types.Event{},
			expectedItems: []string{
				"No events found",
			},
		},
		{
			name: "EV_MODE_UPCOMING with event without location data",
			mode: constants.EV_MODE_UPCOMING,
			events: []types.Event{
				{
					Id:              "event-no-loc",
					Name:            "Event Without Location",
					Description:     "An event without location data",
					Address:         "Unknown",
					Lat:             0,
					Long:            0,
					StartTime:       1704067200,
					Timezone:        *loc,
					EventOwners:     []string{"owner-1"},
					EventOwnerName:  "Test Owner",
					EventSourceType: constants.ES_SINGLE_EVENT,
				},
			},
			expectedItems: []string{
				"No events found",
			},
		},
		{
			name: "EV_MODE_UPCOMING with event without name",
			mode: constants.EV_MODE_UPCOMING,
			events: []types.Event{
				{
					Id:              "event-no-name",
					Name:            "",
					Description:     "An event without a name",
					Address:         "123 Main St",
					Lat:             40.7128,
					Long:            -74.0060,
					StartTime:       1704067200,
					Timezone:        *loc,
					EventOwners:     []string{"owner-1"},
					EventOwnerName:  "Test Owner",
					EventSourceType: constants.ES_SINGLE_EVENT,
				},
			},
			expectedItems: []string{
				"No events found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventsInner(tt.events, tt.mode, []constants.RoleClaim{}, "", pageUser, false, "")

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering EventsInner: %v", err)
			}

			renderedContent := buf.String()

			for _, expected := range tt.expectedItems {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
					t.Logf("Rendered content:\n%s", renderedContent)
				}
			}
		})
	}
}
