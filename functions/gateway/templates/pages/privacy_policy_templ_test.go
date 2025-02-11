package pages

import (
	"bytes"
	"context"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
)

func TestPrivacyPolicy(t *testing.T) {
	// Create test data
	page := helpers.SitePage{
		Name: "Privacy Policy",
	}

	testCases := []struct {
		name            string
		expectedStrings []string
	}{
		{
			name: "Contains required privacy policy content",
			expectedStrings: []string{
				"Privacy Policy",
				"Last updated October 01, 2024",
				"TABLE OF CONTENTS",
				"brian@meetnear.me",
				"Google API Services User Data Policy",
				"Meet Near Me LLC",
				"Dover, DE 19904",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			component := PrivacyPolicyPage(page)

			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering PrivacyPolicy: %v", err)
			}

			renderedContent := buf.String()
			for _, expected := range tc.expectedStrings {
				if !bytes.Contains(buf.Bytes(), []byte(expected)) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
					t.Errorf("Rendered content:\n%s", renderedContent)
				}
			}
		})
	}
}
