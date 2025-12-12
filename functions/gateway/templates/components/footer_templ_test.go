package components

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFooter(t *testing.T) {
	ctx := context.Background()

	t.Run("not logged in users see create account button", func(t *testing.T) {
		NOT_LOGGED_IN := false
		footer := Footer(NOT_LOGGED_IN)

		var buf bytes.Buffer
		err := footer.Render(ctx, &buf)

		if err != nil {
			t.Fatalf("Failed to render template: %v", err)
		}

		renderedContent := buf.String()
		expectedContent := []string{
			"Get local events straight to your inbox",
			"Get Personalized Events",
			"Follow Us",
			"Terms of Service",
			"Privacy Policy",
			"Music",
			"Party",
		}

		for _, element := range expectedContent {
			if !strings.Contains(renderedContent, element) {
				t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
			}
		}
	})

	t.Run("get personalized events button does not show for logged in users", func(t *testing.T) {
		LOGGED_IN := true
		footer := Footer(LOGGED_IN)

		var buf bytes.Buffer
		err := footer.Render(ctx, &buf)

		if err != nil {
			t.Fatalf("Failed to render template: %v", err)
		}

		renderedContent := buf.String()
		expectedContent := []string{
			"Follow Us",
			"Terms of Service",
			"Privacy Policy",
			"Music",
			"Party",
		}

		for _, element := range expectedContent {
			if !strings.Contains(renderedContent, element) {
				t.Errorf(`Expected "%s" to be rendered but it didn't.`, element)
			}
		}

		unexpectedContent := []string{
			"Get local events straight to your inbox",
			"Get Personalized Events",
		}

		for _, element := range unexpectedContent {
			if strings.Contains(renderedContent, element) {
				t.Errorf(`Did not expect "%s" to be rendered but it was.`, element)
			}
		}
	})

}
