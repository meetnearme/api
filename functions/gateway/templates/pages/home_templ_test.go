package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestHomePage(t *testing.T) {
	// Mock data
	events := []types.Event{
		{
			Id:          "123",
			Name:        "Test Event 1",
			Description: "Description for Test Event 1",
		},
		{
			Id:          "456",
			Name:        "Test Event 2",
			Description: "Description for Test Event 2",
		},
	}

	cfLocation := helpers.CdnLocation{
		City: "New York",
		CCA2: "US",
	}

	latStr := "40.7128"
	lonStr := "-74.0060"
	origLatStr := ""
	origLonStr := ""

	// Test cases
	tests := []struct {
		name          string
		pageUser      *helpers.UserSearchResult
		expectedItems []string
	}{
		{
			name:     "Without page user",
			pageUser: nil,
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
			},
		},
		{
			name: "With page user",
			// pageUser: &helpers.UserSearchResult{ID: "1234567890", DisplayName: "Test User", Meta: map[string]string{"about": "Welcome to Brian's Pub"}},
			pageUser: &helpers.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian Feister",
			},
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"data-page-user-id=\"1234567890\"",
			},
		},
		{
			name: "With page user and `about` section",
			pageUser: &helpers.UserSearchResult{
				UserID:      "1234567890",
				DisplayName: "Brian's Pub",
				Metadata: map[string]string{
					helpers.META_ABOUT_KEY: "Welcome to Brian's Pub",
				},
			},
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"data-page-user-id=\"1234567890\"",
				"Welcome to Brian&#39;s Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the HomePage function
			component := HomePage(events, tt.pageUser, cfLocation, latStr, lonStr, origLatStr, origLonStr)

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
