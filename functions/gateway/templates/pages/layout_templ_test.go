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

func TestGenerateSecondaryColor(t *testing.T) {
	testCases := []struct {
		name           string
		primaryHex     string
		colorScheme    string
		expectedResult string
		expectSame     bool
	}{
		{
			name:        "Dark primary on light background - good contrast",
			primaryHex:  "#000000", // Black
			colorScheme: "light",
			expectSame:  true,
		},
		{
			name:        "Light primary on light background - poor contrast",
			primaryHex:  "#ffffff", // White
			colorScheme: "light",
			expectSame:  false,
		},
		{
			name:        "Dark primary on dark background - poor contrast",
			primaryHex:  "#000000", // Black
			colorScheme: "dark",
			expectSame:  false,
		},
		{
			name:        "Medium primary on light background - poor contrast",
			primaryHex:  "#6366f1", // Indigo
			colorScheme: "light",
			expectSame:  false,
		},
		{
			name:        "Medium primary on dark background - poor contrast",
			primaryHex:  "#6366f1", // Indigo
			colorScheme: "dark",
			expectSame:  false,
		},
		{
			name:        "Very light primary on light background - poor contrast",
			primaryHex:  "#f0f0f0", // Very light gray
			colorScheme: "light",
			expectSame:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := generateSecondaryColor(tc.primaryHex, tc.colorScheme)
			if err != nil {
				t.Fatalf("generateSecondaryColor failed: %v", err)
			}

			if tc.expectSame {
				if result != tc.primaryHex {
					t.Errorf("Expected secondary to be same as primary (%s), got %s", tc.primaryHex, result)
				}
			} else {
				if result == tc.primaryHex {
					t.Errorf("Expected secondary to be different from primary (%s), but got the same", tc.primaryHex)
				}
			}
		})
	}
}

func TestCalculateContrast(t *testing.T) {
	testCases := []struct {
		name        string
		color1      string
		color2      string
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "Black and white - maximum contrast",
			color1:      "#000000",
			color2:      "#ffffff",
			expectedMin: 20.0,
			expectedMax: 21.0,
		},
		{
			name:        "Same color - no contrast",
			color1:      "#000000",
			color2:      "#000000",
			expectedMin: 1.0,
			expectedMax: 1.0,
		},
		{
			name:        "Similar colors - low contrast",
			color1:      "#ffffff",
			color2:      "#f0f0f0",
			expectedMin: 1.0,
			expectedMax: 2.0,
		},
		{
			name:        "Medium contrast colors",
			color1:      "#000000",
			color2:      "#808080",
			expectedMin: 5.0,
			expectedMax: 6.0,
		},
		{
			name:        "Primary Indigo and white background",
			color1:      "#6366f1",
			color2:      "#ffffff",
			expectedMin: 4.0,
			expectedMax: 5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contrast, err := calculateContrast(tc.color1, tc.color2)
			if err != nil {
				t.Fatalf("calculateContrast failed: %v", err)
			}

			if contrast < tc.expectedMin || contrast > tc.expectedMax {
				t.Errorf("Expected contrast between %.2f and %.2f, got %.2f", tc.expectedMin, tc.expectedMax, contrast)
			}
		})
	}
}

func TestHexToOklch(t *testing.T) {
	testCases := []struct {
		name      string
		hex       string
		expectedL string
		expectedC string
		expectedH string
	}{
		{
			name:      "Black to OKLCH",
			hex:       "#000000",
			expectedL: "0.0000",
			expectedC: "0.0000",
			expectedH: "0.000",
		},
		{
			name:      "White to OKLCH",
			hex:       "#ffffff",
			expectedL: "1.0000",
			expectedC: "0.0000",
			expectedH: "89.876",
		},
		{
			name:      "Red to OKLCH",
			hex:       "#ff0000",
			expectedL: "0.6280",
			expectedC: "0.2577",
			expectedH: "29.234",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hexToOklch(tc.hex)

			// Parse the OKLCH string (format: "L C H")
			parts := strings.Fields(result)
			if len(parts) != 3 {
				t.Fatalf("Expected 3 parts in OKLCH result, got %d: %s", len(parts), result)
			}

			if parts[0] != tc.expectedL {
				t.Errorf("Expected L value '%s', got '%s'", tc.expectedL, parts[0])
			}
			if parts[1] != tc.expectedC {
				t.Errorf("Expected C value '%s', got '%s'", tc.expectedC, parts[1])
			}
			if parts[2] != tc.expectedH {
				t.Errorf("Expected H value '%s', got '%s'", tc.expectedH, parts[2])
			}
		})
	}
}
