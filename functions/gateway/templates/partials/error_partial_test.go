package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestErrorHTML(t *testing.T) {
	// Sample input for the template
	body := []byte("An unexpected error occurred")
	reqID := "12345"

	// Call the ErrorHTML function to render the template
	component := ErrorHTML(body, reqID)

	// Create a buffer to store the rendered output
	var buf bytes.Buffer

	// Render the template
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check if the rendered content contains expected parts
	renderedContent := buf.String()

	// Define the expected content that should appear in the rendered HTML
	expectedContent := []string{
		"An unexpected error occurred",         // The body content
		"This error has been logged",           // Static text
		"Request ID: <strong>12345</strong>",   // The reqID content
		"alert alert-error",                    // The alert class
		"svg",                                  // The SVG tag for the icon
	}

	// Check if each expected string is present in the rendered content
	for _, expected := range expectedContent {
		if !strings.Contains(renderedContent, expected) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
		}
	}
}

