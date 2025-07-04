package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAddEventSource(t *testing.T) {
	// Call the AddEventSource function
	component := AddEventSource()
	fakeContext := context.Background()
	// Add MNM_OPTIONS_CTX_KEY to context
	fakeContext = context.WithValue(fakeContext, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{})
	// Create a layout template
	layoutTemplate := Layout(helpers.SitePages["add-event-source"], helpers.UserInfo{}, component, types.Event{}, fakeContext, []string{})

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
		"Add an Event Source",
		"Add a Target URL",
		"Verify Events",
		"Add to Site",
		"event-source-steps",
		"event-source-container",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(renderedContent, expected) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
		}
	}
}
