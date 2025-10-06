package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAboutPage(t *testing.T) {
	// Create mock user info
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
	fakeContext := context.Background()
	fakeContext = context.WithValue(fakeContext, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123", "--p": "#000000", "themeMode": "dark"})

	// Call the AboutPage function
	aboutPage := AboutPage()
	// Create a layout template
	layoutTemplate := Layout(helpers.SitePages["admin"], mockUserInfo, aboutPage, types.Event{}, false, fakeContext, []string{})

	// Render the template using the same context
	var buf bytes.Buffer
	err := layoutTemplate.Render(fakeContext, &buf)

	// Check for rendering errors
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check if the rendered content contains expected information
	renderedContent := buf.String()
	expectedContent := []string{
		"Our Why",
	}

	for _, element := range expectedContent {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}
}
