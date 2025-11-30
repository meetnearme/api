package components

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
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
			component := Navbar(mockUserInfo, tt.subnavItems, tt.event, context.Background(), false)
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

func TestGetUserPageURL(t *testing.T) {
	// Save original APEX_URL
	originalApexURL := os.Getenv("APEX_URL")
	defer func() {
		if originalApexURL != "" {
			os.Setenv("APEX_URL", originalApexURL)
		} else {
			os.Unsetenv("APEX_URL")
		}
	}()

	tests := []struct {
		name           string
		userInfo       constants.UserInfo
		userMetaClaims map[string]interface{}
		apexURL        string
		expectedURL    string
		expectedPrefix string // For partial matching (when APEX_URL varies)
	}{
		{
			name: "With subdomain in userMetaClaims",
			userInfo: constants.UserInfo{
				Sub: "user123",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("mysubdomain")),
			},
			apexURL:        "https://example.com",
			expectedURL:    "https://mysubdomain.example.com",
			expectedPrefix: "",
		},
		{
			name: "Without userMetaClaims in context",
			userInfo: constants.UserInfo{
				Sub: "user456",
			},
			userMetaClaims: nil,
			apexURL:        "https://example.com",
			expectedURL:    "/user/user456",
			expectedPrefix: "",
		},
		{
			name: "With subdomain but different APEX_URL format",
			userInfo: constants.UserInfo{
				Sub: "user789",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("test")),
			},
			apexURL:        "https://meetnearme.com",
			expectedURL:    "https://test.meetnearme.com",
			expectedPrefix: "",
		},
		{
			name: "Without subdomain with empty user Sub",
			userInfo: constants.UserInfo{
				Sub: "",
			},
			userMetaClaims: nil,
			apexURL:        "https://example.com",
			expectedURL:    "/user/",
			expectedPrefix: "",
		},
		{
			name: "With empty subdomain in userMetaClaims",
			userInfo: constants.UserInfo{
				Sub: "user999",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("")),
			},
			apexURL:        "https://example.com",
			expectedURL:    "/user/user999",
			expectedPrefix: "",
		},
		{
			name: "With subdomain key missing from userMetaClaims",
			userInfo: constants.UserInfo{
				Sub: "user111",
			},
			userMetaClaims: map[string]interface{}{
				"other_key": "some_value",
			},
			apexURL:        "https://example.com",
			expectedURL:    "/user/user111",
			expectedPrefix: "",
		},
		{
			name: "With non-string value in userMetaClaims for subdomain",
			userInfo: constants.UserInfo{
				Sub: "user111",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: 12345,
			},
			apexURL:        "https://example.com",
			expectedURL:    "/user/user111",
			expectedPrefix: "",
		},
		{
			name: "Multiple subdomains in subdomain value",
			userInfo: constants.UserInfo{
				Sub: "user222",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("multi.sub.domain")),
			},
			apexURL:        "https://example.com",
			expectedURL:    "https://multi.sub.domain.example.com",
			expectedPrefix: "",
		},
		{
			name: "With invalid base64 subdomain value",
			userInfo: constants.UserInfo{
				Sub: "user333",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: "invalid-base64!@#",
			},
			apexURL:        "https://example.com",
			expectedURL:    "/user/user333",
			expectedPrefix: "",
		},
		{
			name: "With empty userMetaClaims map",
			userInfo: constants.UserInfo{
				Sub: "user444",
			},
			userMetaClaims: map[string]interface{}{},
			apexURL:        "https://example.com",
			expectedURL:    "/user/user444",
			expectedPrefix: "",
		},
		{
			name: "With subdomain but APEX_URL is localhost:8000 (no dot)",
			userInfo: constants.UserInfo{
				Sub: "user555",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("mysubdomain")),
			},
			apexURL:        "localhost:8000",
			expectedURL:    "/user/user555",
			expectedPrefix: "",
		},
		{
			name: "With subdomain but APEX_URL is http://localhost:8000 (no dot)",
			userInfo: constants.UserInfo{
				Sub: "user666",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("mysubdomain")),
			},
			apexURL:        "http://localhost:8000",
			expectedURL:    "/user/user666",
			expectedPrefix: "",
		},
		{
			name: "With subdomain but APEX_URL is https://localhost:8000 (no dot)",
			userInfo: constants.UserInfo{
				Sub: "user777",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("mysubdomain")),
			},
			apexURL:        "https://localhost:8000",
			expectedURL:    "/user/user777",
			expectedPrefix: "",
		},
		{
			name: "Without subdomain and APEX_URL is localhost:8000 (no dot)",
			userInfo: constants.UserInfo{
				Sub: "user888",
			},
			userMetaClaims: nil,
			apexURL:        "localhost:8000",
			expectedURL:    "/user/user888",
			expectedPrefix: "",
		},
		{
			name: "With subdomain and APEX_URL with dot but http protocol (not handled, falls back to relative path)",
			userInfo: constants.UserInfo{
				Sub: "user999",
			},
			userMetaClaims: map[string]interface{}{
				constants.SUBDOMAIN_KEY: base64.StdEncoding.EncodeToString([]byte("mysubdomain")),
			},
			apexURL:        "http://example.com",
			expectedURL:    "/user/user999",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set APEX_URL for this test
			os.Setenv("APEX_URL", tt.apexURL)

			// Create context with userMetaClaims if provided
			ctx := context.Background()
			if tt.userMetaClaims != nil {
				ctx = context.WithValue(ctx, "userMetaClaims", tt.userMetaClaims)
			}

			result := GetUserPageURL(tt.userInfo, ctx)

			if tt.expectedPrefix != "" {
				// Partial match for cases where exact URL might vary
				if !strings.HasPrefix(result, tt.expectedPrefix) {
					t.Errorf("getUserPageURL() = %q, want prefix %q", result, tt.expectedPrefix)
				}
			} else {
				// Exact match
				if result != tt.expectedURL {
					t.Errorf("getUserPageURL() = %q, want %q", result, tt.expectedURL)
				}
			}
		})
	}
}
