package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
)

func TestEventDetailsPage(t *testing.T) {
	tests := []struct {
		name     string
		event    services.Event
		expected []string
	}{
		{
			name: "Valid event",
			event: services.Event{
				Id:          "123",
				Name:        "Test Event",
				Description: "This is a test event",
				Address:     "123 Test St",
				StartTime:   "2023-05-01T12:00:00Z",
			},
			expected: []string{
				"Test Event",
				"This is a test event",
				"123 Test St",
				"May 1, 2023",
				"12:00pm",
			},
		},
		{
			name:  "Empty event",
			event: services.Event{},
			expected: []string{
				"404 - Can't Find That Event",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventDetailsPage(tt.event)

			// Wrap the component with Layout
			layoutTemplate := Layout("Event Details", helpers.UserInfo{}, component)

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