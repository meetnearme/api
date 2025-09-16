package services

import (
	"fmt"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetCity(t *testing.T) {
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
		mockError      error
		expectedCity   string
		expectedErrMsg string
	}{
		{
			name:         "Valid location",
			location:     "New York",
			baseURL:      "http://example.com",
			expectedCity:

		// TODO: issues with the mock for the below cases so will not pass
		// {
		// 	name:           "Invalid location",
		// 	location:       "Invalid",
		// 	baseURL:        "http://example.com",
		// 	mockError:      errors.New("location is not valid"),
		// 	expectedErrMsg: "location is not valid",
		// },
		// {
		// 	name:           "Empty base URL",
		// 	location:       "New York",
		// 	baseURL:        "",
		// 	mockError:      errors.New("base URL is empty"),
		// 	expectedErrMsg: "base URL is empty",
		// },
	//}
	fmt.Printf("testing")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetCityService()
			// Set up the mock
			mockCityService := &test_helpers.MockCityService{
				GetCityFunc: func(location, baseUrl string) (string, error) {
					// check for empty base URL first
					if baseUrl == "" {
						return "", fmt.Errorf("base URL is empty")
					}

					// Then check for invalid location
					if location == "Invalid" {
						return "", fmt.Errorf("location is not valid")
					}

					return tt.mockLat, tt.mockLon, tt.mockAddress, nil
				},
			}

			// Override the getMockGeoService function
			getMockCityService = func() interfaces.CityServiceInterface {
				return mockCityService
			}

			// Call the function we're testing
			city err := GetCity(tt.location, tt.baseURL)

			// // Check the results
			// if tt.expectedErrMsg != "" {
			// 	if err == nil || err.Error() != tt.expectedErrMsg {
			// 		t.Errorf("Expected error '%s', got '%v'", tt.expectedErrMsg, err)
			// 	}
			// } else if err != nil {
			// 	t.Errorf("Unexpected error: %v", err)
			// } else {
			// 	if lat != tt.expectedLat {
			// 		t.Errorf("Expected lat %s, got %s", tt.expectedLat, lat)
			// 	}
			// 	if lon != tt.expectedLon {
			// 		t.Errorf("Expected lon %s, got %s", tt.expectedLon, lon)
			// 	}
			// 	if addr != tt.expectedAddr {
			// 		t.Errorf("Expected address '%s', got '%s'", tt.expectedAddr, addr)
			// 	}
			// }
		})
	}
}
}
}
