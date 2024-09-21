package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSuccessBannerHTML(t *testing.T) {
	// Sample input for the template
	msg := "Operation completed successfully"

	// Call the SuccessBannerHTML function to render the template
	component := SuccessBannerHTML(msg)

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
		"Operation completed successfully",    // The success message
		"alert alert-success",                 // The alert class
		"svg",                                 // The SVG tag for the icon
	}

	// Check if each expected string is present in the rendered content
	for _, expected := range expectedContent {
		if !strings.Contains(renderedContent, expected) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
		}
	}
}

