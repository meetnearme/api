package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestGetFirstChar(t *testing.T) {
	tests := []struct {
		name     string
		userInfo constants.UserInfo
		expected string
	}{
		{
			name: "ASCII uppercase",
			userInfo: constants.UserInfo{
				Name: "John Doe",
			},
			expected: "J",
		},
		{
			name: "ASCII lowercase",
			userInfo: constants.UserInfo{
				Name: "john doe",
			},
			expected: "J",
		},
		{
			name: "Spanish name with accent",
			userInfo: constants.UserInfo{
				Name: "Énrique García",
			},
			expected: "É",
		},
		{
			name: "Chinese name",
			userInfo: constants.UserInfo{
				Name: "张伟",
			},
			expected: "张",
		},
		{
			name: "Emoji flag as first character",
			userInfo: constants.UserInfo{
				Name: "🇨🇦 John Smith",
			},
			expected: "🇨",
		},
		{
			name: "Emoji name",
			userInfo: constants.UserInfo{
				Name: "😀 Happy User",
			},
			expected: "😀",
		},
		{
			name: "Emoji name with multiple emojis",
			userInfo: constants.UserInfo{
				Name: "🎉✨ Party Person",
			},
			expected: "🎉",
		},
		{
			name: "Name with leading whitespace",
			userInfo: constants.UserInfo{
				Name: "  Alice Brown",
			},
			expected: "A",
		},
		{
			name: "Name with trailing whitespace",
			userInfo: constants.UserInfo{
				Name: "Bob Wilson  ",
			},
			expected: "B",
		},
		{
			name: "Name with surrounding whitespace",
			userInfo: constants.UserInfo{
				Name: "  Carol White  ",
			},
			expected: "C",
		},
		{
			name: "Empty string",
			userInfo: constants.UserInfo{
				Name: "",
			},
			expected: "?",
		},
		{
			name: "Whitespace only",
			userInfo: constants.UserInfo{
				Name: "   ",
			},
			expected: "?",
		},
		{
			name: "Single emoji",
			userInfo: constants.UserInfo{
				Name: "🎵",
			},
			expected: "🎵",
		},
		{
			name: "Japanese name",
			userInfo: constants.UserInfo{
				Name: "山田太郎",
			},
			expected: "山",
		},
		{
			name: "Russian name",
			userInfo: constants.UserInfo{
				Name: "Александр",
			},
			expected: "А",
		},
		{
			name: "Arabic name",
			userInfo: constants.UserInfo{
				Name: "أحمد محمد",
			},
			expected: "أ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFirstChar(tt.userInfo)
			if result != tt.expected {
				t.Errorf("getFirstChar(%q) = %q, want %q", tt.userInfo.Name, result, tt.expected)
			}
		})
	}
}

func TestAddEventSource(t *testing.T) {

	// Define test cases
	tests := []struct {
		name             string
		subnavItems      []string
		expectedContent  []string
		doNotShowContent []string
		event            types.Event
	}{
		{
			name:        string("Event Details, registration / purchasable"),
			subnavItems: constants.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"Main Nav",
				"John Doe",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
				">Checkout<",
				">Register<",
			},
			event: types.Event{},
		},
		{
			name:        string("Event Details, with purchasable"),
			subnavItems: constants.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"flyout-tab-cart",
				"John Doe",
				">Checkout<",
			},
			doNotShowContent: []string{
				">Register<",
			},
			event: types.Event{HasPurchasable: true},
		},
		{
			name:        string("Event Details, with registration"),
			subnavItems: constants.SitePages["event-detail"].SubnavItems,
			expectedContent: []string{
				"flyout-tab-cart",
				"John Doe",
				">Register<",
			},
			doNotShowContent: []string{
				">Checkout<",
			},
			event: types.Event{HasRegistrationFields: true},
		},
		{
			name:        string("About page"),
			subnavItems: constants.SitePages["about"].SubnavItems,
			expectedContent: []string{
				"John Doe",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
				"flyout-tab-filters",
			},
			event: types.Event{HasPurchasable: true},
		},
		{
			name:        string("Home / event search page"),
			subnavItems: constants.SitePages["home"].SubnavItems,
			expectedContent: []string{
				"John Doe",
				"flyout-tab-filters",
			},
			doNotShowContent: []string{
				"flyout-tab-cart",
			},
			event: types.Event{HasPurchasable: true},
		},
	}

	mockUserInfo := constants.UserInfo{
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
			component := Navbar(mockUserInfo, tt.subnavItems, tt.event, context.Background())
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
