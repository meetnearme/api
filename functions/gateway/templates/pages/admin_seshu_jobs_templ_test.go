package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/types"
)

func TestAdminSeshuJobsPage_EmptyState(t *testing.T) {
	tests := []struct {
		name          string
		jobs          []types.SeshuJob
		isSuperAdmin  bool
		expectedItems []string
		notExpected   []string
	}{
		{
			name:         "Empty jobs shows empty state message",
			jobs:         []types.SeshuJob{},
			isSuperAdmin: false,
			expectedItems: []string{
				"Event Source URLs",
				"Add New Source",
				"You don't have any event sources yet.",
				"Add a new source",
				"/admin/add-event-source",
				"to get started.",
			},
			notExpected: []string{
				"seshu-jobs-content", // The content div should not appear when empty
			},
		},
		{
			name:         "Empty jobs as superAdmin shows empty state",
			jobs:         []types.SeshuJob{},
			isSuperAdmin: true,
			expectedItems: []string{
				"You don't have any event sources yet.",
				"Add a new source",
			},
		},
		{
			name: "With jobs shows job content not empty state",
			jobs: []types.SeshuJob{
				{
					NormalizedUrlKey: "https://example.com/events",
					Status:           "HEALTHY",
					OwnerID:          "user-123",
				},
			},
			isSuperAdmin: false,
			expectedItems: []string{
				"Event Source URLs",
				"seshu-jobs-content", // Content div should appear with jobs
			},
			notExpected: []string{
				"You don't have any event sources yet.", // Empty state should not appear
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := AdminSeshuJobsPage(tt.jobs, 1, 10, 1, len(tt.jobs), tt.isSuperAdmin)

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering AdminSeshuJobsPage: %v", err)
			}

			renderedContent := buf.String()

			// Check for expected items
			for _, expected := range tt.expectedItems {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
					t.Logf("Rendered content:\n%s", renderedContent)
				}
			}

			// Check that not expected items are absent
			for _, notExpected := range tt.notExpected {
				if strings.Contains(renderedContent, notExpected) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", notExpected)
				}
			}
		})
	}
}

func TestAdminSeshuJobsPage_TableStructure(t *testing.T) {
	// Test that empty state has proper table structure with correct colspan
	component := AdminSeshuJobsPage([]types.SeshuJob{}, 1, 10, 1, 0, false)

	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Error rendering AdminSeshuJobsPage: %v", err)
	}

	renderedContent := buf.String()

	// Verify table header columns exist
	expectedHeaders := []string{
		"Status",
		"URL",
		"Type",
		"Location",
		"Scan",
		"Failures",
		"Actions",
	}

	for _, header := range expectedHeaders {
		if !strings.Contains(renderedContent, header) {
			t.Errorf("Expected table header '%s' to exist, but it didn't", header)
		}
	}

	// Verify colspan="7" is used for empty state to span all columns
	if !strings.Contains(renderedContent, `colspan="7"`) {
		t.Errorf("Expected empty state td to have colspan=\"7\" to span all columns")
	}
}

