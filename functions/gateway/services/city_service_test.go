package services

import (
	_ "embed"
	"errors"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/types"
)

//go:embed geo_service_test_mock_html1.html
var cityMockHTML1 string

//go:embed geo_service_test_mock_html2.html
var cityMockHTML2 string

type cityMockHTMLFetcher struct {
	HTMLResponse string
	Error        error
}

func (m *cityMockHTMLFetcher) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return m.HTMLResponse, m.Error
}

func TestCityService(t *testing.T) {
	origEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", origEnv)
	os.Setenv("GO_ENV", "test")

	tests := []struct {
		name          string
		location      string
		baseURL       string
		mockHTML      string
		mockError     error
		expectedCity  string
		expectedError bool
	}{
		{
			name:          "successful geocoding with city only",
			location:      "Georgetown, TX", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      cityMockHTML1,
			mockError:     nil,
			expectedCity:  "Georgetown, TX",
			expectedError: false,
		},
		{
			name:          "successful geocoding with full address",
			location:      "3400 Wolf Ranch Pkwy, Georgetown, Texas", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      cityMockHTML2,
			mockError:     nil,
			expectedCity:  "Georgetown, TX",
			expectedError: false,
		},
		{
			name:     "successful geocoding with newline in HTML",
			location: "Doesntmatter, NY",
			baseURL:  "https://example.com",
			mockHTML: `random words "New York,
			                  NY 10001, USA", [40.712800, -74.006000] random words []]`,
			mockError:     nil,
			expectedCity:  "New York, NY",
			expectedError: false,
		},
		{
			name:          "HTML fetch error",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      "",
			mockError:     errors.New("network error"),
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "invalid HTML format - no coordinates",
			location:      "Invalid Location",
			baseURL:       "https://example.com",
			mockHTML:      `"Some text without coordinates"`,
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "valid coordinates but no address",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      `[40.712800, -74.006000]`,
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "out of bounds longitude is not captured",
			location:      "Doesntmatter",
			baseURL:       "https://example.com",
			mockHTML:      `[40.7, 181.006000]`,
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "empty base URL",
			location:      "Doesntmatter",
			baseURL:       "",
			mockHTML:      "",
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFetcher := &cityMockHTMLFetcher{
				HTMLResponse: tt.mockHTML,
				Error:        tt.mockError,
			}

			service := &RealCityService{
				htmlFetcher: mockFetcher,
			}

			city, err := service.GetCity(tt.location, tt.baseURL)

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

			if city != tt.expectedCity {
				t.Errorf("Expected %s, got %s", tt.expectedCity, city)
			}
		})
	}
}
