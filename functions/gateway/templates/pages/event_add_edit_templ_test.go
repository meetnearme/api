package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAddOrEditEventPage(t *testing.T) {
	lALoc, _ := time.LoadLocation("America/Los_Angeles")
	tests := []struct {
		name     string
		event    types.Event
		isEditor bool
		sitePage helpers.SitePage
		cfLat    float64
		cfLon    float64
		expected []string
	}{
		{
			name:     "New event form",
			event:    types.Event{},
			isEditor: false,
			sitePage: helpers.SitePages["add-event"],
			cfLat:    services.InitialEmptyLatLong,
			cfLon:    services.InitialEmptyLatLong,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Publish",
				"Venue Address",
			},
		},
		{
			name:     "New event form, with admin user geolocation",
			event:    types.Event{},
			isEditor: false,
			sitePage: helpers.SitePages["add-event"],
			cfLat:    38.893725,
			cfLon:    -77.096975,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Publish",
				"Venue Address",
				"data-cf-lat=\"38.893725\"",
				"data-cf-lon=\"-77.096975\"",
			},
		},
		{
			name: "Edit existing event",
			event: types.Event{
				Id:             "123",
				Name:           "Test Event",
				Description:    "This is a test event",
				Address:        "123 Test St",
				EventOwners:    []string{"abc-uuid"},
				EventOwnerName: "Brians Pub",
				Timezone:       *lALoc,
			},
			isEditor: true,
			sitePage: helpers.SitePages["edit-event"],
			cfLat:    39.764252,
			cfLon:    -104.937511,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Venue Address",
				"Publish",
				"This is a test event",
				"123 Test St",
				"abc-uuid",
				"data-cf-lat=\"39.764252\"",
				"data-cf-lon=\"-104.937511\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := AddOrEditEventPage(helpers.SitePages["event-detail"], tt.event, tt.isEditor, tt.cfLat, tt.cfLon)

			// Render the component to a string
			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering component: %v", err)
			}
			result := buf.String()

			// Check if all expected strings are in the result
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected string not found: %s", exp)
					t.Errorf("Result: %s", result)
				}
			}
		})
	}
}
