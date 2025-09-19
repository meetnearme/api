package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAddOrEditEventPage(t *testing.T) {
	lALoc, _ := time.LoadLocation("America/Los_Angeles")
	userInfo := helpers.UserInfo{
		Sub:  "abc-uuid",
		Name: "Test User",
	}
	tests := []struct {
		name               string
		userInfo           helpers.UserInfo
		event              types.Event
		isEditor           bool
		isCompetitionAdmin bool
		sitePage           helpers.SitePage
		cfLat              float64
		cfLon              float64
		expected           []string
		notExpected        []string
	}{
		{
			name:               "New event form",
			userInfo:           userInfo,
			event:              types.Event{},
			isEditor:           false,
			isCompetitionAdmin: false,
			sitePage:           helpers.SitePages["add-event"],
			cfLat:              helpers.INITIAL_EMPTY_LAT_LONG,
			cfLon:              helpers.INITIAL_EMPTY_LAT_LONG,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Publish",
				"Address",
				"Test User", // via chip for default in `owners` array
			},
		},
		{
			name:               "New event form, with admin user geolocation",
			userInfo:           userInfo,
			event:              types.Event{},
			isEditor:           false,
			isCompetitionAdmin: false,
			sitePage:           helpers.SitePages["add-event"],
			cfLat:              38.893725,
			cfLon:              -77.096975,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Publish",
				"Address",
				"data-cf-lat=\"38.893725\"",
				"data-cf-lon=\"-77.096975\"",
			},
		},
		{
			name:     "Edit existing event",
			userInfo: userInfo,
			event: types.Event{
				Id:                  "123",
				Name:                "Test Event",
				Description:         "This is a test event",
				Address:             "123 Test St",
				CompetitionConfigId: "def-uuid",
				EventOwners:         []string{"xyz-uuid"},
				EventOwnerName:      "Brians Pub",
				Timezone:            *lALoc,
			},
			isEditor:           true,
			isCompetitionAdmin: false,
			sitePage:           helpers.SitePages["edit-event"],
			cfLat:              39.764252,
			cfLon:              -104.937511,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Address",
				"Publish",
				"This is a test event",
				"123 Test St",
				"def-uuid",
				`data-event-owners="[&#34;xyz-uuid&#34;]`, // NOTE: this intentionally omits the current user
				"data-cf-lat=\"39.764252\"",
				"data-cf-lon=\"-104.937511\"",
				"Only competition admins can modify competition config settings",
			},
			notExpected: []string{
				"Add Competition</button",
			},
		},
		{
			name:     "Edit existing event, as competition admin",
			userInfo: userInfo,
			event: types.Event{
				Id:                  "123",
				Name:                "Test Event",
				Description:         "This is a test event",
				Address:             "123 Test St",
				CompetitionConfigId: "xyz-uuid",
				EventOwners:         []string{"abc-uuid"},
				EventOwnerName:      "Brians Pub",
				Timezone:            *lALoc,
			},
			isEditor:           true,
			isCompetitionAdmin: true,
			sitePage:           helpers.SitePages["edit-event"],
			cfLat:              39.764252,
			cfLon:              -104.937511,
			expected: []string{
				"Event Name",
				"Description",
				"Date &amp; Media",
				"Start Date &amp; Time",
				"End Date &amp; Time",
				"Address",
				"Publish",
				"This is a test event",
				"123 Test St",
				"abc-uuid",
				"data-cf-lat=\"39.764252\"",
				"data-cf-lon=\"-104.937511\"",
				"Add Competition</button",
			},
		},
		{
			name:     "Edit unpublished event",
			userInfo: userInfo,
			event: types.Event{
				Id:                  "123",
				Name:                "Test Event",
				Description:         "This is a test event",
				Address:             "123 Test St",
				CompetitionConfigId: "xyz-uuid",
				EventOwners:         []string{"abc-uuid"},
				EventOwnerName:      "Brians Pub",
				Timezone:            *lALoc,
				EventSourceType:     helpers.ES_SINGLE_EVENT_UNPUB,
			},
			isEditor:           true,
			isCompetitionAdmin: true,
			sitePage:           helpers.SitePages["edit-event"],
			cfLat:              39.764252,
			cfLon:              -104.937511,
			expected: []string{
				`<option value="_UNPUB" selected`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := AddOrEditEventPage(helpers.SitePages["event-detail"], tt.userInfo, tt.event, tt.isEditor, tt.cfLat, tt.cfLon, tt.isCompetitionAdmin)

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
			for _, notExp := range tt.notExpected {
				if strings.Contains(result, notExp) {
					t.Errorf("Unexpected string found: %s", notExp)
					t.Errorf("Result: %s", result)
				}
			}
		})
	}
}
