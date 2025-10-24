package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
)

func TestAdminLeftNavContents(t *testing.T) {
	// Create mock user info
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
	}

	// Test cases for different role scenarios
	tests := []struct {
		name             string
		roleClaims       []constants.RoleClaim
		expectedLinks    []string
		notExpectedLinks []string
	}{
		{
			name:       "User with no special roles",
			roleClaims: []constants.RoleClaim{},
			expectedLinks: []string{
				"<a href=\"/admin/home\">Home</a>",
				"<a href=\"/admin/profile/settings\">Interests</a>",
				"<a>Add Event (Soon)</a>",
				"<a>Host a Competition (Soon)</a>",
			},
			notExpectedLinks: []string{
				"<a href=\"/admin/event/new\">Create Event</a>",
				"<a href=\"/admin/competition/new\">Create Competition</a>",
			},
		},
		{
			name: "User with event admin role",
			roleClaims: []constants.RoleClaim{
				{Role: constants.Roles[constants.EventAdmin], ProjectID: "project-id"},
			},
			expectedLinks: []string{
				"<a href=\"/admin/home\">Home</a>",
				"<a href=\"/admin/profile/settings\">Interests</a>",
				"<a href=\"/admin/event/new\">Create Event</a>",
				"<a>Host a Competition (Soon)</a>",
			},
			notExpectedLinks: []string{
				"<a href=\"/admin/competition/new\">Create Competition</a>",
			},
		},
		{
			name: "User with super admin role",
			roleClaims: []constants.RoleClaim{
				{Role: constants.Roles[constants.SuperAdmin], ProjectID: "project-id"},
			},
			expectedLinks: []string{
				"<a href=\"/admin/home\">Home</a>",
				"<a href=\"/admin/profile/settings\">Interests</a>",
				"<a href=\"/admin/event/new\">Create Event</a>",
				"<a href=\"/admin/competition/new\">Create Competition</a>",
			},
			notExpectedLinks: []string{
				"<a>Add Event (Soon)</a>",
				"<a>Host a Competition (Soon)</a>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with user info and role claims
			ctx := context.WithValue(context.Background(), "userInfo", mockUserInfo)
			ctx = context.WithValue(ctx, "roleClaims", tt.roleClaims)

			// Call the actual template function
			navContent := AdminLeftNavContents(ctx)

			// Render the template
			var buf bytes.Buffer
			err := navContent.Render(ctx, &buf)
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Get the rendered content
			renderedContent := buf.String()

			// Check for expected links
			for _, link := range tt.expectedLinks {
				if !strings.Contains(renderedContent, link) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", link)
				}
			}

			// Check for links that should not be present
			for _, link := range tt.notExpectedLinks {
				if strings.Contains(renderedContent, link) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", link)
				}
			}

			// Check for proper section headers
			expectedHeaders := []string{
				"<h3 class=\"font-bold menu-title my-2\">Admin</h3>",
				"<h3 class=\"font-bold menu-title my-2\">Events</h3>",
				"<h3 class=\"font-bold menu-title my-2\">Competitions</h3>",
			}
			for _, header := range expectedHeaders {
				if !strings.Contains(renderedContent, header) {
					t.Errorf("Expected rendered content to contain header '%s', but it didn't", header)
				}
			}
		})
	}
}

func TestAdminNav(t *testing.T) {
	// Create mock user info
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
	}

	// Create context with user info
	ctx := context.WithValue(context.Background(), "userInfo", mockUserInfo)

	// Call the actual template function
	navContent := AdminNav(ctx)

	// Render the template
	var buf bytes.Buffer
	err := navContent.Render(ctx, &buf)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Get the rendered content
	renderedContent := buf.String()

	// Check for proper container structure
	expectedContainer := `<div class="self-start sticky top-0 col-span-2 md:mr-5 mb-5 card border border-base-300 bg-base-200 rounded-box md:place-items-center ">`
	if !strings.Contains(renderedContent, expectedContainer) {
		t.Error("Expected proper container structure with correct classes")
	}

	// Check for proper menu structure
	expectedMenu := `<ul class="menu bg-base-200 rounded-box w-56">`
	if !strings.Contains(renderedContent, expectedMenu) {
		t.Error("Expected proper menu structure with correct classes")
	}

	// Check that the nav content is included
	if !strings.Contains(renderedContent, "<h3 class=\"font-bold menu-title my-2\">Admin</h3>") {
		t.Error("Expected nav content to be included in the rendered output")
	}
}
