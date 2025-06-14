package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAdminPage(t *testing.T) {
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

	mockRoleClaims := []helpers.RoleClaim{
		{
			Role:        "orgAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "superAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "sysAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
	}

	interests := []string{"Concerts", "Photography"}
	subdomain := "brians-pub"

	// Create a layout template
	fakeContext := context.Background()
	fakeContext = context.WithValue(fakeContext, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123", "--p": "#000000", "themeMode": "dark"})

	// Call the AdminPage function
	profilePage := AdminPage(mockUserInfo, mockRoleClaims, interests, subdomain, "userId=123;--p=#000000;themeMode=dark", "Test about me text", context.Background())

	layoutTemplate := Layout(helpers.SitePages["profile"], mockUserInfo, profilePage, types.Event{}, fakeContext, []string{})

	// Render the template
	var buf bytes.Buffer
	err := layoutTemplate.Render(fakeContext, &buf)

	// Check for rendering errors
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check if the rendered content contains expected information
	renderedContent := buf.String()
	expectedContent := []string{
		mockUserInfo.Email,
		mockUserInfo.Name,
		mockUserInfo.Sub,
		mockUserInfo.Locale,
		"Yes", // mockUserInfo.EmailVerified yields a value "Yes" or "No"
		"Test about me text",
		"#000000",
	}

	for _, element := range expectedContent {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}
}
