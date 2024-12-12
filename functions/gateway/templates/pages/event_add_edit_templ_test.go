package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAddOrEditEventPage(t *testing.T) {
	tests := []struct {
		name     string
		event    types.Event
		isEditor bool
		sitePage helpers.SitePage
		expected []string
	}{
		{
			name:     "New event form",
			event:    types.Event{},
			isEditor: false,
			sitePage: helpers.SitePages["add-event"],
			expected: []string{
				"Event Name",
				"Description",
				"Location Details",
				"Start Date & Time",
				"End Date & Time",
				"Publish",
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
			},
			isEditor: true,
			sitePage: helpers.SitePages["edit-event"],
			expected: []string{
				"Event Name",
				"Description",
				"Location Details",
				"Start Date & Time",
				"End Date & Time",
				"Publish",
				"This is a test event",
				"123 Test St",
				"abc-uuid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := AddOrEditEventPage(helpers.SitePages["event-detail"], tt.event, tt.isEditor)

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
