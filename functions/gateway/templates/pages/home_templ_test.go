package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestHomePage(t *testing.T) {
	// Mock data
	events := []types.Event{
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

	// NOTE: we need an additional test to cover the scenario where there is discrepancy
	// between cfLocation from cloudflare for lat / lon and query params set by user in URL
	origLatStr := ""
	origLonStr := ""

	// Call the HomePage function
	component := HomePage(events, cfLocation, latStr, lonStr, origLatStr, origLonStr)

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
	}

	for _, element := range expectedElements {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}
}
