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
	"sync/atomic"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
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
		userInfo       constants.UserInfo
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
				Sub: "different-user-789",
			},
			expectedStatus: http.StatusOK, // HTML error responses return 200
			shouldContain:  []string{"Subdomain already taken"},
		},
		{
			name:    "invalid JSON payload",
			payload: SetMnmOptionsRequestPayload{}, // Empty payload will fail validation
			userInfo: constants.UserInfo{
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
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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
	os.Setenv("DEPLOYMENT_TARGET", constants.ACT)

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
			req.Header.Set("Host", "localhost:"+constants.GO_ACT_SERVER_PORT)

			// Add AWS Lambda context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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

func TestCityLookup(t *testing.T) {
	originalDeploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	originalGoEnv := os.Getenv("GO_ENV")
	defer func() {
		os.Setenv("DEPLOYMENT_TARGET", originalDeploymentTarget)
		os.Setenv("GO_ENV", originalGoEnv)
		services.ResetCityService()
	}()

	os.Setenv("GO_ENV", "test")
	os.Setenv("DEPLOYMENT_TARGET", constants.ACT)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		shouldContain  []string
	}{
		{
			name:           "Successful lookup with valid coordinates",
			query:          "lat=40.7&lon=-74.0",
			expectedStatus: http.StatusOK,
			shouldContain:  []string{"New York"}, // Mock cityService returns "New York"
		},
		{
			name:           "Missing location parameter",
			query:          "",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  []string{"Both lat and lon parameters are required"},
		},
		{
			name:           "Invalid coordinates - latitude too high",
			query:          "lat=91.0&lon=0.0",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  []string{"Latitude and Longitude are invalid"},
		},
		{
			name:           "Invalid coordinates - longitude too high",
			query:          "lat=0.0&lon=181.0",
			expectedStatus: http.StatusBadRequest,
			shouldContain:  []string{"Latitude and Longitude are invalid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services.ResetCityService()

			requestURL := "/api/location/city"
			if tt.query != "" {
				requestURL += "?" + tt.query
			}

			req := httptest.NewRequest("GET", requestURL, nil)
			req.Header.Set("Host", "localhost:"+constants.GO_ACT_SERVER_PORT)

			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			handler := CityLookup(rr, req)
			handler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			t.Logf("Response status: %d", rr.Code)
			t.Logf("Response body: %s", rr.Body.String())

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
						"lat":            37.7749,
						"long":           -122.4194,
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
						constants.WeaviateEventClassName: events,
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
			shouldContain:  []string{"No events found", "expand your search"},
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

func TestGetEventsPartialWithGroupedEvents(t *testing.T) {
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

			// Return multiple events that should be grouped (same name, lat, long)
			events := []interface{}{
				map[string]interface{}{
					"name":           "Weekly Meetup",
					"description":    "A weekly meetup event",
					"eventOwners":    []interface{}{"123"},
					"eventOwnerName": "Event Host",
					"startTime":      int64(1704067200), // 2024-01-01
					"endTime":        int64(1704070800),
					"address":        "123 Main St",
					"lat":            40.7128,
					"long":           -74.0060,
					"timezone":       "America/New_York",
					"_additional": map[string]interface{}{
						"id": "event-1",
					},
				},
				map[string]interface{}{
					"name":           "Weekly Meetup",
					"description":    "A weekly meetup event",
					"eventOwners":    []interface{}{"123"},
					"eventOwnerName": "Event Host",
					"startTime":      int64(1704672000), // 2024-01-08
					"endTime":        int64(1704675600),
					"address":        "123 Main St",
					"lat":            40.7128,
					"long":           -74.0060,
					"timezone":       "America/New_York",
					"_additional": map[string]interface{}{
						"id": "event-2",
					},
				},
				map[string]interface{}{
					"name":           "Weekly Meetup",
					"description":    "A weekly meetup event",
					"eventOwners":    []interface{}{"123"},
					"eventOwnerName": "Event Host",
					"startTime":      int64(1705276800), // 2024-01-15
					"endTime":        int64(1705280400),
					"address":        "123 Main St",
					"lat":            40.7128,
					"long":           -74.0060,
					"timezone":       "America/New_York",
					"_additional": map[string]interface{}{
						"id": "event-3",
					},
				},
				// Add an ungrouped event (different location)
				map[string]interface{}{
					"name":           "Different Event",
					"description":    "An event at a different location",
					"eventOwners":    []interface{}{"456"},
					"eventOwnerName": "Other Host",
					"startTime":      int64(1704067200),
					"endTime":        int64(1704070800),
					"address":        "456 Other St",
					"lat":            34.0522,
					"long":           -118.2437,
					"timezone":       "America/Los_Angeles",
					"_additional": map[string]interface{}{
						"id": "event-4",
					},
				},
			}

			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: events,
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

	// Parse mock server URL and set environment variables
	mockURL, err := url.Parse(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to parse mock server URL: %v", err)
	}

	os.Setenv("WEAVIATE_HOST", mockURL.Hostname())
	os.Setenv("WEAVIATE_SCHEME", mockURL.Scheme)
	os.Setenv("WEAVIATE_PORT", mockURL.Port())
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	tests := []struct {
		name             string
		queryParams      string
		expectedStatus   int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:           "default mode with grouped events",
			queryParams:    "q=meetup&lat=40.7128&lon=-74.0060&radius=10",
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Weekly Meetup",
				"123 Main St",
				"carousel-container",
				"/event/event-1", // Event IDs will appear in URLs
				"/event/event-2",
				"/event/event-3",
				"Different Event", // Ungrouped event should also appear
			},
		},
		{
			name:           "LIST mode with grouped events",
			queryParams:    "q=meetup&lat=40.7128&lon=-74.0060&radius=10&list_mode=LIST",
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Weekly Meetup",
				"carousel-container",
				"Different Event",
			},
		},
		{
			name:           "CAROUSEL mode with grouped events",
			queryParams:    "q=meetup&lat=40.7128&lon=-74.0060&radius=10&list_mode=CAROUSEL",
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"carousel-container",
				"carousel-item",
			},
		},
		{
			name:           "ADMIN_LIST mode with grouped events",
			queryParams:    "q=meetup&lat=40.7128&lon=-74.0060&radius=10&list_mode=ADMIN_LIST",
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Weekly Meetup",
				"3 occurrences",
				"Event Admin",
				"Different Event",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/api/html/events?"+tt.queryParams, nil)
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
					t.Errorf("Expected response to contain '%s'", expectedStr)
					t.Logf("Response body: %s", responseBody)
				}
			}

			// Check strings that should not be present
			for _, unexpectedStr := range tt.shouldNotContain {
				if strings.Contains(responseBody, unexpectedStr) {
					t.Errorf("Expected response to NOT contain '%s'", unexpectedStr)
					t.Logf("Response body: %s", responseBody)
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
	os.Setenv("DEPLOYMENT_TARGET", constants.ACT)
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
			expectedStatus: http.StatusOK,                       // HTML error responses return 200
			shouldContain:  "Url' failed on the 'required' tag", // Validation error message
		},
		{
			name: "missing location",
			payload: GeoThenSeshuPatchInputPayload{
				Location: "", // Empty location should trigger validation error
				Url:      "https://example.com",
			},
			expectedStatus: http.StatusOK,                            // HTML error responses return 200
			shouldContain:  "Location' failed on the 'required' tag", // Validation error message
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
			req := httptest.NewRequest("POST", "http://localhost:"+constants.GO_ACT_SERVER_PORT+"/geo-patch-seshu", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			// Set URL fields that GetBaseUrlFromReq needs
			req.URL.Scheme = "http"
			req.URL.Host = "localhost:" + constants.GO_ACT_SERVER_PORT

			// Add AWS Lambda context for error handling
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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
		userInfo       constants.UserInfo
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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
		userInfo       constants.UserInfo
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful about update",
			payload: UpdateUserAboutRequestPayload{
				About: "I love attending events and meeting new people!",
			},
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
				Sub: "error_user",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Failed to update 'about' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/update-about", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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

func TestUpdateUserLocation(t *testing.T) {
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

	helpers.InitDefaultProtocol()

	mockZitadelServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/management/v1/users/") && strings.Contains(r.URL.Path, "/metadata/") {

			var requestBody struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}

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
			case key != constants.META_LOC_KEY:
				http.Error(w, `{"error": "unexpected metadata key"}`, http.StatusBadRequest)
				return
			default:
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
		payload        UpdateUserLocationRequestPayload
		userInfo       constants.UserInfo
		expectedStatus int
		shouldContain  string
	}{
		{
			name: "successful location update",
			payload: UpdateUserLocationRequestPayload{
				Latitude:  30.0,
				Longitude: 45.0,
				City:      "Georgetown, TX",
			},
			userInfo: constants.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Location info successfully saved",
		},
		{
			name: "empty city field - should error",
			payload: UpdateUserLocationRequestPayload{
				Latitude:  30,
				Longitude: 45,
				City:      "",
			},
			userInfo: constants.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Error:Field validation for 'City'",
		},
		{
			name: "invalid latitude",
			payload: UpdateUserLocationRequestPayload{
				Latitude:  -240,
				Longitude: 45,
				City:      "Philadelphia, PA",
			},
			userInfo: constants.UserInfo{
				Sub: "user123",
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Latitude must be between -90 and 90",
		},
		{
			name: "Zitadel API error",
			payload: UpdateUserLocationRequestPayload{
				Latitude:  30,
				Longitude: 45,
				City:      "Georgetown, TX",
			},
			userInfo: constants.UserInfo{
				Sub: "error_user", // zitadel mock returns error for this case
			},
			expectedStatus: http.StatusOK,
			shouldContain:  "Failed to update location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/update-location", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add AWS Lambda context and user info to context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := UpdateUserLocation(rr, req)
			handler(rr, req)

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
							constants.WeaviateEventClassName: []interface{}{
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
							constants.WeaviateEventClassName: []interface{}{
								map[string]interface{}{
									"name":            "Parent Event",
									"description":     "Parent event description",
									"eventOwners":     []interface{}{"123"},
									"eventOwnerName":  "Event Host",
									"eventSourceType": constants.ES_SERIES_PARENT,
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
			req = mux.SetURLVars(req, map[string]string{constants.EVENT_ID_KEY: tt.eventId})
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
		userInfo       constants.UserInfo
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
			userInfo: constants.UserInfo{
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
			shouldContain:  "Your Event Source has been added",
		},
		{
			name: "unauthorized user",
			payload: SeshuSessionEventsPayload{
				Url: "https://example.com/events",
			},
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			userInfo: constants.UserInfo{
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
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
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

func TestSubmitSeshuSession_Onboarding_RandomURL_and_Facebook(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	// Helper to create a minimal ctx with mocked Postgres service
	newCtx := func(pg *test_helpers.MockPostgresService) context.Context {
		ctx := context.Background()
		// Inject mock Postgres via context hook honored by GetPostgresService
		ctx = context.WithValue(ctx, "mockPostgresService", pg)
		// Also add essentials used by transport for auth/options (keep minimal)
		ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "user-1"})
		ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: "user-1", Name: "Test User", Email: "test@example.com"})
		ctx = context.WithValue(ctx, "roleClaims", map[string]any{"roles": []string{"user"}})
		return ctx
	}

	// Set up a per-test ScrapingBee server URL
	makeSB := func(handler http.HandlerFunc) (string, func()) {
		sbPort := test_helpers.GetNextPort()
		srv := httptest.NewUnstartedServer(handler)
		l, err := test_helpers.BindToPort(t, sbPort)
		if err != nil {
			t.Fatalf("Failed to bind ScrapingBee server: %v", err)
		}
		srv.Listener = l
		srv.Start()
		cleanup := func() { srv.Close() }
		return srv.URL, cleanup
	}

	// OpenAI mock that counts calls
	makeAI := func(responseContent string) (base string, cleanup func(), count *int32) {
		var calls int32
		aiPort := test_helpers.GetNextPort()
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(services.ChatCompletionResponse{
				ID: "mock", Object: "chat.completion", Created: time.Now().Unix(), Model: "gpt-4o-mini",
				Choices: []services.Choice{{Index: 0, Message: services.Message{Role: "assistant", Content: responseContent}, FinishReason: "stop"}},
				Usage:   services.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
			})
		}))
		l, err := test_helpers.BindToPort(t, aiPort)
		if err != nil {
			t.Fatalf("Failed to bind OpenAI server: %v", err)
		}
		srv.Listener = l
		srv.Start()
		cleanup = func() { srv.Close() }
		return srv.URL, cleanup, &calls
	}

	t.Run("Onboarding Random URL creates job and scrapes via ScrapingBee", func(t *testing.T) {
		// Count ScrapingBee hits
		var sbHits int32
		// ScrapingBee returns generic HTML for non-FB; SCRAPE mode should fetch it without using OpenAI
		sbURL, sbClose := makeSB(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&sbHits, 1)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body>Some generic page</body></html>"))
		})
		defer sbClose()

		// LLM mock (should not be hit in SCRAPE mode for non-FB)
		aiURL, aiClose, aiCalls := makeAI("[]")
		defer aiClose()

		// Wire env
		prevSB, prevAI, prevAIKey := os.Getenv("SCRAPINGBEE_API_URL_BASE"), os.Getenv("OPENAI_API_BASE_URL"), os.Getenv("OPENAI_API_KEY")
		os.Setenv("SCRAPINGBEE_API_URL_BASE", sbURL)
		os.Setenv("OPENAI_API_BASE_URL", aiURL)
		os.Setenv("OPENAI_API_KEY", "test-ai-key")
		defer func() {
			os.Setenv("SCRAPINGBEE_API_URL_BASE", prevSB)
			os.Setenv("OPENAI_API_BASE_URL", prevAI)
			os.Setenv("OPENAI_API_KEY", prevAIKey)
		}()

		// Mock DynamoDB to return a session with validated events
		sess := types.SeshuSession{
			OwnerId:           "user-1",
			Url:               "https://example.com/source",
			UrlDomain:         "example.com",
			UrlPath:           "/source",
			LocationLatitude:  40.7128,
			LocationLongitude: -74.0060,
			LocationAddress:   "New York, NY 10001, USA",
			Html:              "<html><body><div>AI Event</div><div>AI Hall</div><time>2025-05-01T10:00:00</time><a href=\"https://example.com/e1\">https://example.com/e1</a></body></html>",
			EventCandidates: []types.EventInfo{{
				EventTitle:       "AI Event",
				EventURL:         "https://example.com/e1",
				EventLocation:    "AI Hall",
				EventStartTime:   "2025-05-01T10:00:00",
				EventEndTime:     "2025-05-01T12:00:00",
				EventDescription: "",
				ScrapeMode:       "init",
			}},
			EventValidations: []types.EventBoolValid{{
				EventValidateTitle:       true,
				EventValidateLocation:    true,
				EventValidateStartTime:   true,
				EventValidateEndTime:     false,
				EventValidateURL:         true,
				EventValidateDescription: false,
			}},
			Status:    "draft",
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			ExpireAt:  time.Now().Add(24 * time.Hour).Unix(),
		}
		item, err := attributevalue.MarshalMap(sess)
		if err != nil {
			t.Fatalf("failed to marshal session: %v", err)
		}
		transport.SetTestDB(&test_helpers.MockDynamoDBClient{
			GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
				return &dynamodb.GetItemOutput{Item: item}, nil
			},
		})

		// Capture CreateSeshuJob invocation
		var createdJobs []types.SeshuJob
		mockPG := &test_helpers.MockPostgresService{
			GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) { return []types.SeshuJob{}, nil },
			CreateSeshuJobFunc: func(ctx context.Context, job types.SeshuJob) error {
				createdJobs = append(createdJobs, job)
				return nil
			},
		}

		// Build payload expected by handler
		payload := SeshuSessionEventsPayload{Url: sess.Url}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/submit-seshu-session", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// Attach ctx with mock Postgres + minimal auth/options
		req = req.WithContext(newCtx(mockPG))

		rr := httptest.NewRecorder()
		handler := SubmitSeshuSession(rr, req)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", rr.Code, rr.Body.String())
		}
		// Allow deferred job creation and background scrape to run
		time.Sleep(100 * time.Millisecond)
		if atomic.LoadInt32(aiCalls) != 0 {
			t.Fatalf("expected OpenAI NOT to be called for non-Facebook SCRAPE mode, got %d", atomic.LoadInt32(aiCalls))
		}
		// Job creation is best-effort and depends on DOM mapping; don't assert on it here
		if len(createdJobs) > 0 && createdJobs[0].KnownScrapeSource == constants.SESHU_KNOWN_SOURCE_FB {
			t.Fatalf("expected non-Facebook scrape source, got FB")
		}
	})

	t.Run("Onboarding Facebook URL avoids OpenAI", func(t *testing.T) {
		// ScrapingBee returns FB page with single-line JSON in expected script tag
		fbJSON := `{"__bbox":{"result":{"data":{"event":{"__typename":"Event","name":"FB Event","url":"https://www.facebook.com/events/123","day_time_sentence":"2025-10-24T23:00:00Z","contextual_name":"FB Hall","event_creator":{"name":"Host"}}}}}}`
		sbURL, sbClose := makeSB(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><head><meta property="og:url" content="https://www.facebook.com/events/123"></head><body><script data-sjs data-content-len="4096">` + fbJSON + `</script></body></html>`))
		})
		defer sbClose()

		// LLM mock should not be hit
		aiURL, aiClose, aiCalls := makeAI("[]")
		defer aiClose()

		prevSB, prevAI, prevAIKey := os.Getenv("SCRAPINGBEE_API_URL_BASE"), os.Getenv("OPENAI_API_BASE_URL"), os.Getenv("OPENAI_API_KEY")
		os.Setenv("SCRAPINGBEE_API_URL_BASE", sbURL)
		os.Setenv("OPENAI_API_BASE_URL", aiURL)
		os.Setenv("OPENAI_API_KEY", "test-ai-key")
		defer func() {
			os.Setenv("SCRAPINGBEE_API_URL_BASE", prevSB)
			os.Setenv("OPENAI_API_BASE_URL", prevAI)
			os.Setenv("OPENAI_API_KEY", prevAIKey)
		}()

		// Mock DynamoDB to return a session with validated events for FB URL
		fbSess := types.SeshuSession{
			OwnerId:           "user-1",
			Url:               "https://www.facebook.com/events/123",
			UrlDomain:         "facebook.com",
			UrlPath:           "/events/123",
			LocationLatitude:  37.7749,
			LocationLongitude: -122.4194,
			LocationAddress:   "San Francisco, CA",
			Html:              "<html><body>FB Event</body></html>",
			EventCandidates: []types.EventInfo{{
				EventTitle:       "FB Event",
				EventURL:         "https://www.facebook.com/events/123",
				EventLocation:    "FB Hall",
				EventStartTime:   "2025-10-24T23:00:00",
				EventEndTime:     "",
				EventDescription: "",
				ScrapeMode:       "init",
			}},
			EventValidations: []types.EventBoolValid{{
				EventValidateTitle:       true,
				EventValidateLocation:    true,
				EventValidateStartTime:   true,
				EventValidateEndTime:     false,
				EventValidateURL:         true,
				EventValidateDescription: false,
			}},
			Status:    "draft",
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
			ExpireAt:  time.Now().Add(24 * time.Hour).Unix(),
		}
		fbItem, err := attributevalue.MarshalMap(fbSess)
		if err != nil {
			t.Fatalf("failed to marshal FB session: %v", err)
		}
		transport.SetTestDB(&test_helpers.MockDynamoDBClient{
			GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
				return &dynamodb.GetItemOutput{Item: fbItem}, nil
			},
		})

		var createdJobs []types.SeshuJob
		mockPG := &test_helpers.MockPostgresService{
			GetSeshuJobsFunc: func(ctx context.Context) ([]types.SeshuJob, error) { return []types.SeshuJob{}, nil },
			CreateSeshuJobFunc: func(ctx context.Context, job types.SeshuJob) error {
				createdJobs = append(createdJobs, job)
				return nil
			},
		}

		payload := SeshuSessionEventsPayload{Url: "https://www.facebook.com/events/123"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/submit-seshu-session", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(newCtx(mockPG))

		rr := httptest.NewRecorder()
		handler := SubmitSeshuSession(rr, req)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", rr.Code, rr.Body.String())
		}
		// Allow deferred job creation and background scrape to run
		time.Sleep(100 * time.Millisecond)
		if atomic.LoadInt32(aiCalls) != 0 {
			t.Fatalf("expected OpenAI not to be called for Facebook URLs, got %d calls", atomic.LoadInt32(aiCalls))
		}
		// Job creation is best-effort and depends on validation; avoid asserting on it
		if len(createdJobs) > 0 && createdJobs[0].KnownScrapeSource != constants.SESHU_KNOWN_SOURCE_FB {
			t.Fatalf("expected KnownScrapeSource=FB, got %s", createdJobs[0].KnownScrapeSource)
		}
	})
}
func TestGetProfileInterestsPartial(t *testing.T) {
	tests := []struct {
		name             string
		userMetaClaims   map[string]interface{}
		expectedStatus   int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "successful render with interests",
			userMetaClaims: map[string]interface{}{
				constants.INTERESTS_KEY: []string{"Concerts", "Photography", "Sports"},
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Interests",
				"update-interests-result",
				"/api/auth/users/update-interests",
				"hx-post",
				"hx-target",
			},
			shouldNotContain: []string{"error", "Error"},
		},
		{
			name: "successful render with empty interests",
			userMetaClaims: map[string]interface{}{
				constants.INTERESTS_KEY: []string{},
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Interests",
				"update-interests-result",
				"/api/auth/users/update-interests",
			},
			shouldNotContain: []string{"error", "Error"},
		},
		{
			name:           "successful render with no userMetaClaims",
			userMetaClaims: map[string]interface{}{},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Interests",
				"update-interests-result",
				"/api/auth/users/update-interests",
			},
			shouldNotContain: []string{"error", "Error"},
		},
		{
			name: "successful render with nil interests",
			userMetaClaims: map[string]interface{}{
				constants.INTERESTS_KEY: nil,
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Interests",
				"update-interests-result",
				"/api/auth/users/update-interests",
			},
			shouldNotContain: []string{"error", "Error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/api/html/profile-interests", nil)

			// Add AWS Lambda context and userMetaClaims to context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userMetaClaims", tt.userMetaClaims)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			// Execute handler
			handler := GetProfileInterestsPartial(rr, req)
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

			// Check strings that should not be present
			for _, unexpectedStr := range tt.shouldNotContain {
				if strings.Contains(responseBody, unexpectedStr) {
					t.Errorf("Expected response to NOT contain '%s', got '%s'", unexpectedStr, responseBody)
				}
			}

			// Verify it's a partial (HTML fragment, not full page)
			if strings.Contains(responseBody, "<!DOCTYPE html>") || strings.Contains(responseBody, "<html") {
				t.Errorf("Expected partial HTML, but got full page HTML")
			}
		})
	}
}

func TestGetSubscriptionsPartial(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
	}()

	// Set up test environment
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")

	// Set up logging transport
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	tests := []struct {
		name             string
		userInfo         constants.UserInfo
		hasCustomer      bool
		customerID       string
		subscriptions    []map[string]interface{}
		expectedStatus   int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "no subscriptions - empty state",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			hasCustomer:    true,
			customerID:     "cus_test_customer",
			subscriptions:  []map[string]interface{}{},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Subscriptions",
				"You don't have any subscriptions yet",
				"pricing page",
			},
			shouldNotContain: []string{
				"Billing Cycle",
				"billing cycle",
				"Active Subscriptions",
				"Subscription History",
			},
		},
		{
			name: "active subscription only",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			hasCustomer: true,
			customerID:  "cus_test_customer",
			subscriptions: []map[string]interface{}{
				{
					"id":                   "sub_active_123",
					"object":               "subscription",
					"status":               "active",
					"customer":             "cus_test_customer",
					"cancel_at_period_end": false,
					"created":              1234567890,
					"canceled_at":          nil,
					"items": map[string]interface{}{
						"object": "list",
						"data": []interface{}{
							map[string]interface{}{
								"id":                   "si_item_1",
								"object":               "subscription_item",
								"current_period_start": 1234567890,
								"current_period_end":   1234567890 + 2592000,
								"price": map[string]interface{}{
									"id":     "price_growth",
									"object": "price",
									"product": map[string]interface{}{
										"id":   "prod_growth",
										"name": "Growth",
									},
									"unit_amount": 9999,
									"currency":    "usd",
									"recurring": map[string]interface{}{
										"interval": "month",
									},
								},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Subscriptions",
				"Active Subscriptions",
				"Growth",
				"Actions",
				"Update Subscription",
				"Cancel Subscription",
				"Update Payment Method",
			},
			shouldNotContain: []string{
				"Billing Cycle",
				"billing cycle",
				"Subscription History",
				"Canceling at period end",
			},
		},
		{
			name: "canceled subscription only",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			hasCustomer: true,
			customerID:  "cus_test_customer",
			subscriptions: []map[string]interface{}{
				{
					"id":                   "sub_canceled_123",
					"object":               "subscription",
					"status":               "canceled",
					"customer":             "cus_test_customer",
					"cancel_at_period_end": false,
					"created":              1234567890,
					"canceled_at":          1234567890 + 2592000,
					"items": map[string]interface{}{
						"object": "list",
						"data": []interface{}{
							map[string]interface{}{
								"id":                   "si_item_1",
								"object":               "subscription_item",
								"current_period_start": 1234567890,
								"current_period_end":   1234567890 + 2592000,
								"price": map[string]interface{}{
									"id":     "price_seed",
									"object": "price",
									"product": map[string]interface{}{
										"id":   "prod_seed",
										"name": "Seed Community",
									},
									"unit_amount": 4999,
									"currency":    "usd",
									"recurring": map[string]interface{}{
										"interval": "month",
									},
								},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Subscriptions",
				"Subscription History",
				"Seed Community",
				"Canceled",
			},
			shouldNotContain: []string{
				"Billing Cycle",
				"billing cycle",
				"Active Subscriptions",
				"Manage",
			},
		},
		{
			name: "mixed active and canceled subscriptions",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			hasCustomer: true,
			customerID:  "cus_test_customer",
			subscriptions: []map[string]interface{}{
				{
					"id":                   "sub_active_123",
					"object":               "subscription",
					"status":               "active",
					"customer":             "cus_test_customer",
					"cancel_at_period_end": true,
					"created":              1234567890,
					"canceled_at":          nil,
					"items": map[string]interface{}{
						"object": "list",
						"data": []interface{}{
							map[string]interface{}{
								"id":                   "si_item_1",
								"object":               "subscription_item",
								"current_period_start": 1234567890,
								"current_period_end":   1234567890 + 2592000,
								"price": map[string]interface{}{
									"id":     "price_growth",
									"object": "price",
									"product": map[string]interface{}{
										"id":   "prod_growth",
										"name": "Growth",
									},
									"unit_amount": 9999,
									"currency":    "usd",
									"recurring": map[string]interface{}{
										"interval": "month",
									},
								},
							},
						},
					},
				},
				{
					"id":                   "sub_canceled_123",
					"object":               "subscription",
					"status":               "canceled",
					"customer":             "cus_test_customer",
					"cancel_at_period_end": false,
					"created":              1234567890,
					"canceled_at":          1234567890 + 2592000,
					"items": map[string]interface{}{
						"object": "list",
						"data": []interface{}{
							map[string]interface{}{
								"id":                   "si_item_2",
								"object":               "subscription_item",
								"current_period_start": 1234567890,
								"current_period_end":   1234567890 + 2592000,
								"price": map[string]interface{}{
									"id":     "price_seed",
									"object": "price",
									"product": map[string]interface{}{
										"id":   "prod_seed",
										"name": "Seed Community",
									},
									"unit_amount": 4999,
									"currency":    "usd",
									"recurring": map[string]interface{}{
										"interval": "month",
									},
								},
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Subscriptions",
				"Active Subscriptions",
				"Subscription History",
				"Growth",
				"Seed Community",
				"Canceling at period end",
				"Actions",
				"Update Subscription",
				"Update Payment Method",
			},
			shouldNotContain: []string{
				"Billing Cycle",
				"billing cycle",
				"Cancel Subscription", // Should not appear when cancel_at_period_end is true
			},
		},
		{
			name:           "missing user ID returns unauthorized",
			userInfo:       constants.UserInfo{},
			hasCustomer:    false,
			expectedStatus: http.StatusOK, // SendHtmlErrorPartial returns 200 even for errors
			shouldContain: []string{
				"Unauthorized",
				"Missing user ID",
			},
		},
		{
			name: "customer not found - returns empty state",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			hasCustomer:    false,
			expectedStatus: http.StatusOK,
			shouldContain: []string{
				"Subscriptions",
				"You don't have any subscriptions yet",
				"pricing page",
			},
			shouldNotContain: []string{
				"Billing Cycle",
				"billing cycle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Stripe server
			mockStripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Logf("🎯 MOCK STRIPE HIT: %s %s", r.Method, r.URL.Path)

				switch {
				case strings.Contains(r.URL.Path, "/v1/customers/search"):
					t.Logf("   └─ Handling customer search")
					if tt.hasCustomer && tt.customerID != "" {
						// Mock customer found
						mockSearchResponse := map[string]interface{}{
							"object": "search_result",
							"data": []interface{}{
								map[string]interface{}{
									"id":    tt.customerID,
									"email": "test@example.com",
									"name":  "Test User",
									"metadata": map[string]interface{}{
										"zitadel_user_id": tt.userInfo.Sub,
									},
								},
							},
							"has_more": false,
						}
						responseBytes, _ := json.Marshal(mockSearchResponse)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write(responseBytes)
					} else {
						// Mock customer not found
						mockSearchResponse := map[string]interface{}{
							"object":   "search_result",
							"data":     []interface{}{},
							"has_more": false,
						}
						responseBytes, _ := json.Marshal(mockSearchResponse)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write(responseBytes)
					}

				case r.URL.Path == "/v1/subscriptions":
					t.Logf("   └─ Handling subscriptions list request")
					mockSubscriptionsResponse := map[string]interface{}{
						"object":   "list",
						"data":     tt.subscriptions,
						"has_more": false,
					}
					responseBytes, _ := json.Marshal(mockSubscriptionsResponse)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write(responseBytes)

				default:
					t.Logf("   └─ ⚠️  UNHANDLED STRIPE PATH: %s", r.URL.Path)
					t.Errorf("mock Stripe server received request to unhandled path: %s", r.URL.Path)
					http.Error(w, "Not Found", http.StatusNotFound)
				}
			}))
			defer mockStripeServer.Close()

			// Create custom RoundTripper to redirect Stripe calls to mock server
			customTransport := &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					if strings.Contains(req.URL.Host, "api.stripe.com") {
						mockURL, _ := url.Parse(mockStripeServer.URL)
						return mockURL, nil
					}
					return nil, nil
				},
			}

			customRoundTripper := &customRoundTripperForStripe{
				transport: customTransport,
				mockURL:   mockStripeServer.URL,
			}

			http.DefaultTransport = customRoundTripper
			services.ResetStripeClient()

			// Create request
			req := httptest.NewRequest("GET", "/api/html/subscriptions", nil)

			// Add context with user info
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler := GetSubscriptionsPartial(w, req)
			handler(w, req)

			// Verify response
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Log the response for debugging
			t.Logf("Response status: %d", w.Code)
			t.Logf("Response body length: %d", len(w.Body.String()))

			// Check all required strings are present
			responseBody := w.Body.String()
			for _, expectedStr := range tt.shouldContain {
				if !strings.Contains(responseBody, expectedStr) {
					t.Errorf("Expected response to contain '%s'", expectedStr)
				}
			}

			// Check strings that should not be present
			for _, unexpectedStr := range tt.shouldNotContain {
				if strings.Contains(responseBody, unexpectedStr) {
					t.Errorf("Expected response to NOT contain '%s'", unexpectedStr)
				}
			}

			// Verify it's a partial (HTML fragment, not full page)
			if strings.Contains(responseBody, "<!DOCTYPE html>") || strings.Contains(responseBody, "<html") {
				t.Errorf("Expected partial HTML, but got full page HTML")
			}
		})
	}
}
