package services

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetGeo(t *testing.T) {
	// Save the original environment variable
	origEnv := os.Getenv("GO_ENV")

	// Set the environment to "test"
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)

	// Ensure we reset the environment after the test
	defer os.Setenv("GO_ENV", origEnv)

	tests := []struct {
		name           string
		location       string
		baseURL        string
		mockLat        string
		mockLon        string
		mockAddress    string
		mockError      error
		expectedLat    string
		expectedLon    string
		expectedAddr   string
		expectedErrMsg string
	}{
		{
			name:         "Valid location",
			location:     "New York",
			baseURL:      "http://example.com",
			mockLat:      "40.7128",
			mockLon:      "-74.0060",
			mockAddress:  "New York, NY 10001, USA",
			expectedLat:  "40.7128",
			expectedLon:  "-74.0060",
			expectedAddr: "New York, NY 10001, USA",
		},
		{
			name:           "Invalid location",
			location:       "Invalid",
			baseURL:        "http://example.com",
			mockError:      errors.New("location is not valid"),
			expectedErrMsg: "location is not valid",
		},
		{
			name:           "Empty base URL",
			location:       "New York",
			baseURL:        "",
			mockError:      errors.New("base URL is empty"),
			expectedErrMsg: "base URL is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
            ResetGeoService()
			// Set up the mock
			mockGeoService := &test_helpers.MockGeoService{
				GetGeoFunc: func(location, baseUrl string) (string, string, string, error) {
                    // check for empty base URL first
                    if baseUrl == "" {
                        return "", "", "", fmt.Errorf("base URL is empty")
                    }

                    // Then check for invalid location
                    if location == "Invalid" {
                        return "", "", "", fmt.Errorf("location is not valid")
                    }

					return tt.mockLat, tt.mockLon, tt.mockAddress, nil
				},
			}

			// Override the getMockGeoService function
			getMockGeoService = func() interfaces.GeoServiceInterface {
				return mockGeoService
			}

			// Call the function we're testing
			lat, lon, addr, err := GetGeo(tt.location, tt.baseURL)

			// Check the results
			if tt.expectedErrMsg != "" {
				if err == nil || err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected error '%s', got '%v'", tt.expectedErrMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else {
				if lat != tt.expectedLat {
					t.Errorf("Expected lat %s, got %s", tt.expectedLat, lat)
				}
				if lon != tt.expectedLon {
					t.Errorf("Expected lon %s, got %s", tt.expectedLon, lon)
				}
				if addr != tt.expectedAddr {
					t.Errorf("Expected address '%s', got '%s'", tt.expectedAddr, addr)
				}
			}
		})
	}
}
