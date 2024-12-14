package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestEventAttendeesPage(t *testing.T) {
	tests := []struct {
		name     string
		pageObj  helpers.SitePage
		event    types.Event
		isEditor bool
		expected []string
	}{
		{
			name: "Empty event shows 404",
			pageObj: helpers.SitePage{
				Name: "Event Attendees",
			},
			event:    types.Event{},
			isEditor: false,
			expected: []string{
				"404 - Can't Find That Event",
			},
		},
		{
			name: "Valid event with purchasable items",
			pageObj: helpers.SitePage{
				Name: "Event Attendees",
			},
			event: types.Event{
				Id:                    "test-123",
				Name:                  "Test Event",
				HasPurchasable:        true,
				HasRegistrationFields: false,
			},
			isEditor: true,
			expected: []string{
				"Event Attendees",
				"Purchases",
				"data-event-id=\"test-123\"",
				"data-event-has-purchasable=\"true\"",
				"data-event-has-registration-fields=\"false\"",
			},
		},
		{
			name: "Valid event with registration fields",
			pageObj: helpers.SitePage{
				Name: "Event Attendees",
			},
			event: types.Event{
				Id:                    "test-456",
				Name:                  "Test Event",
				HasPurchasable:        false,
				HasRegistrationFields: true,
			},
			isEditor: true,
			expected: []string{
				"Event Attendees",
				"Registrations",
				"data-event-id=\"test-456\"",
				"data-event-has-purchasable=\"false\"",
				"data-event-has-registration-fields=\"true\"",
			},
		},
		{
			name: "Valid event with both purchasable and registration fields",
			pageObj: helpers.SitePage{
				Name: "Event Attendees",
			},
			event: types.Event{
				Id:                    "test-789",
				Name:                  "Test Event",
				HasPurchasable:        true,
				HasRegistrationFields: true,
			},
			isEditor: true,
			expected: []string{
				"Event Attendees",
				"Purchases",
				"Registrations",
				"data-event-id=\"test-789\"",
				"data-event-has-purchasable=\"true\"",
				"data-event-has-registration-fields=\"true\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventAttendeesPage(tt.pageObj, tt.event, tt.isEditor)

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
					t.Logf("Result: %s", result)
				}
			}
		})
	}
}
