package helpers

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func init() {
	os.Setenv("GO_ENV", GO_TEST_ENV)
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectedError string
	}{
		{"Valid date", "2099-05-01T12:00:00Z", "May 1, 2099 (Fri)", ""},
		{"Invalid date", "invalid-date", "", "not a valid unix timestamp"},
		{"Empty string", "", "", "not a valid unix timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := UtcOrUnixToUnix64(tt.input)
			result, err := FormatDate(date)
			if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected err to have: %v, got: %v", tt.expectedError, err)
			} else if result != tt.expected {
				t.Errorf("FormatDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectedError string
	}{
		{"Valid time", "2099-05-01T14:30:00Z", "2:30pm", ""},
		{"Invalid time", "invalid-time", "", "not a valid unix timestamp"},
		{"Empty string", "", "", "not a valid unix timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := UtcOrUnixToUnix64(tt.input)
			result, err := FormatTime(tm)
			if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected err to have: %v, got: %v", tt.expectedError, err)
			} else if result != tt.expected {
				t.Errorf("FormatTime(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateStringByBytes(t *testing.T) {
	tests := []struct {
		name     string
		input1   string
		input2   int
		expected string
	}{
		{"Truncate exceeds by one", "123456789012345678901", 20, "12345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := TruncateStringByBytes(tt.input1, tt.input2)
			if result != tt.expected {
				t.Errorf("TruncateStringByBytes(%q) = %q, want %q", tt.input1, result, tt.expected)
			}
		})
	}
}

func TestGetImgUrlFromHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid hash", "1234567890", "/assets/img/0.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetImgUrlFromHash(tt.input)
			if result != tt.expected {
				t.Errorf("GetImgUrlFromHash(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetCloudFlareKV(t *testing.T) {
	InitDefaultProtocol()
	// Save original environment variables
	originalAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	originalNamespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
	originalAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	originalCfApiBaseUrl := os.Getenv("CLOUDFLARE_API_BASE_URL")
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")

	// Set test environment variables
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-id")
	os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", "test-namespace-id")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-api-token")
	os.Setenv("CLOUDFLARE_API_BASE_URL", MOCK_CLOUDFLARE_URL)
	os.Setenv("ZITADEL_INSTANCE_HOST", MOCK_ZITADEL_HOST)
	// Defer resetting environment variables
	defer func() {
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", originalAccountID)
		os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", originalNamespaceID)
		os.Setenv("CLOUDFLARE_API_TOKEN", originalAPIToken)
		os.Setenv("CLOUDFLARE_API_BASE_URL", originalCfApiBaseUrl)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create a mock HTTP server for Cloudflare
	mockCloudflareServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the GET request to check if the key exists
		if r.Method == "GET" {
			if strings.Contains(r.URL.Path, "existing-subdomain") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"success": false}`))
			return
		}

		// Mock the PUT request to set the KV
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
			return
		}

		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set the mock Cloudflare server URL
	mockCloudflareServer.Listener.Close()
	var err error
	mockCloudflareServer.Listener, err = net.Listen("tcp", MOCK_CLOUDFLARE_URL[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Cloudflare server: %v", err)
	}
	mockCloudflareServer.Start()
	defer mockCloudflareServer.Close()

	// Create a mock HTTP server for Zitadel
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the GET request to return user metadata
		if r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"metadata": {"value": "dGVzdC12YWx1ZQ=="}}`)) // base64 for "test-value"
			return
		}

		// Mock the POST request to user metadata
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
			return
		}

		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set the mock Zitadel server URL
	mockZitadelServer.Listener.Close()
	mockZitadelServer.Listener, err = net.Listen("tcp", MOCK_ZITADEL_HOST)
	if err != nil {
		t.Fatalf("Failed to start mock Zitadel server: %v", err)
	}
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Test cases
	tests := []struct {
		name            string
		subdomainValue  string
		userID          string
		userMetadataKey string
		metadata        map[string]string
		expectedError   error
	}{
		{
			name:            "Successful KV set",
			subdomainValue:  "test-subdomain",
			userID:          "test-user-id",
			userMetadataKey: "test-metadata-key",
			metadata:        map[string]string{"key": "value"},
			expectedError:   nil,
		},
		{
			name:            "Key already exists",
			subdomainValue:  "existing-subdomain",
			userID:          "test-user-id",
			userMetadataKey: "test-metadata-key",
			metadata:        map[string]string{"key": "value"},
			expectedError:   fmt.Errorf(ERR_KV_KEY_EXISTS),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetCloudflareKV(tt.subdomainValue, tt.userID, tt.userMetadataKey, tt.metadata)
			if err != nil && tt.expectedError == nil {
				t.Errorf("SetCloudflareKV() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err == nil && tt.expectedError != nil {
				t.Errorf("SetCloudflareKV() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("SetCloudflareKV() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
		})
	}
}
