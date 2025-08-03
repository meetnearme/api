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
