package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func init() {
	os.Setenv("GO_ENV", GO_TEST_ENV)
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
			subdomainValue:  "test-subdomain",
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
		if subdomainValue == "test-subdomain" {
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
			subdomainValue: "test-subdomain",
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
