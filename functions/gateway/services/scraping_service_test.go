package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetHTMLFromURL(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		expectedHTML string
		expectedErr  error
	}{
		{
			name:         "Pre-escaped URL",
			value:        "https%3A%2F%example.com%2Fpath%3Fquery%3Dvalue",
			expectedHTML: "",
			expectedErr:  fmt.Errorf(URLEscapedErrorMsg),
		},
		{
			name:         "Correctly escaped URL value",
			value:        "https://example.com/path?query=value",
			expectedHTML: basicHTMLresp,
			expectedErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server with proper port rotation
			hostAndPort := test_helpers.GetNextPort()
			mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(basicHTMLresp))
			}))

			listener, err := test_helpers.BindToPort(t, hostAndPort)
			if err != nil {
				t.Fatalf("BindToPort failed: %v", err)
			}
			mockServer.Listener = listener
			mockServer.Start()
			defer mockServer.Close()

			// Use the mock server URL for testing
			baseURL := mockServer.URL

			html, err := GetHTMLFromURLWithBase(baseURL, tc.value, 10, true, "", 1, nil)

			if html != tc.expectedHTML {
				t.Fatalf("Expected %v, got %v", tc.expectedHTML, html)
			}

			// if we expect `nil` and get `nil`, return early, we want to avoid
			// calling `err.Error()` on a `nil` value below
			if tc.expectedErr == nil && err == nil {
				return
			}

			if err.Error() != tc.expectedErr.Error() {
				t.Fatalf("Expected %v, got %v", tc.expectedErr, err)
			}

		})
	}
}

func TestDeriveTimezoneFromCoordinates(t *testing.T) {
	testCases := []struct {
		name           string
		lat            float64
		lng            float64
		expectedResult string
		description    string
	}{
		{
			name:           "Empty input coordinates",
			lat:            0,
			lng:            0,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "Out of range latitude coordinates",
			lat:            91,
			lng:            0,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "Out of range longitude coordinates",
			lat:            0,
			lng:            181,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "United States coordinates",
			lat:            37.875580,
			lng:            -92.473411,
			expectedResult: "America/Chicago", // Missouri, USA
			description:    "Should return America/Chicago timezone for Missouri coordinates",
		},
		{
			name:           "United Kingdom coordinates",
			lat:            52.282165,
			lng:            -0.891387,
			expectedResult: "Europe/London", // England, UK
			description:    "Should return Europe/London timezone for England coordinates",
		},
		{
			name:           "Australia coordinates",
			lat:            -27.507454,
			lng:            144.479705,
			expectedResult: "Australia/Brisbane", // Queensland, Australia
			description:    "Should return Australia/Brisbane timezone for Queensland coordinates",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DeriveTimezoneFromCoordinates(tc.lat, tc.lng)

			if result != tc.expectedResult {
				t.Errorf("Expected timezone '%s', got '%s' for coordinates (%.6f, %.6f) - %s",
					tc.expectedResult, result, tc.lat, tc.lng, tc.description)
			}

			// Log the result for verification
			t.Logf("Coordinates (%.6f, %.6f) -> Timezone: '%s'", tc.lat, tc.lng, result)
		})
	}
}
