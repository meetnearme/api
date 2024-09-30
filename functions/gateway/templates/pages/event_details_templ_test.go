package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestEventDetailsPage(t *testing.T) {
	tm := "2099-05-01T12:00:00Z"
	validEventStartTime, err := helpers.UtcOrUnixToUnix64(tm)
	if err != nil || validEventStartTime == 0 {
		t.Logf("Failed to convert unix time to UTC: %v", tm)
	}
	tests := []struct {
		name     string
		event    types.Event
		checkoutParamVal string
		expected []string
	}{
		{
			name: "Valid event",
			event: types.Event{
				Id:          "123",
				Name:        "Test Event",
				Description: "This is a test event",
				Address:     "123 Test St",
				StartTime:   validEventStartTime,
			},
			checkoutParamVal:  "",
			expected: []string{
				"Test Event",
				"This is a test event",
				"123 Test St",
				"May 1, 2099",
				"12:00pm",
			},
		},
		{
			name:  "Empty event",
			event: types.Event{},
			expected: []string{
				"404 - Can't Find That Event",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventDetailsPage(tt.event, tt.checkoutParamVal)

			// Wrap the component with Layout
			layoutTemplate := Layout(helpers.SitePages["events"], helpers.UserInfo{}, component, types.Event{})

			// Render the component to a string
			var buf bytes.Buffer
			err := layoutTemplate.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering component: %v", err)
			}
			result := buf.String()
			// Check if all expected strings are in the result
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected string not found: %s", exp)
				}
			}
		})
	}
}
