package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFooter(t *testing.T) {
	ctx := context.Background()

	footer := Footer()

	var buf bytes.Buffer
	err := footer.Render(ctx, &buf)

	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	renderedContent := buf.String()
	expectedContent := []string{
		"Get local events straight to your inbox",
		"Follow Us",
		"Terms of Service",
		"Privacy Policy",
		"Music",
		"Party",
		"News",
	}

	for _, element := range expectedContent {
		if !strings.Contains(renderedContent, element) {
			t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
		}
	}
}
