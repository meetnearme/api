package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestCompetitionConfigAdminList(t *testing.T) {
	// Create mock competition configs
	mockConfigs := []types.CompetitionConfig{
		{
			Id:         "config1",
			Name:       "Test Competition 1",
			StartTime:  1704067200, // Jan 1, 2024
			EndTime:    1706745600, // Feb 1, 2024
			ModuleType: constants.ES_SERIES_PARENT,
			CreatedAt:  1704067200,
			UpdatedAt:  1704067200,
		},
		{
			Id:         "config2",
			Name:       "Test Competition 2",
			StartTime:  1706745600, // Feb 1, 2024
			EndTime:    1709251200, // Mar 1, 2024
			ModuleType: constants.ES_EVENT_SERIES,
			CreatedAt:  1706745600,
			UpdatedAt:  1706745600,
		},
	}

	// Call the CompetitionConfigAdminList function
	adminList := CompetitionConfigAdminList(&mockConfigs)

	// Render the template
	var buf bytes.Buffer
	err := adminList.Render(context.Background(), &buf)

	// Check for rendering errors
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Check if the rendered content contains expected information
	renderedContent := buf.String()
	expectedContent := []string{
		"Competition Admin",
		"Test Competition 1",
		"Test Competition 2",
		"Series",
		"Event",
		"/admin/competition/config1/edit",
		"/admin/competition/config2/edit",
	}

	for _, element := range expectedContent {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}

	// Verify table structure
	expectedTableElements := []string{
		"<table",
		"<thead>",
		"<tbody>",
		"<tr",
		"<th colspan=\"5\">Competition</th>",
		"<th>Start</th>",
		"<th>End</th>",
		"<th>Type</th>",
		"<th>Created</th>",
		"<th>Updated</th>",
	}

	for _, element := range expectedTableElements {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain table element '%s', but it didn't", element)
		}
	}
}
