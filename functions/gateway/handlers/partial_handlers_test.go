package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/weaviate/weaviate/entities/models"
)

func TestSetMnmOptions(t *testing.T) {
	// Initialize protocol and save original environment variables
	helpers.InitDefaultProtocol()
	originalTransport := http.DefaultTransport
	originalAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	originalNamespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
	originalAPIToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	originalCfApiClientBaseUrl := os.Getenv("CLOUDFLARE_API_CLIENT_BASE_URL")
	originalCfApiBaseUrl := os.Getenv("CLOUDFLARE_API_BASE_URL")
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalGoEnv := os.Getenv("GO_ENV")

	defer func() {
		http.DefaultTransport = originalTransport
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", originalAccountID)
		os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", originalNamespaceID)
		os.Setenv("CLOUDFLARE_API_TOKEN", originalAPIToken)
		os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", originalCfApiClientBaseUrl)
		os.Setenv("CLOUDFLARE_API_BASE_URL", originalCfApiBaseUrl)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
		os.Setenv("GO_ENV", originalGoEnv)
	}()

	// Set up logging transport for debugging
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Get ports for mock servers
	cfEndpoint := test_helpers.GetNextPort()
	zitadelEndpoint := test_helpers.GetNextPort()

	// Set environment variables
	os.Setenv("GO_ENV", "test")
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-id")
	os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", "test-namespace-id")
	os.Setenv("CLOUDFLARE_API_TOKEN", "test-api-token")
	os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", fmt.Sprintf("http://%s", cfEndpoint))
	os.Setenv("CLOUDFLARE_API_BASE_URL", fmt.Sprintf("http://%s", cfEndpoint))
	os.Setenv("ZITADEL_INSTANCE_HOST", zitadelEndpoint)

	// Re-initialize protocol after setting GO_ENV
	helpers.InitDefaultProtocol()

	// Create mock Cloudflare server
	mockCloudflareServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK CLOUDFLARE HIT: %s %s", r.Method, r.URL.Path)

		// Handle KV store operations
		if strings.Contains(r.URL.Path, "/storage/kv/namespaces/") {
			if r.Method == "GET" {
				// Check if subdomain exists
				if strings.Contains(r.URL.Path, "existing-subdomain") {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`userId=existing-user-123;--p=#FF0000;themeMode=dark`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"success": false}`))
				return
			}

			if r.Method == "PUT" {
				// Mock successful KV store write
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "result": null, "errors": [], "messages": []}`))
				return
			}
		}

		http.Error(w, fmt.Sprintf("unexpected cloudflare request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Create mock Zitadel server
	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK ZITADEL HIT: %s %s", r.Method, r.URL.Path)

		// Mock the GET request to return user metadata
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/metadata/") {
			// Return base64 encoded empty string for subdomain metadata
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"metadata": {"value": ""}}`)) // base64 for empty string
			return
		}

		// Mock the POST request to user metadata
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/metadata/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
			return
		}

		http.Error(w, fmt.Sprintf("unexpected zitadel request: %s %s", r.Method, r.URL), http.StatusBadRequest)
	}))

	// Bind to ports
	cfListener, err := test_helpers.BindToPort(t, cfEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind Cloudflare server: %v", err)
	}
	mockCloudflareServer.Listener = cfListener
	mockCloudflareServer.Start()
	defer mockCloudflareServer.Close()

	zitadelListener, err := test_helpers.BindToPort(t, zitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind Zitadel server: %v", err)
	}
	mockZitadelServer.Listener = zitadelListener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Update environment variables with actual bound addresses
	boundCfAddress := fmt.Sprintf("http://%s", mockCloudflareServer.Listener.Addr().String())
	boundZtAddress := mockZitadelServer.Listener.Addr().String()
	os.Setenv("CLOUDFLARE_API_CLIENT_BASE_URL", boundCfAddress)
	os.Setenv("CLOUDFLARE_API_BASE_URL", boundCfAddress)
	os.Setenv("ZITADEL_INSTANCE_HOST", boundZtAddress)

	t.Logf("🔧 Mock Cloudflare Server bound to: %s", boundCfAddress)
	t.Logf("🔧 Mock Zitadel Server bound to: %s", boundZtAddress)

	tests := []struct {
		name           string
		payload        SetMnmOptionsRequestPayload
		userInfo       helpers.UserInfo
		queryParams    string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name: "successful subdomain set",
			payload: SetMnmOptionsRequestPayload{
				Subdomain:    "test-subdomain",
				PrimaryColor: "#FF0000",
				ThemeMode:    "dark",
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Subdomain set successfully"},
		},
		{
			name: "successful theme update",
			payload: SetMnmOptionsRequestPayload{
				Subdomain:    "theme-subdomain",
				PrimaryColor: "#00FF00",
				ThemeMode:    "light",
			},
			userInfo: helpers.UserInfo{
				Sub: "user456",
			},
			queryParams:    "theme=true",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Theme updated successfully"},
		},
		{
			name: "subdomain already taken error",
			payload: SetMnmOptionsRequestPayload{
				Subdomain:    "existing-subdomain",
				PrimaryColor: "#0000FF",
				ThemeMode:    "auto",
			},
			userInfo: helpers.UserInfo{
				Sub: "different-user-789",
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  []string{"Subdomain already taken"},
		},
		{
			name:    "invalid JSON payload",
			payload: SetMnmOptionsRequestPayload{}, // Empty payload will fail validation
			userInfo: helpers.UserInfo{
				Sub: "user999",
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  []string{"Invalid JSON payload"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			var payloadBytes []byte
			if tt.name == "invalid JSON payload" {
				// Send malformed JSON
				payloadBytes = []byte(`{invalid json}`)
			} else {
				payloadBytes, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest("POST", "/set-mnm-options", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add query params if specified
			if tt.queryParams != "" {
				req.URL.RawQuery = tt.queryParams
			}

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := SetMnmOptions(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

			// Check all required strings are present
			responseBody := rr.Body.String()
			for _, expectedStr := range tt.shouldContain {
				if !strings.Contains(responseBody, expectedStr) {
					t.Errorf("Expected response to contain '%s', got '%s'", expectedStr, responseBody)
				}
			}
		})
	}
}

func TestGeoLookup(t *testing.T) {
	// Initialize and setup environment
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	originalGoEnv := os.Getenv("GO_ENV")
	defer func() {
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
		os.Setenv("GO_ENV", originalGoEnv)
		services.ResetGeoService() // Reset service singleton
	}()

	// Set environment for test mode
	os.Setenv("GO_ENV", "test")
	os.Setenv("DEPLOYMENT_TARGET", helpers.ACT)

	tests := []struct {
		name           string
		payload        GeoLookupInputPayload
		expectedStatus int
		shouldContain  []string
	}{
		{
			name: "successful geo lookup",
			payload: GeoLookupInputPayload{
				Location: "San Francisco, CA",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"type=\"hidden\"", "40.712800", "-74.006000", "New York, NY 10001, USA"},
		},
		{
			name: "successful geo lookup with different location",
			payload: GeoLookupInputPayload{
				Location: "New York, NY",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"type=\"hidden\"", "40.712800", "-74.006000", "New York, NY 10001, USA"},
		},
		{
			name: "empty location validation error",
			payload: GeoLookupInputPayload{
				Location: "",
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  []string{"Invalid Body", "Location", "required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset service for each test
			services.ResetGeoService()

			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/geo-lookup", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Host", "localhost:"+helpers.GO_ACT_SERVER_PORT)

			// Add AWS Lambda context (required for transport layer)
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := GeoLookup(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

			// Check all required strings are present
			responseBody := rr.Body.String()
			for _, expectedStr := range tt.shouldContain {
				if !strings.Contains(responseBody, expectedStr) {
					t.Errorf("Expected response to contain '%s', got '%s'", expectedStr, responseBody)
				}
			}
		})
	}
}

func TestGetEventsPartial(t *testing.T) {
	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport for debugging
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock server using proper port rotation
	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK WEAVIATE HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))
		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Determine if this should return empty results based on query content
			var requestBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&requestBody)
			query, _ := requestBody["query"].(string)
			isEmpty := strings.Contains(query, "nonexistent")

			var events []interface{}
			if !isEmpty {
				events = []interface{}{
					map[string]interface{}{
						"name":           "Test Event",
						"description":    "A test event",
						"eventOwners":    []interface{}{"123"},
						"eventOwnerName": "Event Host",
						"startTime":      int64(1234567890),
						"endTime":        int64(1234567900),
						"address":        "Test Location",
						"timezone":       "America/New_York",
						"_additional": map[string]interface{}{
							"id": "test-event-id",
						},
					},
				}
			}

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						"EventStrict": events,
					},
				},
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock GraphQL response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)
		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	// Parse mock server URL and set environment variables (working pattern)
	mockURL, err := url.Parse(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse mock server URL: %v", err)
	}

	os.Setenv("WEAVIATE_HOST", mockURL.Hostname())
	os.Setenv("WEAVIATE_SCHEME", mockURL.Scheme)
	os.Setenv("WEAVIATE_PORT", mockURL.Port())
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		shouldContain  []string
		mockEmpty      bool
	}{
		{
			name:           "successful events search",
			queryParams:    "q=test&lat=37.7749&lon=-122.4194&radius=10",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Test Event", "LEARN MORE", "Test Location"},
			mockEmpty:      false,
		},
		{
			name:           "events search with list mode",
			queryParams:    "q=test&lat=37.7749&lon=-122.4194&radius=10&list_mode=LIST",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"Test Event", "LEARN MORE", "Test Location"},
			mockEmpty:      false,
		},
		{
			name:           "events search with carousel mode",
			queryParams:    "q=test&lat=37.7749&lon=-122.4194&radius=10&list_mode=CAROUSEL",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"carousel-container", "carousel-item"},
			mockEmpty:      false,
		},
		{
			name:           "no events found",
			queryParams:    "q=nonexistent&lat=37.7749&lon=-122.4194&radius=10",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"No events found", "Expand Your Search"},
			mockEmpty:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/events-partial?"+tt.queryParams, nil)
			rr := httptest.NewRecorder()

			// Execute handler
			handler := GetEventsPartial(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body length: %d", len(rr.Body.String()))

			// Check all required strings are present
			responseBody := rr.Body.String()
			for _, expectedStr := range tt.shouldContain {
				if !strings.Contains(responseBody, expectedStr) {
					t.Errorf("Expected response to contain '%s', got '%s'", expectedStr, responseBody)
				}
			}

			// For successful responses, just verify it's not an error
			if tt.expectedStatus == http.StatusOK && strings.Contains(strings.ToLower(responseBody), "error") {
				t.Errorf("Expected successful response but got error: %s", responseBody)
			}
		})
	}
}

func TestGeoThenPatchSeshuSessionHandler(t *testing.T) {
	// Set up environment for geo service mocking
	originalGoEnv := os.Getenv("GO_ENV")
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	defer func() {
		os.Setenv("GO_ENV", originalGoEnv)
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
		services.ResetGeoService() // Reset singleton
	}()

	os.Setenv("GO_ENV", "test")
	os.Setenv("DEPLOYMENT_TARGET", helpers.ACT)
	services.ResetGeoService() // Reset to pick up test environment

	tests := []struct {
		name           string
		payload        GeoThenSeshuPatchInputPayload
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful geo then patch",
			payload: GeoThenSeshuPatchInputPayload{
				Location: "San Francisco, CA",
				Url:      "https://example.com",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Location Confirmed", // From the GeoLookup badge template
		},
		{
			name: "missing url",
			payload: GeoThenSeshuPatchInputPayload{
				Location: "San Francisco, CA",
				Url:      "", // Empty URL should trigger validation error
			},
			expectedStatus: http.StatusOK,                                   // HTML error responses return 200
			shouldContain:  "Url&#39; failed on the &#39;required&#39; tag", // Validation error message
		},
		{
			name: "missing location",
			payload: GeoThenSeshuPatchInputPayload{
				Location: "", // Empty location should trigger validation error
				Url:      "https://example.com",
			},
			expectedStatus: http.StatusOK,                                        // HTML error responses return 200
			shouldContain:  "Location&#39; failed on the &#39;required&#39; tag", // Validation error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock DynamoDB with proper update functionality
			mockDB := &test_helpers.MockDynamoDBClient{
				UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					t.Logf("Updated seshu session: %s", tt.payload.Url)
					return &dynamodb.UpdateItemOutput{}, nil
				},
			}

			// Create request with proper URL scheme and host
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "http://localhost:"+helpers.GO_ACT_SERVER_PORT+"/geo-patch-seshu", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			// Set URL fields that GetBaseUrlFromReq needs
			req.URL.Scheme = "http"
			req.URL.Host = "localhost:" + helpers.GO_ACT_SERVER_PORT

			// Add AWS Lambda context for error handling
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
			})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			GeoThenPatchSeshuSessionHandler(rr, req, mockDB)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			if tt.shouldContain != "" && !strings.Contains(rr.Body.String(), tt.shouldContain) {
				t.Errorf("Expected response to contain '%s', got '%s'", tt.shouldContain, rr.Body.String())
			}
		})
	}
}

func TestUpdateUserInterests(t *testing.T) {
	// Initialize and setup environment
	helpers.InitDefaultProtocol()
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalGoEnv := os.Getenv("GO_ENV")
	testZitadelEndpoint := test_helpers.GetNextPort()
	os.Setenv("GO_ENV", "test")
	os.Setenv("ZITADEL_INSTANCE_HOST", testZitadelEndpoint)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
		os.Setenv("GO_ENV", originalGoEnv)
	}()

	// Re-initialize protocol after setting GO_ENV
	helpers.InitDefaultProtocol()

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

			t.Logf("🎯 MOCK ZITADEL HIT: %s %s (user: %s, key: %s)", r.Method, r.URL.Path, userID, key)

			switch {
			case userID == "error_user":
				http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
				return
			case key != "interests":
				http.Error(w, `{"error": "unexpected metadata key"}`, http.StatusBadRequest)
				return
			default:
				// Success case - log the flattened interests for verification
				t.Logf("📝 Received interests value: %s", requestBody.Value)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "success"}`))
				return
			}
		}
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))

	// Set up mock server
	listener, err := test_helpers.BindToPort(t, testZitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind to port: %v", err)
	}
	mockZitadelServer.Listener = listener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Set ZITADEL_INSTANCE_HOST to the actual bound address
	boundAddress := mockZitadelServer.Listener.Addr().String()
	os.Setenv("ZITADEL_INSTANCE_HOST", boundAddress)
	t.Logf("🔧 Mock Zitadel Server bound to: %s", boundAddress)

	tests := []struct {
		name           string
		formData       map[string][]string
		userInfo       helpers.UserInfo
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful interests update with categories and subcategories",
			formData: map[string][]string{
				"category":    {"Sports", "Music"},
				"subCategory": {"Football", "Rock Music"},
				"other":       {"ignored"}, // Should be ignored
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "interests have been updated successfully",
		},
		{
			name: "interests update with comma-separated values",
			formData: map[string][]string{
				"category": {"Sports, Music, Technology"},
			},
			userInfo: helpers.UserInfo{
				Sub: "user456",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "interests have been updated successfully",
		},
		{
			name: "interests update with mixed categories",
			formData: map[string][]string{
				"userCategory":    {"Art"},
				"eventCategory":   {"Concerts"},
				"mainSubCategory": {"Classical"},
			},
			userInfo: helpers.UserInfo{
				Sub: "user789",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "interests have been updated successfully",
		},
		{
			name: "empty interests update",
			formData: map[string][]string{
				"other": {"non-category-field"},
			},
			userInfo: helpers.UserInfo{
				Sub: "user999",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "interests have been updated successfully",
		},
		{
			name: "Zitadel API error",
			formData: map[string][]string{
				"category": {"Sports"},
			},
			userInfo: helpers.UserInfo{
				Sub: "error_user",
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  "Failed to save interests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form data request
			form := url.Values{}
			for key, values := range tt.formData {
				for _, value := range values {
					form.Add(key, value)
				}
			}

			req := httptest.NewRequest("POST", "/update-interests", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := UpdateUserInterests(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

			if tt.shouldContain != "" && !strings.Contains(rr.Body.String(), tt.shouldContain) {
				t.Errorf("Expected response to contain '%s', got '%s'", tt.shouldContain, rr.Body.String())
			}
		})
	}
}

func TestUpdateUserAbout(t *testing.T) {
	// Initialize and setup environment
	helpers.InitDefaultProtocol()
	originalZitadelInstanceUrl := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalGoEnv := os.Getenv("GO_ENV")
	testZitadelEndpoint := test_helpers.GetNextPort()
	os.Setenv("GO_ENV", "test")
	os.Setenv("ZITADEL_INSTANCE_HOST", testZitadelEndpoint)
	defer func() {
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelInstanceUrl)
		os.Setenv("GO_ENV", originalGoEnv)
	}()

	// Re-initialize protocol after setting GO_ENV
	helpers.InitDefaultProtocol()

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

			t.Logf("🎯 MOCK ZITADEL HIT: %s %s (user: %s, key: %s)", r.Method, r.URL.Path, userID, key)

			switch {
			case userID == "error_user":
				http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
				return
			case key != "about":
				http.Error(w, `{"error": "unexpected metadata key"}`, http.StatusBadRequest)
				return
			default:
				// Success case
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "success"}`))
				return
			}
		}
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))

	// Set up mock server
	listener, err := test_helpers.BindToPort(t, testZitadelEndpoint)
	if err != nil {
		t.Fatalf("Failed to bind to port: %v", err)
	}
	mockZitadelServer.Listener = listener
	mockZitadelServer.Start()
	defer mockZitadelServer.Close()

	// Set ZITADEL_INSTANCE_HOST to the actual bound address
	boundAddress := mockZitadelServer.Listener.Addr().String()
	os.Setenv("ZITADEL_INSTANCE_HOST", boundAddress)
	t.Logf("🔧 Mock Zitadel Server bound to: %s", boundAddress)

	tests := []struct {
		name           string
		payload        UpdateUserAboutRequestPayload
		userInfo       helpers.UserInfo
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful about update",
			payload: UpdateUserAboutRequestPayload{
				About: "I love attending events and meeting new people!",
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "About section successfully saved",
		},
		{
			name: "empty about field - should succeed (no validation)",
			payload: UpdateUserAboutRequestPayload{
				About: "",
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "About section successfully saved",
		},
		{
			name: "Zitadel API error",
			payload: UpdateUserAboutRequestPayload{
				About: "Some about text",
			},
			userInfo: helpers.UserInfo{
				Sub: "error_user",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Failed to update &#39;about&#39; field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/update-about", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := UpdateUserAbout(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

			if tt.shouldContain != "" && !strings.Contains(rr.Body.String(), tt.shouldContain) {
				t.Errorf("Expected response to contain '%s', got '%s'", tt.shouldContain, rr.Body.String())
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isFakeData", func(t *testing.T) {
		if !isFakeData(services.FakeCity) {
			t.Errorf("Expected FakeCity to be identified as fake data")
		}
		if !isFakeData(services.FakeUrl1) {
			t.Errorf("Expected FakeUrl1 to be identified as fake data")
		}
		if isFakeData("real data") {
			t.Errorf("Expected 'real data' to not be identified as fake data")
		}
	})

	t.Run("getValidatedEvents", func(t *testing.T) {
		candidates := []types.EventInfo{
			{
				EventTitle:     "Valid Event",
				EventLocation:  "Valid Location",
				EventStartTime: "2024-01-01T10:00:00Z",
			},
			{
				EventTitle:     services.FakeEventTitle1, // fake data
				EventLocation:  "Valid Location",
				EventStartTime: "2024-01-01T10:00:00Z",
			},
		}

		validations := []types.EventBoolValid{
			{
				EventValidateTitle:     true,
				EventValidateLocation:  true,
				EventValidateStartTime: true,
			},
			{
				EventValidateTitle:     true, // but title is fake
				EventValidateLocation:  true,
				EventValidateStartTime: true,
			},
		}

		result := getValidatedEvents(candidates, validations, false)
		if len(result) != 1 {
			t.Errorf("Expected 1 validated event, got %d", len(result))
		}
		if len(result) > 0 && result[0].EventTitle != "Valid Event" {
			t.Errorf("Expected first event title to be 'Valid Event', got '%s'", result[0].EventTitle)
		}
	})

	t.Run("DOM helper functions", func(t *testing.T) {
		// Test HTML with various selectable elements
		testHTML := `
		<!DOCTYPE html>
		<html>
		<head><title>Test Document</title></head>
		<body>
			<div id="main-content" class="container primary">
				<header class="page-header">
					<h1 id="title">Event Management</h1>
					<nav class="navigation">
						<a href="/home">Home</a>
						<a href="/events">Events</a>
					</nav>
				</header>
				<main class="content-area">
					<section class="event-section">
						<h2>Upcoming Events</h2>
						<article class="event-card" data-event="1">
							<h3>Concert Night</h3>
							<p class="event-time">Friday 7pm</p>
							<p class="event-location">City Hall</p>
							<div class="event-description">
								<span>Join us for an amazing</span> <strong>concert experience</strong>
							</div>
						</article>
						<article class="event-card" data-event="2">
							<h3>Workshop Day</h3>
							<p class="event-time">Saturday 2pm</p>
							<div class="nested">
								<span>Workshop Day content</span>
							</div>
						</article>
					</section>
				</main>
			</div>
			<footer>
				<p>Footer content</p>
			</footer>
		</body>
		</html>`

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(testHTML))
		if err != nil {
			t.Fatalf("Failed to parse test HTML: %v", err)
		}

		// Test getFullDomPath function
		t.Run("getFullDomPath", func(t *testing.T) {
			tests := []struct {
				name         string
				selector     string
				expectedPath string
			}{
				{
					name:         "element with ID - stops at ID",
					selector:     "#title",
					expectedPath: "h1#title", // Function stops at elements with IDs since they're unique
				},
				{
					name:         "element with class - stops at parent ID",
					selector:     ".event-time",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > p.event-time",
				},
				{
					name:         "nested element - stops at parent ID",
					selector:     ".event-description strong",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > div.event-description > strong",
				},
				{
					name:         "element without ID or class",
					selector:     "footer p",
					expectedPath: "html > body > footer > p",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					selection := doc.Find(tt.selector).First()
					if selection.Length() == 0 {
						t.Fatalf("Could not find element with selector: %s", tt.selector)
					}

					result := getFullDomPath(selection)
					if result != tt.expectedPath {
						t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, result)
					}
				})
			}
		})

		// Test findTagByExactText function
		t.Run("findTagByExactText", func(t *testing.T) {
			tests := []struct {
				name         string
				targetText   string
				expectedPath string
			}{
				{
					name:         "exact match in header with ID",
					targetText:   "Event Management",
					expectedPath: "h1#title", // Stops at element with ID
				},
				{
					name:         "exact match in navigation",
					targetText:   "Events",
					expectedPath: "div#main-content > header.page-header > nav.navigation > a",
				},
				{
					name:         "exact match in article",
					targetText:   "Concert Night",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > h3",
				},
				{
					name:         "exact match with time",
					targetText:   "Friday 7pm",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > p.event-time",
				},
				{
					name:         "no match for partial text",
					targetText:   "Concert", // partial match should return empty
					expectedPath: "",
				},
				{
					name:         "no match for non-existent text",
					targetText:   "Non-existent Text",
					expectedPath: "",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := findTagByExactText(doc, tt.targetText)
					if result != tt.expectedPath {
						t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, result)
					}
				})
			}
		})

		// Test findTagByPartialText function
		t.Run("findTagByPartialText", func(t *testing.T) {
			tests := []struct {
				name         string
				targetText   string
				expectedPath string
			}{
				{
					name:         "partial match finds first occurrence",
					targetText:   "Event",
					expectedPath: "div#main-content > main.content-area > section.event-section > h2", // Finds "Upcoming Events" first
				},
				{
					name:         "partial match in time",
					targetText:   "Friday",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > p.event-time",
				},
				{
					name:         "partial match in location",
					targetText:   "City",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > p.event-location",
				},
				{
					name:         "partial match finds specific child element",
					targetText:   "concert experience",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > div.event-description > strong", // Finds the strong element
				},
				{
					name:         "partial match finds nested span",
					targetText:   "Workshop Day",
					expectedPath: "div#main-content > main.content-area > section.event-section > article.event-card > div.nested > span", // Finds nested span with "Workshop Day content"
				},
				{
					name:         "no match for non-existent text",
					targetText:   "Nonexistent",
					expectedPath: "",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := findTagByPartialText(doc, tt.targetText)
					if result != tt.expectedPath {
						t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, result)
					}
				})
			}
		})

		// Test edge cases
		t.Run("edge cases", func(t *testing.T) {
			emptyHTML := `<html><body></body></html>`
			emptyDoc, _ := goquery.NewDocumentFromReader(strings.NewReader(emptyHTML))

			// Test with empty document
			result := findTagByExactText(emptyDoc, "anything")
			if result != "" {
				t.Errorf("Expected empty result for empty document, got '%s'", result)
			}

			result = findTagByPartialText(emptyDoc, "anything")
			if result != "" {
				t.Errorf("Expected empty result for empty document, got '%s'", result)
			}

			// Test with empty selection for getFullDomPath
			emptySelection := emptyDoc.Find("nonexistent")
			result = getFullDomPath(emptySelection)
			if result != "" {
				t.Errorf("Expected empty result for empty selection, got '%s'", result)
			}
		})
	})
}

func TestGetEventAdminChildrenPartial(t *testing.T) {
	// Save original environment variables
	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport for debugging
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock server using proper port rotation
	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK WEAVIATE HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))
		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Determine request type based on query content - parallel calls means we need to handle both
			var requestBody map[string]interface{}
			bodyBytes, _ := io.ReadAll(r.Body)
			json.Unmarshal(bodyBytes, &requestBody)
			query, _ := requestBody["query"].(string)

			// Different responses for GetEventByID vs SearchEvents
			if strings.Contains(query, `where:{operator: And operands:[{operator: And operands:[{operator: GreaterThanEqual`) {
				// This is a SearchWeaviateEvents call (for children)
				t.Logf("   └─ Detected SearchWeaviateEvents call")
				mockResponse := models.GraphQLResponse{
					Data: map[string]models.JSONObject{
						"Get": map[string]interface{}{
							"EventStrict": []interface{}{
								map[string]interface{}{
									"name":            "Child Event 1",
									"description":     "First child event",
									"eventOwners":     []interface{}{"123"},
									"eventOwnerName":  "Event Host",
									"eventSourceType": "EVS", // helpers.ES_EVENT_SERIES
									"startTime":       int64(1234567890),
									"endTime":         int64(1234567900),
									"address":         "Child Location 1",
									"timezone":        "America/New_York",
									"_additional": map[string]interface{}{
										"id": "child-event-1",
									},
								},
								map[string]interface{}{
									"name":            "Child Event 2",
									"description":     "Second child event",
									"eventOwners":     []interface{}{"123"},
									"eventOwnerName":  "Event Host",
									"eventSourceType": "EVS", // helpers.ES_EVENT_SERIES
									"startTime":       int64(1234567800),
									"endTime":         int64(1234567850),
									"address":         "Child Location 2",
									"timezone":        "America/New_York",
									"_additional": map[string]interface{}{
										"id": "child-event-2",
									},
								},
							},
						},
					},
				}
				responseBytes, _ := json.Marshal(mockResponse)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(responseBytes)
			} else {
				// This is a GetWeaviateEventByID call (for parent)
				t.Logf("   └─ Detected GetWeaviateEventByID call")
				mockResponse := models.GraphQLResponse{
					Data: map[string]models.JSONObject{
						"Get": map[string]interface{}{
							"EventStrict": []interface{}{
								map[string]interface{}{
									"name":            "Parent Event",
									"description":     "Parent event description",
									"eventOwners":     []interface{}{"123"},
									"eventOwnerName":  "Event Host",
									"eventSourceType": "SLF_EVS", // helpers.ES_SERIES_PARENT
									"startTime":       int64(1234567000),
									"endTime":         int64(1234568000),
									"address":         "Parent Location",
									"timezone":        "America/New_York",
									"_additional": map[string]interface{}{
										"id": "parent-event-123",
									},
								},
							},
						},
					},
				}
				responseBytes, _ := json.Marshal(mockResponse)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(responseBytes)
			}
		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	// Parse mock server URL and set environment variables (working pattern)
	mockURL, err := url.Parse(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse mock server URL: %v", err)
	}

	os.Setenv("WEAVIATE_HOST", mockURL.Hostname())
	os.Setenv("WEAVIATE_SCHEME", mockURL.Scheme)
	os.Setenv("WEAVIATE_PORT", mockURL.Port())
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	tests := []struct {
		name           string
		eventId        string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name:           "successful admin children fetch with parallel calls",
			eventId:        "parent-event-123",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"event-diffs", "child-event-1", "child-event-2", "event_0_Id", "event_1_Id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with event ID in URL path
			req := httptest.NewRequest("GET", "/admin/event/"+tt.eventId+"/children", nil)
			req = mux.SetURLVars(req, map[string]string{helpers.EVENT_ID_KEY: tt.eventId})
			rr := httptest.NewRecorder()

			// Execute handler
			handler := GetEventAdminChildrenPartial(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body length: %d", len(rr.Body.String()))

			// Check all required strings are present
			responseBody := rr.Body.String()
			for _, expectedStr := range tt.shouldContain {
				if !strings.Contains(responseBody, expectedStr) {
					t.Errorf("Expected response to contain '%s', got '%s'", expectedStr, responseBody)
				}
			}

			// For successful responses, just verify it's not an error
			if tt.expectedStatus == http.StatusOK && strings.Contains(strings.ToLower(responseBody), "error") {
				t.Errorf("Expected successful response but got error: %s", responseBody)
			}
		})
	}
}

func TestSubmitSeshuSession(t *testing.T) {
	// Save original environment variables
	originalGoEnv := os.Getenv("GO_ENV")
	originalTransport := http.DefaultTransport
	defer func() {
		os.Setenv("GO_ENV", originalGoEnv)
		http.DefaultTransport = originalTransport
		services.ResetGeoService() // Reset service singleton
	}()

	// Set environment for test mode
	os.Setenv("GO_ENV", "test")

	// Set up logging transport for debugging
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	tests := []struct {
		name           string
		payload        SeshuSessionEventsPayload
		userInfo       helpers.UserInfo
		mockDB         *test_helpers.MockDynamoDBClient
		mockPostgres   *test_helpers.MockPostgresService
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful seshu session submission",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
				EventBoolValid: []types.EventBoolValid{
					{
						EventValidateTitle:       true,
						EventValidateLocation:    true,
						EventValidateStartTime:   true,
						EventValidateEndTime:     false,
						EventValidateURL:         true,
						EventValidateDescription: false,
					},
				},
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			mockDB: &test_helpers.MockDynamoDBClient{
				GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					// Mock successful SeshuSession retrieval
					item := map[string]dynamodb_types.AttributeValue{
						"ownerId":           &dynamodb_types.AttributeValueMemberS{Value: "user123"},
						"url":               &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/events"},
						"urlDomain":         &dynamodb_types.AttributeValueMemberS{Value: "example.com"},
						"urlPath":           &dynamodb_types.AttributeValueMemberS{Value: "/events"},
						"locationLatitude":  &dynamodb_types.AttributeValueMemberN{Value: "37.7749"},
						"locationLongitude": &dynamodb_types.AttributeValueMemberN{Value: "-122.4194"},
						"locationAddress":   &dynamodb_types.AttributeValueMemberS{Value: "San Francisco, CA"},
						"html":              &dynamodb_types.AttributeValueMemberS{Value: "<html><body><h1>Test Event</h1><p>Test Location</p><p>2024-01-01T10:00:00Z</p></body></html>"},
						"childId":           &dynamodb_types.AttributeValueMemberS{Value: ""},
						"eventCandidates": &dynamodb_types.AttributeValueMemberL{
							Value: []dynamodb_types.AttributeValue{
								&dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"event_title":          &dynamodb_types.AttributeValueMemberS{Value: "Test Event"},
										"event_location":       &dynamodb_types.AttributeValueMemberS{Value: "Test Location"},
										"event_start_datetime": &dynamodb_types.AttributeValueMemberS{Value: "2024-01-01T10:00:00Z"},
										"event_end_datetime":   &dynamodb_types.AttributeValueMemberS{Value: ""},
										"event_url":            &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/event/1"},
										"event_description":    &dynamodb_types.AttributeValueMemberS{Value: ""},
										"scrape_mode":          &dynamodb_types.AttributeValueMemberS{Value: "init"},
									},
								},
							},
						},
						"eventValidations": &dynamodb_types.AttributeValueMemberL{
							Value: []dynamodb_types.AttributeValue{
								&dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"event_title":          &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_location":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_start_datetime": &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_end_datetime":   &dynamodb_types.AttributeValueMemberBOOL{Value: false},
										"event_url":            &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_description":    &dynamodb_types.AttributeValueMemberBOOL{Value: false},
									},
								},
							},
						},
						"status":    &dynamodb_types.AttributeValueMemberS{Value: "draft"},
						"createdAt": &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"updatedAt": &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"expireAt":  &dynamodb_types.AttributeValueMemberN{Value: "1234654290"},
					}
					return &dynamodb.GetItemOutput{Item: item}, nil
				},
				UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				},
			},
			mockPostgres: &test_helpers.MockPostgresService{
				GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) {
					return []types.SeshuJob{}, nil // No existing jobs
				},
				CreateSeshuJobFunc: func(ctx context.Context, job types.SeshuJob) error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "",
		},
		{
			name: "unauthorized user",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
			},
			userInfo: helpers.UserInfo{
				Sub: "", // Empty user ID
			},
			mockDB:         &test_helpers.MockDynamoDBClient{},
			mockPostgres:   &test_helpers.MockPostgresService{},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  "You must be logged in to submit an event source",
		},
		{
			name:    "invalid JSON payload",
			payload: SeshuSessionEventsPayload{}, // Empty payload will fail validation
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			mockDB:         &test_helpers.MockDynamoDBClient{},
			mockPostgres:   &test_helpers.MockPostgresService{},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  "Invalid request body",
		},
		{
			name: "event source already exists",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
				EventBoolValid: []types.EventBoolValid{
					{
						EventValidateTitle:       true,
						EventValidateLocation:    true,
						EventValidateStartTime:   true,
						EventValidateEndTime:     false,
						EventValidateURL:         true,
						EventValidateDescription: false,
					},
				},
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			mockDB: &test_helpers.MockDynamoDBClient{
				GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					// Mock successful SeshuSession retrieval
					item := map[string]dynamodb_types.AttributeValue{
						"ownerId":           &dynamodb_types.AttributeValueMemberS{Value: "user123"},
						"url":               &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/events"},
						"urlDomain":         &dynamodb_types.AttributeValueMemberS{Value: "example.com"},
						"urlPath":           &dynamodb_types.AttributeValueMemberS{Value: "/events"},
						"locationLatitude":  &dynamodb_types.AttributeValueMemberN{Value: "37.7749"},
						"locationLongitude": &dynamodb_types.AttributeValueMemberN{Value: "-122.4194"},
						"locationAddress":   &dynamodb_types.AttributeValueMemberS{Value: "San Francisco, CA"},
						"html":              &dynamodb_types.AttributeValueMemberS{Value: "<html><body><h1>Test Event</h1></body></html>"},
						"childId":           &dynamodb_types.AttributeValueMemberS{Value: ""},
						"eventCandidates":   &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{}},
						"eventValidations":  &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{}},
						"status":            &dynamodb_types.AttributeValueMemberS{Value: "draft"},
						"createdAt":         &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"updatedAt":         &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"expireAt":          &dynamodb_types.AttributeValueMemberN{Value: "1234654290"},
					}
					return &dynamodb.GetItemOutput{Item: item}, nil
				},
				UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				},
			},
			mockPostgres: &test_helpers.MockPostgresService{
				GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) {
					// Return existing job to trigger conflict
					return []types.SeshuJob{
						{
							NormalizedUrlKey: "https://example.com/events",
							OwnerID:          "user123",
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  "This event source URL already exists",
		},
		{
			name: "postgres service error",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
				EventBoolValid: []types.EventBoolValid{
					{
						EventValidateTitle:       true,
						EventValidateLocation:    true,
						EventValidateStartTime:   true,
						EventValidateEndTime:     false,
						EventValidateURL:         true,
						EventValidateDescription: false,
					},
				},
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			mockDB: &test_helpers.MockDynamoDBClient{
				GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					// Mock successful SeshuSession retrieval
					item := map[string]dynamodb_types.AttributeValue{
						"ownerId":           &dynamodb_types.AttributeValueMemberS{Value: "user123"},
						"url":               &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/events"},
						"urlDomain":         &dynamodb_types.AttributeValueMemberS{Value: "example.com"},
						"urlPath":           &dynamodb_types.AttributeValueMemberS{Value: "/events"},
						"locationLatitude":  &dynamodb_types.AttributeValueMemberN{Value: "37.7749"},
						"locationLongitude": &dynamodb_types.AttributeValueMemberN{Value: "-122.4194"},
						"locationAddress":   &dynamodb_types.AttributeValueMemberS{Value: "San Francisco, CA"},
						"html":              &dynamodb_types.AttributeValueMemberS{Value: "<html><body><h1>Test Event</h1></body></html>"},
						"childId":           &dynamodb_types.AttributeValueMemberS{Value: ""},
						"eventCandidates":   &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{}},
						"eventValidations":  &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{}},
						"status":            &dynamodb_types.AttributeValueMemberS{Value: "draft"},
						"createdAt":         &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"updatedAt":         &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"expireAt":          &dynamodb_types.AttributeValueMemberN{Value: "1234654290"},
					}
					return &dynamodb.GetItemOutput{Item: item}, nil
				},
			},
			mockPostgres: &test_helpers.MockPostgresService{
				GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) {
					return nil, fmt.Errorf("database connection failed")
				},
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  "Failed to get SeshuJobs",
		},
		{
			name: "seshu session submission with missing location data should succeed when no default timezone is set",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
				EventBoolValid: []types.EventBoolValid{
					{
						EventValidateTitle:       true,
						EventValidateLocation:    true,
						EventValidateStartTime:   true,
						EventValidateEndTime:     false,
						EventValidateURL:         true,
						EventValidateDescription: false,
					},
				},
			},
			userInfo: helpers.UserInfo{
				Sub: "user123",
			},
			mockDB: &test_helpers.MockDynamoDBClient{
				GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
					// Mock SeshuSession with missing location data but valid events (copied from working test)
					item := map[string]dynamodb_types.AttributeValue{
						"ownerId":           &dynamodb_types.AttributeValueMemberS{Value: "user123"},
						"url":               &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/events"},
						"urlDomain":         &dynamodb_types.AttributeValueMemberS{Value: "example.com"},
						"urlPath":           &dynamodb_types.AttributeValueMemberS{Value: "/events"},
						"locationLatitude":  &dynamodb_types.AttributeValueMemberN{Value: "37.7749"},   // Valid coordinates but no location data - should derive from event locations
						"locationLongitude": &dynamodb_types.AttributeValueMemberN{Value: "-122.4194"}, // Valid coordinates but no location data - should derive from event locations
						"locationAddress":   &dynamodb_types.AttributeValueMemberS{Value: ""},          // Empty address
						"html":              &dynamodb_types.AttributeValueMemberS{Value: "<html><body><h1>Test Event</h1><p>Test Location</p><p>2024-01-01T10:00:00Z</p></body></html>"},
						"childId":           &dynamodb_types.AttributeValueMemberS{Value: ""},
						"eventCandidates": &dynamodb_types.AttributeValueMemberL{
							Value: []dynamodb_types.AttributeValue{
								&dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"event_title":          &dynamodb_types.AttributeValueMemberS{Value: "Test Event"},    // Use same values as working test
										"event_location":       &dynamodb_types.AttributeValueMemberS{Value: "Test Location"}, // Use same values as working test
										"event_start_datetime": &dynamodb_types.AttributeValueMemberS{Value: "2024-01-01T10:00:00Z"},
										"event_end_datetime":   &dynamodb_types.AttributeValueMemberS{Value: ""},
										"event_url":            &dynamodb_types.AttributeValueMemberS{Value: "https://example.com/event/1"},
										"event_description":    &dynamodb_types.AttributeValueMemberS{Value: ""},
										"scrape_mode":          &dynamodb_types.AttributeValueMemberS{Value: "init"},
									},
								},
							},
						},
						"eventValidations": &dynamodb_types.AttributeValueMemberL{
							Value: []dynamodb_types.AttributeValue{
								&dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"event_title":          &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_location":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_start_datetime": &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_end_datetime":   &dynamodb_types.AttributeValueMemberBOOL{Value: false},
										"event_url":            &dynamodb_types.AttributeValueMemberBOOL{Value: true},
										"event_description":    &dynamodb_types.AttributeValueMemberBOOL{Value: false},
									},
								},
							},
						},
						"status":    &dynamodb_types.AttributeValueMemberS{Value: "draft"},
						"createdAt": &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"updatedAt": &dynamodb_types.AttributeValueMemberN{Value: "1234567890"},
						"expireAt":  &dynamodb_types.AttributeValueMemberN{Value: "1234654290"},
					}
					return &dynamodb.GetItemOutput{Item: item}, nil
				},
				UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
					return &dynamodb.UpdateItemOutput{}, nil
				},
			},
			mockPostgres: &test_helpers.MockPostgresService{
				GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) {
					return []types.SeshuJob{}, nil // No existing jobs
				},
				CreateSeshuJobFunc: func(ctx context.Context, job types.SeshuJob) error {
					return nil // Should succeed - timezone derived from event locations
				},
			},
			expectedStatus: http.StatusOK,                      // Handler should succeed
			shouldContain:  "Your Event Source has been added", // Should contain success message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock DB for this test
			if tt.mockDB != nil {
				transport.SetTestDB(tt.mockDB)
			}

			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/submit-seshu-session", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			ctx = context.WithValue(ctx, "targetUrl", tt.payload.Url)
			ctx = context.WithValue(ctx, "mockPostgresService", tt.mockPostgres)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Set up mock DynamoDB client
			transport.SetTestDB(tt.mockDB)

			// Execute handler with mock DB
			handler := SubmitSeshuSession(rr, req)
			handler(rr, req)

			// Verify response
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

			if tt.shouldContain != "" && !strings.Contains(rr.Body.String(), tt.shouldContain) {
				t.Errorf("Expected response to contain '%s', got '%s'", tt.shouldContain, rr.Body.String())
			}
		})
	}
}
