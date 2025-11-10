package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
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

	cfLocation := constants.CdnLocation{
		City: "New York",
		CCA2: "US",
	}

	origLatStr := ""
	origLonStr := ""
	// Test cases
	tests := []struct {
		name              string
		pageUser          *types.UserSearchResult
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
			cityStr:           "Houston",
			latStr:            "29.760427",
			lonStr:            "-95.369803",
			origQueryLocation: "Houston, TX",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"data-city-label-initial=\"Houston, TX\"",
				"data-page-user-id=\"1234567890\"",
				"data-city-latitude-initial=\"29.760427\"",
				"data-city-longitude-initial=\"-95.369803\"",
				"Welcome to Brian's Pub",
				"Brian&#39;s Pub</h1>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the HomePage function
			component := HomePage(context.Background(), events, tt.pageUser, cfLocation, tt.cityStr, tt.latStr, tt.lonStr, origLatStr, origLonStr, tt.origQueryLocation)

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
	events := []types.Event{
		{
			Id:          "123",
			Name:        "Test Event 1",
			Description: "Description for Test Event 1",
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
