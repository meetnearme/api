package pages

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestLayout(t *testing.T) {
	testCases := []struct {
		name            string
		event           types.Event
		sitePage        helpers.SitePage
		expectedStrings []string
	}{
		{
			name:  "Empty event",
			event: types.Event{},
			sitePage: helpers.SitePage{
				Key:  "event-detail",
				Name: "Events",
			},
			expectedStrings: []string{
				// NOTE: we check for UTF-8 because emojis in event descriptions render wrong without this character set
				`<meta charset="UTF-8"`,
				`<title>Meet Near Me - Events</title>`,
				// NOTE: the hash is generated in prod mode, but plain CSS is used in dev/test mode
				`<link rel="stylesheet" href="[^"]*styles[^"]*\.css"`,
				`<meta name="viewport" content="width=device-width, initial-scale=1"`,
				"hello world!",
			},
		},
		{
			name: "Populated event on event-detail page",
			event: types.Event{
				Id:   "123",
				Name: "Test Event 1",
			},
			sitePage: helpers.SitePage{
				Key:  "event-detail",
				Name: "Events",
			},
			expectedStrings: []string{
				// NOTE: we check for UTF-8 because emojis in event descriptions render wrong without this character set
				`<meta charset="UTF-8"`,
				`<title>Meet Near Me - Test Event 1</title>`,
				// NOTE: the hash is generated in prod mode, but plain CSS is used in dev/test mode
				`<link rel="stylesheet" href="[^"]*styles[^"]*\.css"`,
				`<meta name="viewport" content="width=device-width, initial-scale=1"`,
				"hello world!",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeContext := context.Background()
			// Add MNM_OPTIONS_CTX_KEY to context
			fakeContext = context.WithValue(fakeContext, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{
				"userId": "123",
				"--p":    "#000000",
			})
			component := Layout(tc.sitePage, helpers.UserInfo{}, templ.Raw("hello world!"), tc.event, false, fakeContext, []string{})

			var buf bytes.Buffer
			err := component.Render(fakeContext, &buf)
			if err != nil {
				t.Fatalf("Error rendering Layout: %v", err)
			}

			renderedContent := buf.String()
			for _, expected := range tc.expectedStrings {
				if !strings.Contains(renderedContent, expected) {
					// Replace Contains check with regex match for strings that contain regex patterns
					if strings.Contains(expected, "[") {
						matched, err := regexp.MatchString(expected, renderedContent)
						if err != nil {
							t.Errorf("Error matching regex: %v", err)
						}
						if !matched {
							t.Errorf("Expected rendered content to match pattern '%s', but it didn't", expected)
							t.Errorf("Rendered content:\n%s", renderedContent)
						}
						continue
					}
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
					t.Errorf("Rendered content:\n%s", renderedContent)
				}
			}
		})
	}
}
