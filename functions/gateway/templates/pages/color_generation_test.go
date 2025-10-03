package pages

import (
	"strings"
	"testing"
)

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
			expectSame:  true,
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
