package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func init() {
	os.Setenv("GO_ENV", constants.GO_TEST_ENV)
}

func TestFormatDateL(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectedError string
	}{
		{"Valid date", "2099-05-01T12:00:00Z", "May 1, 2099 (Fri)", ""},
		{"Invalid date", "invalid-date", "", "not a valid time"},
		{"Empty string", "", "", "not a valid time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, _ := time.Parse(time.RFC3339, tt.input)
			result, err := FormatDateLocal(date)
			if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected err to have: %v, got: %v", tt.expectedError, err)
			} else if result != tt.expected {
				t.Errorf("FormatDateL(%q) = %q, want %q", tt.input, result, tt.expected)
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
		{"Invalid time", "invalid-time", "", "not a valid time"},
		{"Empty string", "", "", "not a valid time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, _ := time.Parse(time.RFC3339, tt.input)
			result, err := FormatTimeLocal(tm)
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
		input    types.Event
		expected string
	}{
		{"Valid hash", types.Event{Id: "1234567890"}, "/assets/img/cat_none_16.jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetImgUrlFromHash(tt.input)
			if result != tt.expected {
				t.Errorf("GetImgUrlFromHash(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetCloudflareMnmOptions(t *testing.T) {
	InitDefaultProtocol()
	// Save original environment variables
	originalAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	originalNamespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
	originalAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	originalCfApiBaseUrl := os.Getenv("CLOUDFLARE_API_CLIENT_BASE_URL")
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalZitadelBotAdminToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")

	// Get initial endpoints
	port := test_helpers.GetNextPort()
	cfEndpoint := fmt.Sprintf("http://%s", port)
	zitadelEndpoint := test_helpers.GetNextPort()

	// Set environment variables with http:// prefix
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-id")
	os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", "test-namespace-id")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-api-token")
	os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", cfEndpoint)
	os.Setenv("ZITADEL_INSTANCE_HOST", zitadelEndpoint)
	os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-bot-admin-token")

	// Defer resetting environment variables
	defer func() {
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", originalAccountID)
		os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", originalNamespaceID)
		os.Setenv("CLOUDFLARE_API_TOKEN", originalAPIToken)
		os.Setenv("CLOUDFLARE_API_BASE_URL", originalCfApiBaseUrl)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalZitadelBotAdminToken)
	}()

	// Create mock servers
	mockCloudflareServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the GET request to check if the key exists
		if r.Method == "GET" {
			// For test-nonexistent-subdomain, return 404 (key doesn't exist yet)
			if strings.Contains(r.URL.Path, "test-nonexistent-subdomain") {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"success": false, "errors": [{"code": 10009, "message": "Not Found"}]}`))
				return
			}
			// For existing-subdomain, return 200 (key exists)
			if strings.Contains(r.URL.Path, "existing-subdomain") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
				return
			}
			// Default to 404 for any other subdomain
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"success": false, "errors": [{"code": 10009, "message": "Not Found"}]}`))
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

	// Bind to ports
	cfListener, err := test_helpers.BindToPort(t, cfEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind Cloudflare server: %v", err)
	}
	mockCloudflareServer.Listener = cfListener
	mockCloudflareServer.Start()
	defer mockCloudflareServer.Close()

	boundCfAddress := fmt.Sprintf("http://%s", mockCloudflareServer.Listener.Addr().String())
	os.Setenv("CLOUDFLARE_API_BASE_URL", boundCfAddress)

	zitadelListener, err := test_helpers.BindToPort(t, zitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind Zitadel server: %v", err)
	}
	mockZitadelServer.Listener = zitadelListener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	boundZtAddress := mockZitadelServer.Listener.Addr().String()
	os.Setenv("ZITADEL_INSTANCE_HOST", boundZtAddress)

	// Test cases
	tests := []struct {
		name            string
		subdomainValue  string
		userID          string
		userMetadataKey string
		metadata        map[string]string
		cfMetadataValue string
		expectedError   error
	}{
		{
			name:            "Successful KV set",
			subdomainValue:  "test-nonexistent-subdomain",
			userID:          "test-user-id",
			metadata:        map[string]string{"key": "value"},
			cfMetadataValue: "test-cf-metadata-value",
			expectedError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetCloudflareMnmOptions(tt.subdomainValue, tt.userID, tt.metadata, tt.cfMetadataValue)
			if err != nil && tt.expectedError == nil {
				t.Errorf("SetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err == nil && tt.expectedError != nil {
				t.Errorf("SetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("SetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
		})
	}
}

func TestSearchUsersByIDs(t *testing.T) {
	// Initialize and setup environment
	InitDefaultProtocol()
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	testZitadelEndpoint := test_helpers.GetNextPort()
	os.Setenv("ZITADEL_INSTANCE_HOST", testZitadelEndpoint)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create mock Zitadel server
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/v2/users") {
			w.Header().Set("Content-Type", "application/json")

			// Parse request body to get userIds
			var requestBody struct {
				Queries []struct {
					InUserIdsQuery struct {
						UserIds []string `json:"userIds"`
					} `json:"inUserIdsQuery"`
				} `json:"queries"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Get userIds from request
			var userIds []string
			if len(requestBody.Queries) > 0 {
				userIds = requestBody.Queries[0].InUserIdsQuery.UserIds
			}

			// Prepare response based on input userIds
			var response ZitadelUserSearchResponse
			response.Details.Timestamp = "2099-01-01T00:00:00Z"

			switch {
			case len(userIds) == 0:
				http.Error(w, "no user IDs provided", http.StatusBadRequest)
				return
			case contains(userIds, "error_id"):
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			case contains(userIds, "nonexistent"):
				response.Details.TotalResult = "0"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{}
			default:
				// Return mock users for valid IDs
				response.Details.TotalResult = fmt.Sprintf("%d", len(userIds))
				for _, id := range userIds {
					response.Result = append(response.Result, struct {
						UserID             string `json:"userId"`
						Username           string `json:"username"`
						PreferredLoginName string `json:"preferredLoginName"`
						State              string `json:"state"`
						Human              struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						} `json:"human"`
					}{
						UserID: id,
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Test User " + id,
							},
						},
					})
				}
			}

			responseJSON, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Write(responseJSON)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set up mock server
	mockZitadelServer.Listener.Close()
	var err error

	listener, err := test_helpers.BindToPort(t, testZitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockZitadelServer.Listener = listener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	tests := []struct {
		name          string
		userIDs       []string
		expectedUsers []types.UserSearchResultDangerous
		expectError   bool
	}{
		{
			name:    "successful search with multiple users",
			userIDs: []string{"123", "456"},
			expectedUsers: []types.UserSearchResultDangerous{
				{UserID: "123", DisplayName: "Test User 123"},
				{UserID: "456", DisplayName: "Test User 456"},
			},
		},
		{
			name:          "empty result",
			userIDs:       []string{"nonexistent"},
			expectedUsers: []types.UserSearchResultDangerous{},
		},
		{
			name:        "server error",
			userIDs:     []string{"error_id"},
			expectError: true,
		},
		{
			name:          "empty user IDs",
			userIDs:       []string{},
			expectedUsers: []types.UserSearchResultDangerous{},
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, err := SearchUsersByIDs(tt.userIDs, false)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(users) != len(tt.expectedUsers) {
				t.Errorf("expected %d users, got %d", len(tt.expectedUsers), len(users))
				return
			}

			for i, user := range users {
				if user.UserID != tt.expectedUsers[i].UserID {
					t.Errorf("expected UserID %s, got %s", tt.expectedUsers[i].UserID, user.UserID)
				}
				if user.DisplayName != tt.expectedUsers[i].DisplayName {
					t.Errorf("expected DisplayName %s, got %s", tt.expectedUsers[i].DisplayName, user.DisplayName)
				}
			}
		})
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func TestSearchUserByEmailOrName(t *testing.T) {
	// Initialize and setup environment
	InitDefaultProtocol()
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	testZitadelEndpoint := test_helpers.GetNextPort()
	os.Setenv("ZITADEL_INSTANCE_HOST", testZitadelEndpoint)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create mock Zitadel server
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/v2/users") {
			w.Header().Set("Content-Type", "application/json")

			// Parse request body to get search query
			var requestBody struct {
				Queries []struct {
					OrQuery struct {
						Queries []struct {
							EmailQuery struct {
								EmailAddress string `json:"emailAddress"`
							} `json:"emailQuery"`
							UserNameQuery struct {
								UserName string `json:"userName"`
							} `json:"userNameQuery"`
						} `json:"queries"`
					} `json:"orQuery"`
				} `json:"queries"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Get search query from request
			var searchQuery string
			if len(requestBody.Queries) > 1 && len(requestBody.Queries[1].OrQuery.Queries) > 0 {
				searchQuery = requestBody.Queries[1].OrQuery.Queries[0].EmailQuery.EmailAddress
			}

			// Prepare response based on search query
			var response ZitadelUserSearchResponse
			response.Details.Timestamp = "2099-01-01T00:00:00Z"

			switch searchQuery {
			case "error":
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			case "nonexistent":
				response.Details.TotalResult = "0"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{}
			default:
				// Return mock users for valid search
				response.Details.TotalResult = "2"
				response.Result = []struct {
					UserID             string `json:"userId"`
					Username           string `json:"username"`
					PreferredLoginName string `json:"preferredLoginName"`
					State              string `json:"state"`
					Human              struct {
						Profile struct {
							DisplayName string `json:"displayName"`
						} `json:"profile"`
						Email map[string]interface{} `json:"email"`
					} `json:"human"`
				}{
					{
						UserID: "123",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "John Doe",
							},
						},
					},
					{
						UserID: "456",
						Human: struct {
							Profile struct {
								DisplayName string `json:"displayName"`
							} `json:"profile"`
							Email map[string]interface{} `json:"email"`
						}{
							Profile: struct {
								DisplayName string `json:"displayName"`
							}{
								DisplayName: "Jane Doe",
							},
						},
					},
				}
			}

			responseJSON, err := json.Marshal(response)
			if err != nil {
				http.Error(w, "failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Write(responseJSON)
			return
		}
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set up mock server
	mockZitadelServer.Listener.Close()
	var err error
	listener, err := test_helpers.BindToPort(t, testZitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockZitadelServer.Listener = listener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	tests := []struct {
		name          string
		query         string
		expectedUsers []types.UserSearchResultDangerous
		expectError   bool
	}{
		{
			name:  "successful search with results",
			query: "doe",
			expectedUsers: []types.UserSearchResultDangerous{
				{UserID: "123", DisplayName: "John Doe"},
				{UserID: "456", DisplayName: "Jane Doe"},
			},
		},
		{
			name:          "no results found",
			query:         "nonexistent",
			expectedUsers: []types.UserSearchResultDangerous{},
		},
		{
			name:        "server error",
			query:       "error",
			expectError: true,
		},
		// NOTE: this is faithful to the Zitadel API, which returns all users if the query is empty
		{
			name:  "empty query",
			query: "",
			expectedUsers: []types.UserSearchResultDangerous{
				{UserID: "123", DisplayName: "John Doe"},
				{UserID: "456", DisplayName: "Jane Doe"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, err := SearchUserByEmailOrName(tt.query)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(users) != len(tt.expectedUsers) {
				t.Errorf("expected %d users, got %d", len(tt.expectedUsers), len(users))
				return
			}

			for i, user := range users {
				if user.UserID != tt.expectedUsers[i].UserID {
					t.Errorf("expected UserID %s, got %s", tt.expectedUsers[i].UserID, user.UserID)
				}
				if user.DisplayName != tt.expectedUsers[i].DisplayName {
					t.Errorf("expected DisplayName %s, got %s", tt.expectedUsers[i].DisplayName, user.DisplayName)
				}
			}
		})
	}
}

func TestUpdateUserMetadataKey(t *testing.T) {
	// Initialize and setup environment
	InitDefaultProtocol()
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	testZitadelEndpoint := test_helpers.GetNextPort()
	os.Setenv("ZITADEL_INSTANCE_HOST", testZitadelEndpoint)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
	}()

	// Create mock Zitadel server
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/management/v1/users/") && strings.Contains(r.URL.Path, "/metadata/") {
			// Parse request body
			var requestBody struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

			// Extract userID and key from URL path
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) < 7 {
				http.Error(w, "invalid URL path", http.StatusBadRequest)
				return
			}

			userID := pathParts[4]
			key := pathParts[6]

			// Add validation for empty values
			if userID == "" {
				http.Error(w, `{"error": "user ID cannot be empty"}`, http.StatusBadRequest)
				return
			}

			if key == "" {
				http.Error(w, `{"error": "metadata key cannot be empty"}`, http.StatusBadRequest)
				return
			}

			switch {
			case userID == "error_user":
				http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
				return
			case key == "error_key":
				http.Error(w, `{"error": "invalid metadata key"}`, http.StatusBadRequest)
				return
			default:
				// Success case
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "success"}`))
				return
			}
		}
		http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Set up mock server
	mockZitadelServer.Listener.Close()
	var err error
	listener, err := test_helpers.BindToPort(t, testZitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockZitadelServer.Listener = listener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	tests := []struct {
		name        string
		userID      string
		key         string
		value       string
		expectError bool
	}{
		{
			name:        "successful update",
			userID:      "123",
			key:         "test_key",
			value:       "test_value",
			expectError: false,
		},
		{
			name:        "user not found",
			userID:      "error_user",
			key:         "test_key",
			value:       "test_value",
			expectError: true,
		},
		{
			name:        "empty user ID",
			userID:      "",
			key:         "test_key",
			value:       "test_value",
			expectError: true,
		},
		{
			name:        "empty key",
			userID:      "123",
			key:         "",
			value:       "test_value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateUserMetadataKey(tt.userID, tt.key, tt.value)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
		})
	}
}

func TestUtcToUnix64(t *testing.T) {
	// Load test timezone
	chicagoTZ, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("Failed to load Chicago timezone: %v", err)
	}

	// Load UTC timezone for comparison
	utcTZ := time.UTC

	tests := []struct {
		name        string
		input       interface{}
		timezone    *time.Location
		expected    int64
		expectError bool
		errorMsg    string
	}{
		// Tests for default UtcToUnix64() behavior (trimZ=true - old behavior)
		{
			name:        "UTC format with Z suffix (trimZ behavior)",
			input:       "2026-09-12T17:00:00Z",
			timezone:    chicagoTZ,
			expected:    1789250400, // Unix timestamp for 2026-09-12T17:00:00 in Chicago time (trimmed Z, parsed as local)
			expectError: false,
		},
		{
			name:        "UTC format with Z suffix in UTC timezone (trimZ behavior)",
			input:       "2026-09-12T17:00:00Z",
			timezone:    utcTZ,
			expected:    1789232400, // Unix timestamp for 2026-09-12T17:00:00 in UTC time (trimmed Z, parsed as local)
			expectError: false,
		},
		{
			name:        "leap year date (trimZ behavior)",
			input:       "2024-02-29T12:00:00Z",
			timezone:    chicagoTZ,
			expected:    1709229600, // Unix timestamp for 2024-02-29T12:00:00 in Chicago time (trimmed Z, parsed as local)
			expectError: false,
		},
		{
			name:        "end of year (trimZ behavior)",
			input:       "2023-12-31T23:59:59Z",
			timezone:    chicagoTZ,
			expected:    1704088799, // Unix timestamp for 2023-12-31T23:59:59 in Chicago time (trimmed Z, parsed as local)
			expectError: false,
		},
		// Error cases
		{
			name:        "invalid format (missing T separator)",
			input:       "2026-09-12 17:00:00", // Missing T separator
			timezone:    chicagoTZ,
			expected:    0,
			expectError: true,
			errorMsg:    "invalid date format",
		},
		{
			name:        "empty string",
			input:       "",
			timezone:    chicagoTZ,
			expected:    0,
			expectError: true,
			errorMsg:    "invalid date format",
		},
		{
			name:        "unsupported type (int)",
			input:       1234567890,
			timezone:    chicagoTZ,
			expected:    0,
			expectError: true,
			errorMsg:    "unsupported time format",
		},
		{
			name:        "unsupported type (nil)",
			input:       nil,
			timezone:    chicagoTZ,
			expected:    0,
			expectError: true,
			errorMsg:    "unsupported time format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UtcToUnix64(tt.input, tt.timezone)

			if tt.expectError {
				if err == nil {
					t.Errorf("UtcToUnix64() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("UtcToUnix64() error = %v, expected to contain %v", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("UtcToUnix64() unexpected error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("UtcToUnix64() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUtcToUnix64WithTrimZ(t *testing.T) {
	// Load test timezone
	chicagoTZ, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("Failed to load Chicago timezone: %v", err)
	}

	// Load UTC timezone for comparison
	utcTZ := time.UTC

	tests := []struct {
		name        string
		input       interface{}
		timezone    *time.Location
		trimZ       bool
		expected    int64
		expectError bool
		errorMsg    string
	}{
		// Tests for trimZ=true (old behavior)
		{
			name:        "trimZ=true: UTC format with Z suffix",
			input:       "2026-09-12T17:00:00Z",
			timezone:    chicagoTZ,
			trimZ:       true,
			expected:    1789250400, // Unix timestamp for 2026-09-12T17:00:00 in Chicago time (trimmed Z, parsed as local)
			expectError: false,
		},
		{
			name:        "trimZ=true: UTC format with Z suffix in UTC timezone",
			input:       "2026-09-12T17:00:00Z",
			timezone:    utcTZ,
			trimZ:       true,
			expected:    1789232400, // Unix timestamp for 2026-09-12T17:00:00 in UTC time (trimmed Z, parsed as local)
			expectError: false,
		},
		// Tests for trimZ=false (new behavior)
		{
			name:        "trimZ=false: UTC format with Z suffix",
			input:       "2026-09-12T17:00:00Z",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1789232400, // Unix timestamp for 2026-09-12T12:00:00-05:00 (Chicago time, parsed as UTC first)
			expectError: false,
		},
		{
			name:        "trimZ=false: UTC format with Z suffix in UTC timezone",
			input:       "2026-09-12T17:00:00Z",
			timezone:    utcTZ,
			trimZ:       false,
			expected:    1789232400, // Unix timestamp for 2026-09-12T17:00:00Z (UTC time, parsed as UTC first)
			expectError: false,
		},
		{
			name:        "trimZ=false: timezone offset format with -05:00",
			input:       "2026-09-12T12:00:00-05:00",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1789232400, // Unix timestamp for 2026-09-12T12:00:00-05:00 (Chicago time)
			expectError: false,
		},
		{
			name:        "trimZ=false: timezone offset format with +09:00",
			input:       "2026-09-12T02:00:00+09:00",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1789146000, // Same moment in time, different timezone
			expectError: false,
		},
		{
			name:        "trimZ=false: timezone offset format with +00:00 (UTC)",
			input:       "2026-09-12T17:00:00+00:00",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1789232400, // Same as Z format
			expectError: false,
		},
		// Edge cases for trimZ=false
		{
			name:        "trimZ=false: leap year date",
			input:       "2024-02-29T12:00:00Z",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1709208000, // Unix timestamp for 2024-02-29T06:00:00-06:00 (Chicago time)
			expectError: false,
		},
		{
			name:        "trimZ=false: end of year",
			input:       "2023-12-31T23:59:59Z",
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1704067199, // Unix timestamp for 2023-12-31T17:59:59-06:00 (Chicago time)
			expectError: false,
		},
		// Error cases for trimZ=false
		{
			name:        "trimZ=false: invalid RFC3339 format",
			input:       "2026-09-12 17:00:00", // Missing T separator
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    0,
			expectError: true,
			errorMsg:    "invalid date format",
		},
		{
			name:        "trimZ=false: malformed timezone offset",
			input:       "2026-09-12T17:00:00-25:00", // Invalid timezone offset
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    0,
			expectError: true,
			errorMsg:    "invalid date format",
		},
		// DST transition tests for trimZ=false
		{
			name:        "trimZ=false: DST start (spring forward)",
			input:       "2024-03-10T07:00:00Z", // 2 AM local time (spring forward)
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1710054000, // Unix timestamp for 2024-03-10T01:00:00-06:00 (Chicago time)
			expectError: false,
		},
		{
			name:        "trimZ=false: DST end (fall back)",
			input:       "2024-11-03T06:00:00Z", // 1 AM local time (fall back)
			timezone:    chicagoTZ,
			trimZ:       false,
			expected:    1730613600, // Unix timestamp for 2024-11-03T01:00:00-05:00 (Chicago time)
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UtcToUnix64WithTrimZ(tt.input, tt.timezone, tt.trimZ)

			if tt.expectError {
				if err == nil {
					t.Errorf("UtcToUnix64WithTrimZ() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("UtcToUnix64WithTrimZ() error = %v, expected to contain %v", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("UtcToUnix64WithTrimZ() unexpected error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("UtcToUnix64WithTrimZ() = %v, want %v", result, tt.expected)
			}

			// Additional verification for trimZ=false: convert back to time and check it's correct
			if !tt.trimZ {
				convertedTime := time.Unix(result, 0).In(tt.timezone)
				expectedTime, _ := time.Parse(time.RFC3339, tt.input.(string))
				expectedTimeInTZ := expectedTime.In(tt.timezone)

				if !convertedTime.Equal(expectedTimeInTZ) {
					t.Errorf("UtcToUnix64WithTrimZ() conversion verification failed: got %v, want %v",
						convertedTime.Format(time.RFC3339), expectedTimeInTZ.Format(time.RFC3339))
				}
			}
		})
	}
}

func TestGetBase64ValueFromMap(t *testing.T) {
	tests := []struct {
		name       string
		claimsMeta map[string]interface{}
		key        string
		want       string
	}{
		{
			name: "valid base64 string",
			claimsMeta: map[string]interface{}{
				"test": "SGVsbG8gV29ybGQ=", // "Hello World" in base64
			},
			key:  "test",
			want: "Hello World",
		},
		{
			name: "invalid base64 string",
			claimsMeta: map[string]interface{}{
				"test": "invalid-base64!@#",
			},
			key:  "test",
			want: "",
		},
		{
			name:       "missing key",
			claimsMeta: map[string]interface{}{},
			key:        "nonexistent",
			want:       "",
		},
		{
			name: "non-string value",
			claimsMeta: map[string]interface{}{
				"test": 123,
			},
			key:  "test",
			want: "",
		},
		{
			name: "base64 string without padding",
			claimsMeta: map[string]interface{}{
				"test": "SGVsbG8gV29ybGQ", // "Hello World" in base64 without padding
			},
			key:  "test",
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBase64ValueFromMap(tt.claimsMeta, tt.key)
			if got != tt.want {
				t.Errorf("GetBase64ValueFromMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCloudflareMnmOptions(t *testing.T) {
	// Save original environment variables
	var (
		originalAccountID    = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
		originalNamespaceID  = os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
		originalCfApiBaseUrl = os.Getenv("CLOUDFLARE_API_BASE_URL")
	)
	port := test_helpers.GetNextPort()
	cfEndpoint := fmt.Sprintf("http://%s", port)
	// Set test environment variables
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-id")
	os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", "test-namespace-id")
	os.Setenv("CLOUDFLARE_API_BASE_URL", cfEndpoint)
	os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", cfEndpoint)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", originalAccountID)
		os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", originalNamespaceID)
		os.Setenv("CLOUDFLARE_API_BASE_URL", originalCfApiBaseUrl)
		os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", originalCfApiBaseUrl)
	}()

	// Create mock Cloudflare server
	mockCloudflareServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and method
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if the request is for the correct endpoint
		expectedPath := "/accounts/test-account-id/storage/kv/namespaces/test-namespace-id/values/"
		if !strings.Contains(r.URL.Path, expectedPath) {
			http.Error(w, "Invalid endpoint", http.StatusNotFound)
			return
		}

		// Extract the subdomain value from the path
		subdomainValue := strings.TrimPrefix(r.URL.Path, expectedPath)

		// Mock successful response for existing subdomain
		if subdomainValue == "test-nonexistent-subdomain" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "result": "test-value"}`))
			return
		}

		// Mock 404 for non-existent subdomain
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success": false, "errors": [{"code": 10009, "message": "Not Found"}]}`))
	}))

	// Set up the mock server
	mockCloudflareServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, cfEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Cloudflare server: %v", err)
	}
	mockCloudflareServer.Listener = listener
	mockCloudflareServer.Start()
	defer mockCloudflareServer.Close()

	tests := []struct {
		name           string
		subdomainValue string
		expectedValue  string
		expectedError  error
	}{
		{
			name:           "Successful KV get",
			subdomainValue: "test-nonexistent-subdomain",
			expectedValue:  `{"success": true, "result": "test-value"}`,
			expectedError:  nil,
		},
		{
			name:           "Non-existent subdomain",
			subdomainValue: "non-existent",
			expectedValue:  "",
			expectedError:  fmt.Errorf("error getting cloudflare mnm options: GET \"%s/accounts/test-account-id/storage/kv/namespaces/test-namespace-id/values/non-existent\": 404 Not Found {\"success\": false, \"errors\": [{\"code\": 10009, \"message\": \"Not Found\"}]}", cfEndpoint),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := GetCloudflareMnmOptions(tt.subdomainValue)

			if err != nil && tt.expectedError == nil {
				t.Errorf("GetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err == nil && tt.expectedError != nil {
				t.Errorf("GetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if err != nil && tt.expectedError != nil && err.Error() != tt.expectedError.Error() {
				t.Errorf("GetCloudflareMnmOptions() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if value != tt.expectedValue {
				t.Errorf("GetCloudflareMnmOptions() value = %v, expectedValue %v", value, tt.expectedValue)
			}
		})
	}
}

// func TestNormalizeURL(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		input    string
// 		expected string
// 		wantErr  bool
// 	}{
// 		{"Basic HTTP", "http://example.com", "https://example.com", false},
// 		{"Basic HTTPS", "https://example.com", "https://example.com", false},
// 		{"Uppercase Scheme and Host", "HTTP://EXAMPLE.COM", "https://example.com", false},
// 		{"URL with fragment", "http://example.com#section", "https://example.com", false},
// 		{"URL with user info", "http://user:pass@example.com", "https://example.com", false},
// 		{"URL with default port 80", "http://example.com:80", "https://example.com", false},
// 		{"URL with default port 443", "https://example.com:443", "https://example.com", false},
// 		{"URL with non-default port", "https://example.com:8443", "https://example.com:8443", false},
// 		{"HTTPS with sorted query", "https://example.com?b=2&a=1", "https://example.com?a=1&b=2", false},
// 		{"Query with multiple values", "https://example.com?b=2&b=1", "https://example.com?b=1&b=2", false},
// 		{"Query with encoded characters", "https://example.com?q=a+b", "https://example.com?q=a%2Bb", false},
// 		{"Missing scheme (defaults to HTTPS)", "example.com", "https://example.com", false},
// 		{"Unsupported scheme", "ftp://example.com", "", true},
// 		{"Malformed URL", "http://%41", "", true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := NormalizeURL(tt.input)
// 			if (err != nil) != tt.wantErr {
// 				t.Fatalf("NormalizeURL(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
// 			}
// 			if got != tt.expected && !tt.wantErr {
// 				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
// 			}
// 		})
// 	}
// }

// func TestDomainFromURL(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		input    string
// 		expected string
// 		wantErr  bool
// 	}{
// 		{"Valid URL", "https://example.com/path", "example.com", false},
// 		{"URL with subdomain", "https://sub.example.com/path", "sub.example.com", false},
// 		{"URL with port", "https://example.com:8080/path", "example.com:8080", false},
// 		{"URL with query", "https://example.com/path?query=1", "example.com", false},
// 		{"Invalid URL format", "not-a-url", "", true},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got, err := ExtractBaseDomain(tt.input)
// 			if (err != nil) != tt.wantErr {
// 				t.Fatalf("ExtractBaseDomain(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
// 			}
// 			if got != tt.expected && !tt.wantErr {
// 				t.Errorf("ExtractBaseDomain(%q) = %q, want %q", tt.input, got, tt.expected)
// 			}
// 		})
// 	}
// }

// =============================================================================
// Tests for new Zitadel Authorization API functions
// =============================================================================

func TestGetUserRoles(t *testing.T) {
	// Save original environment variables
	originalHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalToken)
		http.DefaultTransport = originalTransport
	}()

	tests := []struct {
		name           string
		userID         string
		mockResponse   string
		mockStatusCode int
		expectedRoles  []string
		expectedError  string
	}{
		{
			name:   "Success - User with multiple roles",
			userID: "user123",
			mockResponse: `{
				"authorizations": [
					{
						"roles": ["subGrowth", "eventAdmin"]
					}
				]
			}`,
			mockStatusCode: http.StatusOK,
			expectedRoles:  []string{"subGrowth", "eventAdmin"},
			expectedError:  "",
		},
		{
			name:   "Success - User with single role",
			userID: "user456",
			mockResponse: `{
				"authorizations": [
					{
						"roles": ["subSeed"]
					}
				]
			}`,
			mockStatusCode: http.StatusOK,
			expectedRoles:  []string{"subSeed"},
			expectedError:  "",
		},
		{
			name:   "Success - User with no authorizations",
			userID: "user789",
			mockResponse: `{
				"authorizations": []
			}`,
			mockStatusCode: http.StatusOK,
			expectedRoles:  []string{},
			expectedError:  "",
		},
		{
			name:   "Success - User with multiple authorizations and roles",
			userID: "user999",
			mockResponse: `{
				"authorizations": [
					{
						"roles": ["subGrowth"]
					},
					{
						"roles": ["eventAdmin", "orgAdmin"]
					}
				]
			}`,
			mockStatusCode: http.StatusOK,
			expectedRoles:  []string{"subGrowth", "eventAdmin", "orgAdmin"},
			expectedError:  "",
		},
		{
			name:           "Error - HTTP 401 Unauthorized",
			userID:         "user123",
			mockResponse:   `{"error": "unauthorized"}`,
			mockStatusCode: http.StatusUnauthorized,
			expectedRoles:  nil,
			expectedError:  "failed to get authorizations: status 401",
		},
		{
			name:           "Error - HTTP 500 Internal Server Error",
			userID:         "user123",
			mockResponse:   `{"error": "internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectedRoles:  nil,
			expectedError:  "failed to get authorizations: status 500",
		},
		{
			name:           "Error - Invalid JSON response",
			userID:         "user123",
			mockResponse:   `{invalid json}`,
			mockStatusCode: http.StatusOK,
			expectedRoles:  nil,
			expectedError:  "failed to unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				// Verify path
				expectedPath := "/zitadel.authorization.v2beta.AuthorizationService/ListAuthorizations"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("Expected Authorization header with Bearer token")
				}

				// Verify request body contains user ID
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), tt.userID) {
					t.Errorf("Request body should contain user ID %s", tt.userID)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer mockServer.Close()

			// Set environment variables
			zitadelURL := strings.TrimPrefix(mockServer.URL, "http://")
			zitadelURL = strings.TrimPrefix(zitadelURL, "https://")
			os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)
			os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-token")

			// Call the function
			roles, err := GetUserRoles(tt.userID)

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(roles) != len(tt.expectedRoles) {
					t.Errorf("Expected %d roles, got %d", len(tt.expectedRoles), len(roles))
				}
				for i, role := range roles {
					if i >= len(tt.expectedRoles) || role != tt.expectedRoles[i] {
						t.Errorf("Expected role[%d] = %s, got %s", i, tt.expectedRoles[i], role)
					}
				}
			}
		})
	}
}

func TestGetUserAuthorizationID(t *testing.T) {
	// Save original environment variables
	originalHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalToken)
		http.DefaultTransport = originalTransport
	}()

	tests := []struct {
		name           string
		userID         string
		mockResponse   string
		mockStatusCode int
		expectedID     string
		expectedError  string
	}{
		{
			name:   "Success - User with authorization",
			userID: "user123",
			mockResponse: `{
				"authorizations": [
					{
						"id": "auth_abc123"
					}
				]
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "auth_abc123",
			expectedError:  "",
		},
		{
			name:   "Success - User with no authorization returns empty string",
			userID: "user456",
			mockResponse: `{
				"authorizations": []
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "",
			expectedError:  "",
		},
		{
			name:   "Success - Multiple authorizations returns first",
			userID: "user789",
			mockResponse: `{
				"authorizations": [
					{
						"id": "auth_first"
					},
					{
						"id": "auth_second"
					}
				]
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "auth_first",
			expectedError:  "",
		},
		{
			name:           "Error - HTTP 404 Not Found",
			userID:         "user999",
			mockResponse:   `{"error": "not found"}`,
			mockStatusCode: http.StatusNotFound,
			expectedID:     "",
			expectedError:  "failed to get authorizations: status 404",
		},
		{
			name:           "Error - Invalid JSON response",
			userID:         "user123",
			mockResponse:   `{invalid}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "",
			expectedError:  "failed to unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				expectedPath := "/zitadel.authorization.v2beta.AuthorizationService/ListAuthorizations"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer mockServer.Close()

			// Set environment variables
			zitadelURL := strings.TrimPrefix(mockServer.URL, "http://")
			zitadelURL = strings.TrimPrefix(zitadelURL, "https://")
			os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)
			os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-token")

			authID, err := GetUserAuthorizationID(tt.userID)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if authID != tt.expectedID {
					t.Errorf("Expected ID '%s', got '%s'", tt.expectedID, authID)
				}
			}
		})
	}
}

func TestCreateUserAuthorization(t *testing.T) {
	originalHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalToken)
		http.DefaultTransport = originalTransport
	}()

	tests := []struct {
		name           string
		userID         string
		roleKeys       []string
		mockResponse   string
		mockStatusCode int
		expectedID     string
		expectedError  string
	}{
		{
			name:     "Success - Create authorization with single role",
			userID:   "user123",
			roleKeys: []string{"subGrowth"},
			mockResponse: `{
				"id": "auth_new123"
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "auth_new123",
			expectedError:  "",
		},
		{
			name:     "Success - Create authorization with multiple roles",
			userID:   "user456",
			roleKeys: []string{"subGrowth", "eventAdmin", "orgAdmin"},
			mockResponse: `{
				"id": "auth_new456"
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "auth_new456",
			expectedError:  "",
		},
		{
			name:     "Success - Create authorization with empty roles",
			userID:   "user789",
			roleKeys: []string{},
			mockResponse: `{
				"id": "auth_new789"
			}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "auth_new789",
			expectedError:  "",
		},
		{
			name:           "Error - HTTP 400 Bad Request",
			userID:         "user999",
			roleKeys:       []string{"invalidRole"},
			mockResponse:   `{"error": "invalid role"}`,
			mockStatusCode: http.StatusBadRequest,
			expectedID:     "",
			expectedError:  "failed to create authorization: status 400",
		},
		{
			name:           "Error - HTTP 409 Conflict (already exists)",
			userID:         "user888",
			roleKeys:       []string{"subGrowth"},
			mockResponse:   `{"error": "authorization already exists"}`,
			mockStatusCode: http.StatusConflict,
			expectedID:     "",
			expectedError:  "failed to create authorization: status 409",
		},
		{
			name:           "Error - Invalid JSON response",
			userID:         "user777",
			roleKeys:       []string{"subGrowth"},
			mockResponse:   `{invalid}`,
			mockStatusCode: http.StatusOK,
			expectedID:     "",
			expectedError:  "failed to unmarshal response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}

				expectedPath := "/zitadel.authorization.v2beta.AuthorizationService/CreateAuthorization"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify request body
				body, _ := io.ReadAll(r.Body)
				bodyStr := string(body)
				if !strings.Contains(bodyStr, tt.userID) {
					t.Errorf("Request body should contain user ID %s", tt.userID)
				}
				for _, role := range tt.roleKeys {
					if !strings.Contains(bodyStr, role) {
						t.Errorf("Request body should contain role %s", role)
					}
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer mockServer.Close()

			// Set environment variables
			zitadelURL := strings.TrimPrefix(mockServer.URL, "http://")
			zitadelURL = strings.TrimPrefix(zitadelURL, "https://")
			os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)
			os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-token")

			authID, err := CreateUserAuthorization(tt.userID, tt.roleKeys)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if authID != tt.expectedID {
					t.Errorf("Expected ID '%s', got '%s'", tt.expectedID, authID)
				}
			}
		})
	}
}

func TestSetUserRoles(t *testing.T) {
	originalHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalToken)
		http.DefaultTransport = originalTransport
	}()

	tests := []struct {
		name                 string
		userID               string
		roleKeys             []string
		existingAuthID       string
		listAuthResponse     string
		listAuthStatusCode   int
		createAuthResponse   string
		createAuthStatusCode int
		updateAuthResponse   string
		updateAuthStatusCode int
		expectedError        string
		expectCreateCall     bool
		expectUpdateCall     bool
	}{
		{
			name:     "Success - Update existing authorization",
			userID:   "user123",
			roleKeys: []string{"subGrowth", "eventAdmin"},
			listAuthResponse: `{
				"authorizations": [
					{
						"id": "auth_existing123"
					}
				]
			}`,
			listAuthStatusCode:   http.StatusOK,
			updateAuthResponse:   `{}`,
			updateAuthStatusCode: http.StatusOK,
			expectedError:        "",
			expectCreateCall:     false,
			expectUpdateCall:     true,
		},
		{
			name:     "Success - Create new authorization when none exists",
			userID:   "user456",
			roleKeys: []string{"subSeed"},
			listAuthResponse: `{
				"authorizations": []
			}`,
			listAuthStatusCode: http.StatusOK,
			createAuthResponse: `{
				"id": "auth_new456"
			}`,
			createAuthStatusCode: http.StatusOK,
			expectedError:        "",
			expectCreateCall:     true,
			expectUpdateCall:     false,
		},
		{
			name:     "Success - Create authorization with empty roles",
			userID:   "user789",
			roleKeys: []string{},
			listAuthResponse: `{
				"authorizations": []
			}`,
			listAuthStatusCode: http.StatusOK,
			createAuthResponse: `{
				"id": "auth_new789"
			}`,
			createAuthStatusCode: http.StatusOK,
			expectedError:        "",
			expectCreateCall:     true,
			expectUpdateCall:     false,
		},
		{
			name:     "Error - Failed to list authorizations",
			userID:   "user999",
			roleKeys: []string{"subGrowth"},
			listAuthResponse: `{
				"error": "internal error"
			}`,
			listAuthStatusCode: http.StatusInternalServerError,
			expectedError:      "failed to check existing authorizations",
			expectCreateCall:   false,
			expectUpdateCall:   false,
		},
		{
			name:     "Error - Failed to create authorization",
			userID:   "user888",
			roleKeys: []string{"subGrowth"},
			listAuthResponse: `{
				"authorizations": []
			}`,
			listAuthStatusCode: http.StatusOK,
			createAuthResponse: `{
				"error": "creation failed"
			}`,
			createAuthStatusCode: http.StatusBadRequest,
			expectedError:        "failed to create authorization",
			expectCreateCall:     true,
			expectUpdateCall:     false,
		},
		{
			name:     "Error - Failed to update authorization",
			userID:   "user777",
			roleKeys: []string{"subGrowth"},
			listAuthResponse: `{
				"authorizations": [
					{
						"id": "auth_existing777"
					}
				]
			}`,
			listAuthStatusCode: http.StatusOK,
			updateAuthResponse: `{
				"error": "update failed"
			}`,
			updateAuthStatusCode: http.StatusBadRequest,
			expectedError:        "failed to set user roles: status 400",
			expectCreateCall:     false,
			expectUpdateCall:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createCalled := false
			updateCalled := false

			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "ListAuthorizations"):
					w.WriteHeader(tt.listAuthStatusCode)
					w.Write([]byte(tt.listAuthResponse))

				case strings.Contains(r.URL.Path, "CreateAuthorization"):
					createCalled = true
					if !tt.expectCreateCall {
						t.Error("CreateAuthorization was called but not expected")
					}
					w.WriteHeader(tt.createAuthStatusCode)
					w.Write([]byte(tt.createAuthResponse))

				case strings.Contains(r.URL.Path, "UpdateAuthorization"):
					updateCalled = true
					if !tt.expectUpdateCall {
						t.Error("UpdateAuthorization was called but not expected")
					}
					// Verify request body contains roles
					body, _ := io.ReadAll(r.Body)
					bodyStr := string(body)
					for _, role := range tt.roleKeys {
						if !strings.Contains(bodyStr, role) {
							t.Errorf("Request body should contain role %s", role)
						}
					}
					w.WriteHeader(tt.updateAuthStatusCode)
					w.Write([]byte(tt.updateAuthResponse))

				default:
					t.Errorf("Unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer mockServer.Close()

			// Set environment variables
			zitadelURL := strings.TrimPrefix(mockServer.URL, "http://")
			zitadelURL = strings.TrimPrefix(zitadelURL, "https://")
			os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)
			os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-token")

			err := SetUserRoles(tt.userID, tt.roleKeys)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Verify that the expected calls were made
			if tt.expectCreateCall && !createCalled {
				t.Error("Expected CreateAuthorization to be called but it wasn't")
			}
			if tt.expectUpdateCall && !updateCalled {
				t.Error("Expected UpdateAuthorization to be called but it wasn't")
			}
		})
	}
}

func TestSetUserRoles_RaceCondition(t *testing.T) {
	// Test concurrent calls to SetUserRoles for the same user
	originalHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalToken := os.Getenv("ZITADEL_BOT_ADMIN_TOKEN")

	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalHost)
		os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", originalToken)
	}()

	callCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.Path, "ListAuthorizations") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"authorizations": []}`))
		} else if strings.Contains(r.URL.Path, "CreateAuthorization") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "auth_123"}`))
		}
	}))
	defer mockServer.Close()

	// Set environment variables
	zitadelURL := strings.TrimPrefix(mockServer.URL, "http://")
	zitadelURL = strings.TrimPrefix(zitadelURL, "https://")
	os.Setenv("ZITADEL_INSTANCE_HOST", zitadelURL)
	os.Setenv("ZITADEL_BOT_ADMIN_TOKEN", "test-token")

	// Run multiple concurrent calls
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			err := SetUserRoles("user123", []string{fmt.Sprintf("role%d", index)})
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		if err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// All calls should have completed successfully
	if callCount < numGoroutines {
		t.Logf("Warning: Only %d calls were made (expected at least %d)", callCount, numGoroutines)
	}
}
