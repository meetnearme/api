package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNestedCheckboxList(t *testing.T) {

	// Define test cases
	tests := []struct {
		name            string
		isInDropdown    bool
		expectedContent []string
		interests       []string
	}{
		{
			name:            string("Nested checkbox list, in dropdown"),
			isInDropdown:    true,
			expectedContent: []string{"Civic &amp; Advocacy", "<summary"},
			interests:       []string{},
		},
		{
			name:            string("Nested checkbox list, solo"),
			isInDropdown:    false,
			expectedContent: []string{"Civic &amp; Advocacy"},
			interests:       []string{},
		},
		{
			name:         "Nested checkbox list with pre-selected interests",
			isInDropdown: true,
			expectedContent: []string{
				"Civic &amp; Advocacy",
				"checked=\"checked\"", // Verify checkbox is checked
			},
			interests: []string{"Civic & Advocacy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := NestedCheckboxList(tt.isInDropdown, tt.interests)
			// Render the template
			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)

			// Check for rendering errors
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Check if the rendered content contains expected information
			renderedContent := buf.String()

			for _, expected := range tt.expectedContent {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
				}
			}
		})
	}
}
