package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAddEventSource(t *testing.T) {

	// Define test cases
	tests := []struct {
		name             string
		subnavItems      []string
		expectedContent  []string
		doNotShowContent []string
		event            types.Event
		pageUser         *types.UserSearchResult
	}{
		{
			name:        string("Event Details, registration / purchasable"),
			subnavItems: helpers.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"Main Nav",
				"John Doe",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
				">Checkout<",
				">Register<",
			},
			event:    types.Event{},
			pageUser: nil,
		},
		{
			name:        string("Event Details, with purchasable"),
			subnavItems: helpers.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"flyout-tab-cart",
				"John Doe",
				">Checkout<",
			},
			doNotShowContent: []string{
				">Register<",
			},
			event:    types.Event{HasPurchasable: true},
			pageUser: nil,
		},
		{
			name:        string("Event Details, with registration"),
			subnavItems: helpers.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"flyout-tab-cart",
				"John Doe",
				">Register<",
			},
			doNotShowContent: []string{
				">Checkout<",
			},
			event:    types.Event{HasRegistrationFields: true},
			pageUser: nil,
		},
		{
			name:        string("About page"),
			subnavItems: helpers.SitePages["about"].SubnavItems,
			expectedContent: []string{
				"John Doe",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
				"flyout-tab-filters",
			},
			event:    types.Event{HasPurchasable: true},
			pageUser: nil,
		},
		{
			name:        string("Home / event search page"),
			subnavItems: helpers.SitePages["home"].SubnavItems,
			expectedContent: []string{
				"John Doe",
				"flyout-tab-filters",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
			},
			event:    types.Event{HasPurchasable: true},
			pageUser: nil,
		}, {
			name:        string("Home / event search page, with pageUser"),
			subnavItems: helpers.SitePages["home"].SubnavItems,
			expectedContent: []string{
				"John Doe",
				"flyout-tab-filters",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
				"Sign Up",
			},
			event: types.Event{HasPurchasable: true},
			pageUser: &types.UserSearchResult{
				DisplayName: "John Doe",
			},
		},
	}

	mockUserInfo := helpers.UserInfo{
		Email:             "test@example.com",
		EmailVerified:     true,
		FamilyName:        "Doe",
		GivenName:         "John",
		Locale:            "en-US",
		Name:              "John Doe",
		PreferredUsername: "johndoe",
		Sub:               "user123",
		UpdatedAt:         1234567890,
		Metadata:          "",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := Navbar(mockUserInfo, tt.subnavItems, tt.event, tt.pageUser)
			// Render the template
			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)

			// Check for rendering errors
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Check if the rendered content contains expected information
			renderedContent := buf.String()

			for _, expected := range tt.expectedContent {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
				}
			}

			for _, unexpected := range tt.doNotShowContent {
				if strings.Contains(renderedContent, unexpected) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", unexpected)
				}
			}
		})
	}
}
