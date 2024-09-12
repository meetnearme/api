package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
)

func TestHomePage(t *testing.T) {
	// Mock data
	events := []services.Event{
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

	cfLocation := helpers.CdnLocation{
		City: "New York",
		CCA2: "US",
	}

	latStr := "40.7128"
	lonStr := "-74.0060"

	// Call the HomePage function
	component := HomePage(events, cfLocation, latStr, lonStr)

	// Render the component
	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering HomePage: %v", err)
	}

	// Check if the rendered content contains expected elements
	renderedContent := buf.String()
	expectedElements := []string{
		"Test Event 1",
		"Test Event 2",
		"New York, US",
		"40.7128",
		"-74.0060",
	}

	for _, element := range expectedElements {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}
}
