package services

import (
	_ "embed"
	"errors"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/types"
)

//go:embed geo_service_test_mock1.html
var mockHTML1 string

//go:embed geo_service_test_mock2.html
var mockHTML2 string

type mockHTMLFetcher struct {
	HTMLResponse string
	Error        error
}

func (m *mockHTMLFetcher) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return m.HTMLResponse, m.Error
}

func TestGeoService(t *testing.T) {
	origEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", origEnv)
	os.Setenv("GO_ENV", "test")

	tests := []struct {
		name          string
		location      string
		baseURL       string
		mockHTML      string
		mockError     error
		expectedLat   string
		expectedLon   string
		expectedAddr  string
		expectedError bool
	}{
		{
			name:          "successful geocoding with city only",
			location:      "Georgetown, TX", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      mockHTML1,
			mockError:     nil,
			expectedLat:   "30.6332618",
			expectedLon:   "-97.6779842",
			expectedAddr:  "Georgetown, TX",
			expectedError: false,
		},
		{
			name:          "successful geocoding with full address",
			location:      "3400 Wolf Ranch Pkwy, Georgetown, Texas", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      mockHTML2,
			mockError:     nil,
			expectedLat:   "30.6272609",
			expectedLon:   "-97.71859189999999",
			expectedAddr:  "3400 Wolf Ranch Parkway, Georgetown, TX 78628",
			expectedError: false,
		},
		{
			name:     "successful geocoding with newline in HTML",
			location: "Doesntmatter, NY",
			baseURL:  "https://example.com",
			mockHTML: `random words "New York,
			                  NY 10001, USA", [40.712800, -74.006000] random words []]`,
			mockError:     nil,
			expectedLat:   "40.712800",
			expectedLon:   "-74.006000",
			expectedAddr:  "New York, NY 10001, USA",
			expectedError: false,
		},
		{
			name:          "HTML fetch error",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      "",
			mockError:     errors.New("network error"),
			expectedLat:   "",
			expectedLon:   "",
			expectedAddr:  "",
			expectedError: true,
		},
		{
			name:          "invalid HTML format - no coordinates",
			location:      "Invalid Location",
			baseURL:       "https://example.com",
			mockHTML:      `"Some text without coordinates"`,
			mockError:     nil,
			expectedLat:   "",
			expectedLon:   "",
			expectedAddr:  "",
			expectedError: true,
		},
		{
			name:          "valid coordinates but no address",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      `[40.712800, -74.006000]`,
			mockError:     nil,
			expectedLat:   "40.712800",
			expectedLon:   "-74.006000",
			expectedAddr:  "No address found",
			expectedError: false,
		},
		{
			name:          "out of bounds longitude is not captured",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      `[40.7, 181.006000]`,
			mockError:     nil,
			expectedLat:   "",
			expectedLon:   "",
			expectedAddr:  "",
			expectedError: true,
		},
		{
			name:          "empty base URL",
			location:      "Doesntmatter",
			baseURL:       "",
			mockHTML:      "",
			mockError:     nil,
			expectedLat:   "",
			expectedLon:   "",
			expectedAddr:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFetcher := &mockHTMLFetcher{
				HTMLResponse: tt.mockHTML,
				Error:        tt.mockError,
			}

			service := &RealGeoService{
				htmlFetcher: mockFetcher,
			}

			lat, lon, address, err := service.GetGeo(tt.location, tt.baseURL)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if lat != tt.expectedLat {
				t.Errorf("Expected lat %s, got %s", tt.expectedLat, lat)
			}
			if lon != tt.expectedLon {
				t.Errorf("Expected lon %s, got %s", tt.expectedLon, lon)
			}
			if address != tt.expectedAddr {
				t.Errorf("Expected address %#v, got %#v", tt.expectedAddr, address)
			}
		})
	}
}
