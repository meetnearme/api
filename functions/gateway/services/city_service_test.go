package services

import (
	_ "embed"
	"errors"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/types"
)

//go:embed city_service_test_mock_html1.html
var cityMockHTML1 string

//go:embed city_service_test_mock_html2.html
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
			name:          "lat+lon in Georgetown, TX",
			location:      "30.631799878085577+-97.70332501413287", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      cityMockHTML1,
			mockError:     nil,
			expectedCity:  "Georgetown, Texas",
			expectedError: false,
		},
		{
			name:          "lat+lon in Badr, Egypt",
			location:      "30.6317+30.703325", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      cityMockHTML2,
			mockError:     nil,
			expectedCity:  "Badr, Egypt",
			expectedError: false,
		},
		{
			name:          "lat+lon in Mexico City, Mexico",
			location:      "19.4326077+-99.133208", // I manually copied the html using this location
			baseURL:       "https://example.com",
			mockHTML:      `["76F2CVM8+2PV"],["CVM8+2PV Mexico City, Mexico"],1]]],null,null,null,`,
			mockError:     nil,
			expectedCity:  "Mexico City, Mexico",
			expectedError: false,
		},
		{
			name:     "lat+lon with newline and spaces and two word city",
			location: "",
			baseURL:  "https://example.com",
			mockHTML: `XXFC+MF Copperas Cove,
										       Texas"]]],null,null`,
			mockError:     nil,
			expectedCity:  "Copperas Cove, Texas",
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
			name:          "invalid plus code", // The city always comes after a valid plus code
			location:      "Invalid Location",
			baseURL:       "https://example.com",
			mockHTML:      `"JRQ+  TVW NotACity, HI"`,
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "Invalid lat+lon",
			location:      "",
			baseURL:       "https://example.com",
			mockHTML:      `null,null,null,null,null,null,null,null,null,null,null,null,[[[120000000,0,0],null,null,13.10000038146973],null,0],null,null,null`,
			mockError:     nil,
			expectedCity:  "",
			expectedError: true,
		},
		{
			name:          "empty base URL",
			location:      "",
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
				t.Errorf("Expected %#v, got %#v", tt.expectedCity, city)
			}
		})
	}
}
