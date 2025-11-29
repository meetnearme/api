package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/playwright-community/playwright-go"
	"github.com/weaviate/weaviate/entities/models"
)

func TestGetHomeOrUserPage(t *testing.T) {
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

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	// Create mock Weaviate server (following established pattern)
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql (home page event search)")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Return events for the home page
			// NOTE: This mock should ONLY return published event types (SLF, SLF_EVS)
			// The filter logic should already exclude unpublished types (SLF_UNPUB, etc.)
			// so they shouldn't even be in the Weaviate query results
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"name":            "First Test Event",
								"description":     "Description of the first event",
								"eventOwners":     []interface{}{"789"},
								"eventOwnerName":  "First Event Host",
								"eventSourceType": "SLF", // Published single event
								"startTime":       time.Now().Add(48 * time.Hour).Unix(),
								"endTime":         time.Now().Add(50 * time.Hour).Unix(),
								"address":         "123 First St",
								"lat":             40.7128,
								"long":            -74.0060,
								"timezone":        "America/New_York",
								"_additional": map[string]interface{}{
									"id": "123",
								},
							},
							map[string]interface{}{
								"name":            "Second Test Event",
								"description":     "Description of the second event",
								"eventOwners":     []interface{}{"012"},
								"eventOwnerName":  "Second Event Host",
								"eventSourceType": "SLF_EVS", // Published series parent
								"startTime":       time.Now().Add(72 * time.Hour).Unix(),
								"endTime":         time.Now().Add(74 * time.Hour).Unix(),
								"address":         "456 Second St",
								"lat":             34.0522,
								"long":            -118.2437,
								"timezone":        "America/New_York",
								"_additional": map[string]interface{}{
									"id": "456",
								},
							},
						},
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
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 HOME PAGE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

	// Add MNM_OPTIONS_CTX_KEY to context
	fakeContext := context.Background()
	// fakeContext = context.WithValue(fakeContext, constants.MNM_OPTIONS_CTX_KEY, map[string]string{})

	// Create a request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(fakeContext)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := GetHomeOrUserPage(rr, req)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body (you might want to add more specific checks)
	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}

	if !strings.Contains(rr.Body.String(), ">First Test Event") {
		t.Errorf("First event title is missing from the page")
	}

	if !strings.Contains(rr.Body.String(), ">Second Test Event") {
		t.Errorf("Second event title is missing from the page")
	}

	// Verify that unpublished event types are NOT present
	// Since our filter uses "field" tokenization and ContainsAny,
	// it should only match exact values: "SLF" and "SLF_EVS"
	// Events with type "SLF_UNPUB" should not appear
	if strings.Contains(rr.Body.String(), "data-event-type=\"SLF_UNPUB\"") {
		t.Errorf("Unpublished event (SLF_UNPUB) should not appear on home page")
	}
	if strings.Contains(rr.Body.String(), "data-event-type=\"SLF_EVS_UNPUB\"") {
		t.Errorf("Unpublished series event (SLF_EVS_UNPUB) should not appear on home page")
	}
}

func TestGetHomeOrUserPage_SubdomainLogic_NilPointerSafety(t *testing.T) {
	// This test specifically ensures that the code doesn't panic when
	// mnmOptions context is missing (which would cause GetMnmOptionsFromContext
	// to return an empty map, not nil, but we want to test the edge case)
	t.Run("Subdomain without mnmOptions in context should not panic", func(t *testing.T) {
		// Create a request with subdomain but NO context set at all
		// This simulates a request that bypasses the middleware
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "subdomain.example.com"
		// Explicitly do NOT set X-Mnm-Options header
		// And do NOT set MNM_OPTIONS_CTX_KEY in context

		// Create context WITHOUT MNM_OPTIONS_CTX_KEY
		// This is the edge case that could cause issues
		fakeContext := context.Background()
		req = req.WithContext(fakeContext)

		// Create a ResponseRecorder
		rr := httptest.NewRecorder()

		// This should NOT panic, even if mnmOptions handling is incorrect
		// The function should handle the case where context doesn't have mnmOptions
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Handler panicked with nil pointer: %v", r)
			}
		}()

		// Call the handler - this should not panic
		handler := GetHomeOrUserPage(rr, req)
		handler.ServeHTTP(rr, req)

		// Verify we got a response (not a panic)
		if rr.Code == 0 {
			t.Error("Handler did not write a response (may have panicked)")
		}

		// When mnmOptions is empty (from GetMnmOptionsFromContext returning empty map),
		// and we have a subdomain, we should show the error page
		body := rr.Body.String()
		if !strings.Contains(body, "User Not Found") {
			t.Error("Expected error page when subdomain exists and mnmOptions is empty")
		}
	})

	t.Run("Subdomain with nil mnmOptions map in context should not panic", func(t *testing.T) {
		// Test the edge case where context has MNM_OPTIONS_CTX_KEY but value is nil
		// This shouldn't happen with GetMnmOptionsFromContext, but let's be defensive
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "subdomain.example.com"

		// Create context with nil value (edge case)
		fakeContext := context.WithValue(context.Background(), constants.MNM_OPTIONS_CTX_KEY, nil)
		req = req.WithContext(fakeContext)

		rr := httptest.NewRecorder()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Handler panicked with nil pointer: %v", r)
			}
		}()

		// This should not panic - GetMnmOptionsFromContext should return empty map, not nil
		handler := GetHomeOrUserPage(rr, req)
		handler.ServeHTTP(rr, req)

		if rr.Code == 0 {
			t.Error("Handler did not write a response (may have panicked)")
		}
	})

	t.Run("Subdomain with empty mnmOptions map should show error page", func(t *testing.T) {
		// This test ensures that when mnmOptions is empty (not nil, but empty map),
		// we correctly show the error page. This catches the bug where checking
		// mnmOptions != nil would always be true (since GetMnmOptionsFromContext
		// returns empty map, not nil), causing incorrect behavior.
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "subdomain.example.com"

		// Create context with empty map (what GetMnmOptionsFromContext returns when not found)
		fakeContext := context.WithValue(context.Background(), constants.MNM_OPTIONS_CTX_KEY, map[string]string{})
		req = req.WithContext(fakeContext)

		rr := httptest.NewRecorder()

		handler := GetHomeOrUserPage(rr, req)
		handler.ServeHTTP(rr, req)

		// Should show error page when mnmOptions is empty
		body := rr.Body.String()
		if !strings.Contains(body, "User Not Found") {
			t.Error("Expected error page when subdomain exists and mnmOptions is empty map")
		}
	})

	t.Run("Subdomain with populated mnmOptions should proceed normally", func(t *testing.T) {
		// This test ensures that when mnmOptions has values, we don't show the error page
		// Note: This test only checks the early return logic, not the full page rendering
		// which would require Weaviate setup
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "subdomain.example.com"

		// Create context with populated mnmOptions
		fakeContext := context.WithValue(context.Background(), constants.MNM_OPTIONS_CTX_KEY, map[string]string{
			"userId": "123",
		})
		req = req.WithContext(fakeContext)

		rr := httptest.NewRecorder()

		handler := GetHomeOrUserPage(rr, req)
		handler.ServeHTTP(rr, req)

		// Should NOT show the subdomain error page when mnmOptions has values
		// (may show different error from DeriveEventsFromRequest, but not the subdomain claim error)
		body := rr.Body.String()
		if strings.Contains(body, "claim this subdomain") {
			t.Error("Expected NOT to show subdomain claim error when mnmOptions has values")
		}
	})
}

func TestGetHomeOrUserPage_SubdomainLogic(t *testing.T) {
	tests := []struct {
		name                string
		host                string
		hostHeader          string // Optional Host header (for worker proxy scenarios)
		mnmOptionsHeader    string
		isLocalAct          string // IS_LOCAL_ACT environment variable value (empty means don't set it)
		expectedErrorPage   bool
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:              "Subdomain without X-Mnm-Options header should show error page",
			host:              "subdomain.example.com",
			mnmOptionsHeader:  "",
			expectedErrorPage: true,
			expectedContains: []string{
				"User Not Found",
				"claim this subdomain",
				`<a class="link link-text" href="/admin">`, // HTML should be rendered, not escaped
			},
			expectedNotContains: []string{
				"&lt;a", // HTML should not be escaped
				"&lt;br",
			},
		},
		{
			name:              "Subdomain with X-Mnm-Options header should proceed normally",
			host:              "subdomain.example.com",
			mnmOptionsHeader:  "userId=123",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name:              "Subdomain with quoted X-Mnm-Options header should proceed normally",
			host:              "subdomain.example.com",
			mnmOptionsHeader:  `"userId=123"`,
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name:              "Apex domain (example.com has 2 parts) should proceed normally without header",
			host:              "example.com",
			mnmOptionsHeader:  "",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name:              "Apex domain with X-Mnm-Options header should proceed normally",
			host:              "example.com",
			mnmOptionsHeader:  "userId=123",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name: "127.0.0.1 with subdomain.localhost Host header and IS_LOCAL_ACT=true and no mnmOptions should show error page",
			// When worker forwards to 127.0.0.1:8000, r.Host will always be "127.0.0.1:8000" (or "127.0.0.1")
			// regardless of what the Host header is set to. The proxy ensures r.Host reflects the connection target.
			// The condition checks: IS_LOCAL_ACT=true && r.Host contains "127.0.0.1" && Host header has subdomain && no mnmOptions
			// The Host header "test.localhost" has 2 parts, and the logic checks if first part is not "localhost",
			// so "test.localhost" will be detected as a subdomain and show the error page.
			host:              "127.0.0.1:8000",
			hostHeader:        "test.localhost", // Host header set by local dev worker with subdomain
			mnmOptionsHeader:  "",
			isLocalAct:        "true",
			expectedErrorPage: true, // Should show error page when subdomain in Host header and no mnmOptions
			expectedContains: []string{
				"User Not Found",
				"claim this subdomain",
				`<a class="link link-text" href="/admin">`,
			},
			expectedNotContains: []string{
				"&lt;a",
				"&lt;br",
			},
		},
		{
			name: "127.0.0.1 with subdomain.localhost Host header and IS_LOCAL_ACT=true and mnmOptions should proceed normally",
			// When proxied, r.Host is always 127.0.0.1:8000, Host header is separate
			host:              "127.0.0.1:8000",
			hostHeader:        "subdomain.localhost:8000", // Host header set by local dev worker (but r.Host will still be 127.0.0.1:8000)
			mnmOptionsHeader:  "userId=123",
			isLocalAct:        "true",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name: "127.0.0.1 with localhost Host header (no subdomain) and IS_LOCAL_ACT=true should proceed normally",
			// When proxied, r.Host is always 127.0.0.1:8000
			host:              "127.0.0.1:8000",
			hostHeader:        "localhost:8000", // Host header set by local dev worker (but r.Host will still be 127.0.0.1:8000)
			mnmOptionsHeader:  "",
			isLocalAct:        "true",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
		{
			name:              "127.0.0.1 with subdomain.localhost Host header and IS_LOCAL_ACT=false should show error page (IP treated as subdomain)",
			host:              "127.0.0.1:8000",
			hostHeader:        "subdomain.localhost:8000", // Host header set by worker (but r.Host will still be 127.0.0.1:8000)
			mnmOptionsHeader:  "",
			isLocalAct:        "false",
			expectedErrorPage: true, // Current logic treats IP addresses as subdomains
			expectedContains: []string{
				"User Not Found",
				"claim this subdomain",
				`<a class="link link-text" href="/admin">`,
			},
			expectedNotContains: []string{
				"&lt;a",
				"&lt;br",
			},
		},
		{
			name: "127.0.0.1:8000 directly (no Host header manipulation) with IS_LOCAL_ACT=true should proceed normally",
			// Direct connection to 127.0.0.1:8000, r.Host will be 127.0.0.1:8000
			host:              "127.0.0.1:8000",
			mnmOptionsHeader:  "",
			isLocalAct:        "true",
			expectedErrorPage: false,
			expectedNotContains: []string{
				"User Not Found",
				"claim this subdomain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore IS_LOCAL_ACT environment variable
			originalIsLocalAct := os.Getenv("IS_LOCAL_ACT")
			defer func() {
				if originalIsLocalAct == "" {
					os.Unsetenv("IS_LOCAL_ACT")
				} else {
					os.Setenv("IS_LOCAL_ACT", originalIsLocalAct)
				}
			}()

			// Set IS_LOCAL_ACT if specified in test case
			if tt.isLocalAct != "" {
				os.Setenv("IS_LOCAL_ACT", tt.isLocalAct)
			} else {
				os.Unsetenv("IS_LOCAL_ACT")
			}

			// Create a request with the specified host
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = tt.host

			// Set X-Original-Host header if provided (for worker proxy scenarios)
			// The worker sets X-Original-Host with the subdomain (e.g., "test.localhost")
			// to preserve it when proxying to 127.0.0.1:8000 where r.Host is always "127.0.0.1:8000"
			if tt.hostHeader != "" {
				req.Header.Set("X-Original-Host", tt.hostHeader)
				// Ensure req.Host matches the proxy behavior where r.Host is always the connection target
				req.Host = tt.host
			}

			// Set X-Mnm-Options header if provided
			if tt.mnmOptionsHeader != "" {
				req.Header.Set("X-Mnm-Options", tt.mnmOptionsHeader)
			}

			// Add context with mnmOptions populated from header using the shared helper function
			// This ensures the test uses the exact same parsing logic as the middleware
			fakeContext := context.Background()
			mnmOptions := helpers.ParseMnmOptionsHeader(tt.mnmOptionsHeader)
			fakeContext = context.WithValue(fakeContext, constants.MNM_OPTIONS_CTX_KEY, mnmOptions)
			req = req.WithContext(fakeContext)

			// Create a ResponseRecorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler := GetHomeOrUserPage(rr, req)
			handler.ServeHTTP(rr, req)

			body := rr.Body.String()

			if tt.expectedErrorPage {
				// Verify status is OK (SendHtmlErrorPage returns 200)
				if rr.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
				}

				// Verify expected content is present
				for _, expected := range tt.expectedContains {
					if !strings.Contains(body, expected) {
						t.Errorf("Expected response to contain '%s', but it didn't", expected)
					}
				}

				// Verify HTML is rendered (not escaped)
				for _, notExpected := range tt.expectedNotContains {
					if strings.Contains(body, notExpected) {
						t.Errorf("Expected response to NOT contain escaped HTML '%s', but it did", notExpected)
					}
				}

				// Verify the link is clickable (HTML is rendered)
				if !strings.Contains(body, `<a class="link link-text" href="/admin">`) {
					t.Error("Expected HTML link to be rendered, but it appears to be escaped or missing")
				}
			} else {
				// For non-error cases, verify the subdomain-specific error page content is NOT present
				// (may have other errors from DeriveEventsFromRequest, but not the subdomain claim error)
				for _, notExpected := range tt.expectedNotContains {
					if strings.Contains(body, notExpected) {
						// Only fail if it's the subdomain-specific error message
						if notExpected == "claim this subdomain" {
							t.Errorf("Expected response to NOT contain subdomain claim error '%s', but it did", notExpected)
						}
						// For "User Not Found", check if it's the subdomain version (with claim link)
						// vs the generic version (without claim link)
						if notExpected == "User Not Found" && strings.Contains(body, "claim this subdomain") {
							t.Errorf("Expected response to NOT contain subdomain error '%s', but it did", notExpected)
						}
					}
				}
			}
		})
	}
}

func TestGetHomeOrUserPage_NoDuplicateAPICalls(t *testing.T) {
	// This test ensures that clicking "Apply Filters" doesn't cause duplicate /api/html/events requests
	// The regression: if handleFilterSubmit calls setParam() multiple times instead of setParams() once,
	// each setParam() call triggers a form submission, causing duplicate API calls

	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalIsLocalAct := os.Getenv("IS_LOCAL_ACT")
	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		os.Setenv("IS_LOCAL_ACT", originalIsLocalAct)
	}()

	// Set IS_LOCAL_ACT for proxy scenario testing
	os.Setenv("IS_LOCAL_ACT", "true")

	// Set up mock Weaviate server
	hostAndPort := test_helpers.GetNextPort()
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/meta":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version":"1.0"}`))
		case "/v1/graphql":
			// Mock response for home page search
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{},
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
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to point to mock server
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")

	// Set up router
	router := test_helpers.SetupStaticTestRouter(t, "./assets")
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Always parse mnmOptions header and add to context (even if empty)
		mnmOptions := helpers.ParseMnmOptionsHeader("")
		ctx := context.WithValue(r.Context(), constants.MNM_OPTIONS_CTX_KEY, mnmOptions)
		r = r.WithContext(ctx)
		GetHomeOrUserPage(w, r).ServeHTTP(w, r)
	})

	// Create test server
	testServerPort := test_helpers.GetNextPort()
	testServer := httptest.NewUnstartedServer(router)
	testServerListener, err := test_helpers.BindToPort(t, testServerPort)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	testServer.Listener = testServerListener
	testServer.Start()
	defer testServer.Close()

	// Set up Playwright
	browser, err := test_helpers.GetPlaywrightBrowser()
	if err != nil {
		t.Skipf("skipping playwright flow: %v", err)
	}
	if browser == nil {
		t.Skip("skipping playwright flow: browser unavailable")
	}
	page, err := (*browser).NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	// Set up HTTP request listener BEFORE navigation to catch all requests
	requestCount := 0
	page.OnRequest(func(request playwright.Request) {
		if strings.Contains(request.URL(), "/api/html/events") {
			requestCount++
			t.Logf("🌐 HTTP Request #%d to /api/html/events at %s", requestCount, time.Now().Format("15:04:05.000"))
		}
	})

	// Navigate to the page
	fullURL := fmt.Sprintf("%s/", testServer.URL)
	t.Logf("Navigating to: %s", fullURL)
	if _, err = page.Goto(fullURL); err != nil {
		t.Fatalf("could not goto: %v", err)
	}

	// Wait a bit for initial page load and any async requests to complete
	time.Sleep(100 * time.Millisecond)

	// Record initial request count (should be 1 from initial page load)
	initialRequestCount := requestCount
	t.Logf("Initial HTTP requests to /api/html/events after page load: %d", initialRequestCount)

	// Open the drawer/sidebar if it's not already open (click the hamburger menu)
	drawerToggle := page.Locator("#main-drawer")
	if checked, _ := drawerToggle.IsChecked(); !checked {
		// Use the "open sidebar" label specifically (not the overlay close button)
		menuButton := page.Locator("label[aria-label='open sidebar']")
		if err := menuButton.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(300),
		}); err != nil {
			t.Logf("Note: Could not open drawer (might already be open): %v", err)
		} else {
			t.Logf("Opened drawer")
			// No sleep needed - Categories wait will handle timing
		}
	}

	// Ensure filters tab is selected (it should be by default, but just in case)
	filtersTab := page.Locator("#flyout-tab-filters")
	if visible, _ := filtersTab.IsVisible(); visible {
		if err := filtersTab.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(500),
		}); err != nil {
			t.Logf("Note: Could not click filters tab (might already be selected): %v", err)
		}
	}

	// Wait for Categories section to be visible (confirms filters are loaded)
	// This needs a bit more time as it depends on drawer animation and content rendering
	categoriesHeading := page.Locator("h3:has-text('Categories')")
	if err := categoriesHeading.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(1000),
	}); err != nil {
		t.Fatalf("Categories section not visible: %v", err)
	}
	t.Logf("Categories section is visible")

	// Click a category checkbox - find by name attribute pattern (first category checkbox)
	// Category checkboxes have name like "itm-0-category", "itm-1-category", etc.
	checkboxLocator := page.Locator("input[type='checkbox'][name^='itm-'][name$='-category']").First()
	if err := checkboxLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(500),
	}); err != nil {
		t.Fatalf("could not click category checkbox: %v", err)
	}
	t.Logf("Clicked category checkbox")

	// Also change the radius to trigger a second setParam() call
	// This will cause the bug to manifest: handleFilterSubmit will call setParam() twice
	// (once for categories, once for radius), causing duplicate form submissions
	radiusSelect := page.Locator("select[x-model='radius']")
	_, err = radiusSelect.SelectOption(playwright.SelectOptionValues{
		Values: &[]string{"25"},
	}, playwright.LocatorSelectOptionOptions{
		Timeout: playwright.Float(500),
	})
	if err != nil {
		t.Logf("Note: Could not change radius (might not be necessary): %v", err)
	} else {
		t.Logf("Changed radius to 25 mi")
	}

	// Click the "Apply Filters" button
	applyFiltersLocator := page.Locator("button:has-text('Apply Filters')")
	if err := applyFiltersLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(500),
	}); err != nil {
		t.Fatalf("could not click Apply Filters button: %v", err)
	}
	t.Logf("Clicked Apply Filters button at %s", time.Now().Format("15:04:05.000"))

	// Wait to catch any duplicate API calls that might be triggered
	// The bug causes multiple form submissions. Each setParam() has a 50ms setTimeout,
	// and duplicates fire within ~100ms. We'll wait 200ms total (with 50ms intervals)
	// to catch duplicates while keeping the test fast when the bug is fixed.
	requestCountBeforeWait := requestCount
	t.Logf("HTTP request count before wait: %d", requestCountBeforeWait)

	// Poll every 50ms for up to 200ms total (4 iterations)
	// This is sufficient to catch duplicates (which fire within ~100ms) while
	// keeping the test fast when the bug is fixed and no duplicates occur
	duplicateDetected := false
	maxIterations := 4 // 4 * 50ms = 200ms max wait
	for i := 0; i < maxIterations; i++ {
		time.Sleep(50 * time.Millisecond)
		currentRequestCount := requestCount

		// Check if we've detected duplicates
		if currentRequestCount > requestCountBeforeWait+1 {
			t.Logf("⚠️  DUPLICATE HTTP REQUEST DETECTED after %dms! Total requests: %d", (i+1)*50, currentRequestCount)
			duplicateDetected = true
		}

		// If we detected duplicates, we can exit early
		if duplicateDetected {
			t.Logf("Exiting early after %dms - duplicate detected", (i+1)*50)
			break
		}

		// Log progress for first few iterations
		if i < 3 && currentRequestCount > requestCountBeforeWait {
			t.Logf("After %dms: HTTP request count: %d", (i+1)*50, currentRequestCount)
		}
	}

	// Final snapshot
	finalRequestCount := requestCount
	requestsAfterFilter := finalRequestCount - requestCountBeforeWait
	t.Logf("Final HTTP request count: %d (initial: %d, after filter: %d)", finalRequestCount, initialRequestCount, requestsAfterFilter)

	// Verify that clicking "Apply Filters" triggers exactly 1 request to /api/html/events
	// If we see more than 1, it indicates duplicate form submissions
	expectedRequestsAfterFilter := 1
	if requestsAfterFilter != expectedRequestsAfterFilter {
		t.Errorf("Expected exactly %d HTTP request(s) to /api/html/events after clicking Apply Filters, but got %d. This indicates duplicate form submissions (the bug is present - handleFilterSubmit is calling setParam() multiple times instead of setParams() once).",
			expectedRequestsAfterFilter, requestsAfterFilter)
	}

	if requestsAfterFilter == expectedRequestsAfterFilter {
		t.Logf("✅ Clicking Apply Filters resulted in exactly %d HTTP request(s) to /api/html/events (as expected)", requestsAfterFilter)
	}
}

func TestGetHomeOrUserPage_WithGroupedEvents(t *testing.T) {
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

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup
	hostAndPort := test_helpers.GetNextPort()

	// Create mock Weaviate server with grouped events
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql (home page with grouped events)")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Return grouped events - same name, same location, different dates
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"name":            "Weekly Meetup",
								"description":     "A weekly meetup event",
								"eventOwners":     []interface{}{"789"},
								"eventOwnerName":  "Event Host",
								"eventSourceType": "SLF",
								"startTime":       time.Now().Add(48 * time.Hour).Unix(),
								"endTime":         time.Now().Add(50 * time.Hour).Unix(),
								"address":         "123 Main St",
								"lat":             40.7128,
								"long":            -74.0060,
								"timezone":        "America/New_York",
								"_additional": map[string]interface{}{
									"id": "event-1",
								},
							},
							map[string]interface{}{
								"name":            "Weekly Meetup",
								"description":     "A weekly meetup event",
								"eventOwners":     []interface{}{"789"},
								"eventOwnerName":  "Event Host",
								"eventSourceType": "SLF",
								"startTime":       time.Now().Add(120 * time.Hour).Unix(), // 5 days later
								"endTime":         time.Now().Add(122 * time.Hour).Unix(),
								"address":         "123 Main St",
								"lat":             40.7128,
								"long":            -74.0060,
								"timezone":        "America/New_York",
								"_additional": map[string]interface{}{
									"id": "event-2",
								},
							},
							map[string]interface{}{
								"name":            "Weekly Meetup",
								"description":     "A weekly meetup event",
								"eventOwners":     []interface{}{"789"},
								"eventOwnerName":  "Event Host",
								"eventSourceType": "SLF",
								"startTime":       time.Now().Add(192 * time.Hour).Unix(), // 8 days later
								"endTime":         time.Now().Add(194 * time.Hour).Unix(),
								"address":         "123 Main St",
								"lat":             40.7128,
								"long":            -74.0060,
								"timezone":        "America/New_York",
								"_additional": map[string]interface{}{
									"id": "event-3",
								},
							},
							// Add an ungrouped event (different location)
							map[string]interface{}{
								"name":            "Different Event",
								"description":     "An event at a different location",
								"eventOwners":     []interface{}{"012"},
								"eventOwnerName":  "Other Host",
								"eventSourceType": "SLF",
								"startTime":       time.Now().Add(72 * time.Hour).Unix(),
								"endTime":         time.Now().Add(74 * time.Hour).Unix(),
								"address":         "456 Other St",
								"lat":             34.0522,
								"long":            -118.2437,
								"timezone":        "America/Los_Angeles",
								"_additional": map[string]interface{}{
									"id": "event-4",
								},
							},
						},
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
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 HOME PAGE GROUPED EVENTS TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)

	// Add MNM_OPTIONS_CTX_KEY to context
	fakeContext := context.Background()

	// Create a request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = req.WithContext(fakeContext)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := GetHomeOrUserPage(rr, req)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}

	responseBody := rr.Body.String()

	// Verify grouped events are displayed with carousel
	if !strings.Contains(responseBody, "Weekly Meetup") {
		t.Errorf("Grouped event 'Weekly Meetup' should appear on the page")
	}

	if !strings.Contains(responseBody, "carousel-container") {
		t.Errorf("Carousel container should appear for grouped events")
	}

	if !strings.Contains(responseBody, "123 Main St") {
		t.Errorf("Grouped event address should appear")
	}

	// Verify ungrouped event also appears
	if !strings.Contains(responseBody, "Different Event") {
		t.Errorf("Ungrouped event 'Different Event' should also appear on the page")
	}

	// Verify event IDs appear in URLs (they will be in the carousel links)
	if !strings.Contains(responseBody, "/event/event-1") {
		t.Errorf("Event ID should appear in URL for grouped events")
	}
}

func TestGetHomeOrUserPage_EventTypeFiltering(t *testing.T) {
	// ==================================================================================
	// UNIT TEST: Filter Construction and Default Values
	// ==================================================================================
	// This test verifies that:
	// 1. The home page query uses DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES
	// 2. The filter is sent to Weaviate with the correct event source types
	// 3. Only events with matching types appear in the response
	//
	// ==================================================================================
	// INTEGRATION TEST REQUIRED: Weaviate Tokenization Behavior
	// ==================================================================================
	// ⚠️  WHAT THIS TEST CANNOT VERIFY:
	// This unit test CANNOT verify that Weaviate's "field" tokenization prevents
	// substring matching. That requires an integration test with a real Weaviate instance.
	//
	// 🔍 THE PROBLEM:
	// With "word" tokenization (default), "SLF_UNPUB" tokenizes to ["SLF", "UNPUB"].
	// A filter for "SLF" using ContainsAny would match "SLF_UNPUB" (substring match).
	//
	// ✅ THE SOLUTION:
	// With "field" tokenization (configured in weaviate_service.go schema), "SLF_UNPUB"
	// is treated as a single token. A filter for "SLF" will NOT match "SLF_UNPUB".
	//
	// 🧪 TO VALIDATE THE FIX:
	// 1. Restart the server after changing the schema name to force recreation
	// 2. Seed events with different types: SLF, SLF_EVS, SLF_UNPUB, etc.
	// 3. Query the home page: curl localhost:8000
	// 4. Verify that ONLY SLF and SLF_EVS events appear (not SLF_UNPUB)
	//
	// 📝 SCHEMA CONFIGURATION:
	// See weaviate_service.go line ~178:
	//   {Name: "eventSourceType", DataType: []string{"text"},
	//    Tokenization: "field", // ← This is critical for exact matching
	//    ...
	//   }
	// ==================================================================================

	// Verify that DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES only includes published types
	expectedTypes := []string{constants.ES_SERIES_PARENT, constants.ES_SINGLE_EVENT} // ["SLF_EVS", "SLF"]
	if len(constants.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES) != len(expectedTypes) {
		t.Errorf("DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES length mismatch: got %d, want %d",
			len(constants.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES), len(expectedTypes))
	}

	for i, expectedType := range expectedTypes {
		if constants.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[i] != expectedType {
			t.Errorf("DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[%d] = %s, want %s",
				i, constants.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[i], expectedType)
		}
	}

	// Verify that unpublished types are NOT in the default searchable types
	unpublishedTypes := []string{
		constants.ES_SINGLE_EVENT_UNPUB,  // "SLF_UNPUB"
		constants.ES_SERIES_PARENT_UNPUB, // "SLF_EVS_UNPUB"
		constants.ES_EVENT_SERIES_UNPUB,  // "EVS_UNPUB"
	}

	for _, unpubType := range unpublishedTypes {
		for _, searchableType := range constants.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES {
			if searchableType == unpubType {
				t.Errorf("DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES should not include unpublished type: %s", unpubType)
			}
		}
	}
}

func TestGetAdminPage(t *testing.T) {
	req, err := http.NewRequest("GET", "/profile", nil)
	if err != nil {
		t.Fatal(err)
	}

	mockUserInfo := constants.UserInfo{
		Email:             "test@domain.com",
		EmailVerified:     true,
		GivenName:         "Demo",
		FamilyName:        "User",
		Name:              "Demo User",
		PreferredUsername: "test@domain.com",
		Sub:               "testID",
		UpdatedAt:         123234234,
	}

	mockRoleClaims := []constants.RoleClaim{
		{
			Role:        "orgAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "superAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "sysAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
	}

	// Save original environment variables
	originalAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	originalNamespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")

	// Set test environment variables
	os.Setenv("CLOUDFLARE_ACCOUNT_ID", "test-account-id")
	os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", "test-namespace-id")

	// Defer resetting environment variables
	defer func() {
		os.Setenv("CLOUDFLARE_ACCOUNT_ID", originalAccountID)
		os.Setenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID", originalNamespaceID)
	}()

	// Create mock Cloudflare server
	mockCloudflareServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and method
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if the request is for the correct endpoint
		if !strings.Contains(r.URL.Path, "/client/v4/accounts/test-account-id/storage/kv/namespaces/test-namespace-id/values/") {
			http.Error(w, "Invalid endpoint", http.StatusNotFound)
			return
		}

		// Mock successful response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "result": "test-value"}`))
	}))

	// Set up the mock server using proper port binding
	cloudflareHostAndPort := test_helpers.GetNextPort()
	listener, err := test_helpers.BindToPort(t, cloudflareHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind Cloudflare server: %v", err)
	}
	mockCloudflareServer.Listener = listener
	mockCloudflareServer.Start()
	defer mockCloudflareServer.Close()

	ctx := context.WithValue(req.Context(), "userInfo", mockUserInfo)
	ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
	ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := GetAdminPage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}

func TestGetMapEmbedPage(t *testing.T) {
	req, err := http.NewRequest("GET", "/map-embed?address=New York", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up context with APIGatewayV2HTTPRequest
	ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		QueryStringParameters: map[string]string{"address": "New York"},
	})
	// Add MNM_OPTIONS_CTX_KEY to context
	ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := GetMapEmbedPage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}

func TestGetEventDetailsPage(t *testing.T) {
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

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK WEAVIATE SERVER HIT: %s %s", r.Method, r.URL.Path)

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

			// Mock response for event details page
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"_additional": map[string]interface{}{
									"id": "123",
								},
								"eventOwners":           []interface{}{"789"},
								"eventOwnerName":        "Event Host Test",
								"name":                  "Test Event",
								"description":           "This is a test event",
								"address":               "123 Main St, Anytown, USA",
								"hasPurchasable":        true,
								"hasRegistrationFields": true,
								"startingPrice":         50,
								"timezone":              "America/New_York",
								"startTime":             time.Now().Add(48 * time.Hour).Unix(),
							},
						},
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

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 EVENT DETAILS PAGE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Weaviate Server bound to: %s", actualAddr)

	const eventID = "123"
	req, err := http.NewRequest("GET", "/event/"+eventID, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up context with APIGatewayV2HTTPRequest
	ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		PathParameters: map[string]string{
			constants.EVENT_ID_KEY: eventID,
		},
	})

	mockUserInfo := constants.UserInfo{
		Email:             "test@domain.com",
		EmailVerified:     true,
		GivenName:         "Demo",
		FamilyName:        "User",
		Name:              "Demo User",
		PreferredUsername: "test@domain.com",
		Sub:               "testID",
		UpdatedAt:         123234234,
	}

	mockRoleClaims := []constants.RoleClaim{
		{
			Role:        "superAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
	}

	ctx = context.WithValue(req.Context(), "userInfo", mockUserInfo)
	ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
	ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
	req = req.WithContext(ctx)

	// Set up router to extract variables
	router := test_helpers.SetupStaticTestRouter(t, "./assets")

	// Add middleware to inject context values into all requests (for Playwright requests)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Inject context values into the request
			ctx := r.Context()
			ctx = context.WithValue(ctx, "userInfo", mockUserInfo)
			ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
			ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
			// Also add APIGatewayV2HTTPRequest context for path parameters
			if vars := mux.Vars(r); vars != nil {
				if eventID, ok := vars[constants.EVENT_ID_KEY]; ok {
					ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
						PathParameters: map[string]string{
							constants.EVENT_ID_KEY: eventID,
						},
					})
				}
			}
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	})

	router.HandleFunc("/event/{"+constants.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		GetEventDetailsPage(w, r).ServeHTTP(w, r)
	})

	router.HandleFunc("/api/purchasables/{"+constants.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		json, _ := json.Marshal(map[string]interface{}{
			"purchasable_items": []map[string]interface{}{
				{
					"name":              "Test Ticket",
					"cost":              1000,
					"inventory":         10,
					"description":       "Test Description",
					"currency":          "USD",
					"registration_type": "text",
					"registration_fields": []string{
						"Test Field",
					},
				},
			},
		})
		w.Write(json)
	})

	router.HandleFunc("/api/registration-fields/{"+constants.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		json, _ := json.Marshal(map[string]interface{}{
			"registration_fields": []map[string]interface{}{
				{
					"name": "Test Field",
				},
			},
		})
		w.Write(json)
	})

	router.HandleFunc("/api/checkout/{"+constants.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		json, _ := json.Marshal(map[string]interface{}{
			"checkout_url": "https://checkout.stripe.com/test_checkout_url",
		})
		w.Write(json)
	})

	// Create a real HTTP server using the router
	testServerPort := test_helpers.GetNextPort()
	testServer := httptest.NewUnstartedServer(router)
	testServerListener, err := test_helpers.BindToPort(t, testServerPort)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	testServer.Listener = testServerListener
	testServer.Start()
	defer testServer.Close()

	browser, err := test_helpers.GetPlaywrightBrowser()
	if err != nil {
		t.Skipf("skipping playwright flow: %v", err)
	}
	if browser == nil {
		t.Skip("skipping playwright flow: browser unavailable")
	}
	page, err := (*browser).NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v\n", err)
	}

	// Now use testServer.URL to access the server
	if _, err = page.Goto(fmt.Sprintf("%s/event/123", testServer.URL)); err != nil {
		log.Fatalf("could not goto: %v\n", err)
	}

	// Check if the event title is visible
	if _, err := page.Locator("h1").IsVisible(); err != nil {
		t.Errorf("Event title is not visible")
	}

	title, err := page.Locator("h1").AllTextContents()
	if err != nil {
		t.Errorf("Error getting event title: %v", err)
	}

	if title[0] != string("Test Event") {
		t.Errorf("Failed to find event title, found: %s", title[0])
	}

	// Add timeout and error handling for the buy tickets click
	buyTktsLocator := page.Locator("#buy-tkts")
	if err := buyTktsLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(5000), // 5 second timeout
	}); err != nil {
		// Take a screenshot for debugging
		screenshotPath := fmt.Sprintf("debug_buy_tkts_%s.png", eventID)
		test_helpers.ScreenshotToStandardDir(t, page, screenshotPath)
		t.Fatalf("Failed to click #buy-tkts button: %v", err)
	}

	// Add timeout for increment buttons
	if err := page.Locator("[data-input-counter-increment]").Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Failed to click first increment button: %v", err)
	}
	if err := page.Locator("[data-input-counter-increment]").Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Failed to click second increment button: %v", err)
	}
	wasCheckoutCalled := false
	// Expect API call to checkout endpoint
	page.OnRequest(func(request playwright.Request) {
		if strings.Contains(request.URL(), "api/checkout") {
			wasCheckoutCalled = true
			body, err := request.PostData()
			if err != nil {
				t.Fatalf("Failed to get request body: %v", err)
			}
			expectedBody := `{"event_name":"Test Event","purchased_items":[{"name":"Test Ticket","cost":1000,"quantity":2,"currency":"USD","reg_responses":[]}],"total":2000,"currency":"USD"}`
			if body != expectedBody {
				t.Errorf("Expected request body %s, got %s", expectedBody, body)
			}
			wasCheckoutCalled = true
		}
	})

	// Click the checkout button
	checkoutLocator := page.Locator("button:has-text('Checkout')")
	if err := checkoutLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Failed to click checkout button: %v", err)
	}
	_, err = page.ExpectRequest("**/api/checkout/**", func() error {
		return checkoutLocator.Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(5000),
		})
	})
	if err != nil {
		t.Fatalf("Checkout request not observed: %v", err)
	}
	// Verify the request was made to the mock server
	// The mock server will handle the request and we can verify its response
	// in the mock server handler above
	if !wasCheckoutCalled {
		t.Errorf("Checkout API call was not made")
		screenshotName := fmt.Sprintf("event_details_%s.png", eventID)
		test_helpers.ScreenshotToStandardDir(t, page, screenshotName)
	}
}

func TestGetAddEventSourcePage(t *testing.T) {
	req, err := http.NewRequest("GET", "/admin", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := GetAddEventSourcePage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}

func TestGetSearchParamsFromReq(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		cfRay          string
		expectedQuery  string
		expectedCity   string
		expectedLoc    []float64
		expectedRadius float64
		expectedStart  int64
		expectedEnd    int64
		expectedCfLoc  constants.CdnLocation
	}{
		{
			name: "All parameters provided",
			queryParams: map[string]string{
				"start_time": "4070908800",
				"end_time":   "4071808800",
				"lat":        "40.7128",
				"lon":        "-74.0060",
				"radius":     "1000",
				"q":          "test query",
				"location":   "New York, NY",
			},
			cfRay:          "1234567890000-EWR",
			expectedQuery:  "test query",
			expectedCity:   "New York, NY",
			expectedLoc:    []float64{40.7128, -74.0060},
			expectedRadius: 1000,
			expectedStart:  4070908800,
			expectedEnd:    4071808800,
			expectedCfLoc:  helpers.CfLocationMap["EWR"],
		},
		{
			name: "Lat + lon params with no radius",
			queryParams: map[string]string{
				"start_time": "4070908800",
				"end_time":   "4071808800",
				"lat":        "40.7128",
				"lon":        "-74.0060",
				"radius":     "",
				"q":          "",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedCity:   "",
			expectedLoc:    []float64{40.7128, -74.0060},
			expectedRadius: constants.DEFAULT_SEARCH_RADIUS,
			expectedStart:  4070908800,
			expectedEnd:    4071808800,
			expectedCfLoc:  constants.CdnLocation{},
		},
		{
			name:           "No parameters provided",
			queryParams:    map[string]string{},
			cfRay:          "",
			expectedQuery:  "",
			expectedCity:   "",
			expectedLoc:    []float64{helpers.Cities[0].Latitude, helpers.Cities[0].Longitude},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  constants.CdnLocation{},
		},
		{
			name: "Only location parameters",
			queryParams: map[string]string{
				"lat":    "35.6762",
				"lon":    "139.6503",
				"radius": "500",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedCity:   "",
			expectedLoc:    []float64{35.6762, 139.6503},
			expectedRadius: 500,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  constants.CdnLocation{},
		},
		{
			name: "Only time parameters",
			queryParams: map[string]string{
				"start_time": "this_week",
				"end_time":   "",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedCity:   "",
			expectedLoc:    []float64{helpers.Cities[0].Latitude, helpers.Cities[0].Longitude},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be 7 days from now in Unix seconds
			expectedCfLoc:  constants.CdnLocation{},
		},
		{
			name:           "Only CF-Ray header",
			queryParams:    map[string]string{},
			cfRay:          "1234567890000-LAX",
			expectedLoc:    []float64{helpers.CfLocationMap["LAX"].Lat, helpers.CfLocationMap["LAX"].Lon}, // Los Angeles coordinates
			expectedCfLoc:  helpers.CfLocationMap["LAX"],
			expectedRadius: constants.DEFAULT_SEARCH_RADIUS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL:    &url.URL{RawQuery: encodeParams(tt.queryParams)},
				Header: make(http.Header),
			}
			if tt.cfRay != "" {
				req.Header.Set("cf-ray", tt.cfRay)
			}

			// TODO: need to test `categories` and `ownerIds` returned here
			query, city, loc, radius, start, end, cfLoc, _, _, _, _, _, _ := GetSearchParamsFromReq(req)

			if query != tt.expectedQuery {
				t.Errorf("Expected query %s, got %s", tt.expectedQuery, query)
			}

			if city != tt.expectedCity {
				t.Errorf("Expected city: %#v, got %s", tt.expectedCity, city)
			}

			if !floatSliceEqual(loc, tt.expectedLoc, 0.0001) {
				t.Errorf("Expected location %v, got %v", tt.expectedLoc, loc)
			}

			if math.Abs(radius-tt.expectedRadius) > 0.0001 {
				t.Errorf("Expected radius %f, got %f", tt.expectedRadius, radius)
			}

			if tt.expectedStart != 0 {
				if start != tt.expectedStart {
					t.Errorf("Expected start time %d, got %d", tt.expectedStart, start)
				}
			} else {
				if start <= 0 {
					t.Errorf("Expected start time to be greater than 0, got %d", start)
				}
			}

			if tt.expectedEnd != 0 {
				if end != tt.expectedEnd {
					t.Errorf("Expected end time %d, got %d", tt.expectedEnd, end)
				}
			} else {
				if end <= start {
					t.Errorf("Expected end time to be greater than start time, got start: %d, end: %d", start, end)
				}
			}

			if cfLoc != tt.expectedCfLoc {
				t.Errorf("Expected CF location %v, got %v", tt.expectedCfLoc, cfLoc)
			}
		})
	}
}

func encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	return values.Encode()
}

func floatSliceEqual(a, b []float64, epsilon float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > epsilon {
			return false
		}
	}
	return true
}

func TestGetAddOrEditEventPage(t *testing.T) {
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

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK WEAVIATE SERVER HIT: %s %s", r.Method, r.URL.Path)

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

			// Mock response for event lookup
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"_additional": map[string]interface{}{
									"id": "123",
								},
								"eventOwners":    []interface{}{"testID"}, // Match the test user's Sub
								"eventOwnerName": "Event Host Test",
								"name":           "Test Event",
								"description":    "This is a test event",
								"timezone":       "America/New_York",
								"startTime":      time.Now().Add(48 * time.Hour).Unix(),
							},
						},
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

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 ADD/EDIT EVENT PAGE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Weaviate Server bound to: %s", actualAddr)

	// Test cases
	tests := []struct {
		name           string
		eventID        string
		userInfo       constants.UserInfo
		roleClaims     []constants.RoleClaim
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "Add new event as superAdmin",
			eventID: "",
			userInfo: constants.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "superAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Add Event",
		},
		{
			name:    "Edit existing event as event owner",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Edit Event",
		},
		{
			name:    "Unauthorized user",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "user", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Only event editors can add or edit events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request path based on whether it's add or edit
			path := "/event"
			if tt.eventID != "" {
				path = fmt.Sprintf("/event/%s/edit", tt.eventID)
			}

			req, err := http.NewRequest("GET", path, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set up context with user info and role claims
			ctx := context.WithValue(req.Context(), "userInfo", tt.userInfo)
			ctx = context.WithValue(ctx, "roleClaims", tt.roleClaims)
			// Add API Gateway context with path parameters if we have an event ID
			if tt.eventID != "" {
				ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
					PathParameters: map[string]string{
						constants.EVENT_ID_KEY: tt.eventID,
					},
				})
			}

			req = req.WithContext(ctx)

			// Set up router to extract variables
			router := mux.NewRouter()
			router.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
				GetAddOrEditEventPage(w, r).ServeHTTP(w, r)
			})
			router.HandleFunc("/event/{"+constants.EVENT_ID_KEY+"}/edit", func(w http.ResponseWriter, r *http.Request) {
				GetAddOrEditEventPage(w, r).ServeHTTP(w, r)
			})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Logf("Handler returned body: %s", rr.Body.String())
				t.Errorf("Handler returned unexpected body: expected to contain %q", tt.expectedBody)
			}
		})
	}
}

func TestGetEventAttendeesPage(t *testing.T) {
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

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK WEAVIATE SERVER HIT: %s %s", r.Method, r.URL.Path)

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

			// Mock response for event lookup - depends on test case
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						constants.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"_additional": map[string]interface{}{
									"id": "123",
								},
								"eventOwners":    []interface{}{"authorizedUserID"},
								"eventOwnerName": "Event Host Test",
								"name":           "Test Event",
								"description":    "This is a test event",
								"timezone":       "America/New_York",
								"startTime":      time.Now().Add(48 * time.Hour).Unix(),
							},
						},
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

	// Use the same binding pattern as working test
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 EVENT ATTENDEES PAGE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Weaviate Server bound to: %s", actualAddr)

	tests := []struct {
		name           string
		eventID        string
		userInfo       constants.UserInfo
		roleClaims     []constants.RoleClaim
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "Authorized user (event owner)",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "authorized@example.com",
				Sub:   "authorizedUserID",
				Name:  "Authorized User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Test Event", // Or some other expected content from the attendees page
		},
		{
			name:    "Unauthorized user (not event owner)",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "unauthorized@example.com",
				Sub:   "unauthorizedUserID",
				Name:  "Unauthorized User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "You are not authorized to edit this event",
		},
		{
			name:    "Superadmin can access any event",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "admin@example.com",
				Sub:   "adminUserID",
				Name:  "Admin User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "superAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Test Event",
		},
		{
			name:    "User without required role",
			eventID: "123",
			userInfo: constants.UserInfo{
				Email: "user@example.com",
				Sub:   "regularUserID",
				Name:  "Regular User",
			},
			roleClaims: []constants.RoleClaim{
				{Role: "user", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Only event editors can add or edit events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/event/%s/attendees", tt.eventID)
			req, err := http.NewRequest("GET", path, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set up context with user info and role claims
			ctx := context.WithValue(req.Context(), "userInfo", tt.userInfo)
			ctx = context.WithValue(ctx, "roleClaims", tt.roleClaims)
			ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{
					constants.EVENT_ID_KEY: tt.eventID,
				},
			})
			req = req.WithContext(ctx)

			// Set up router to extract variables
			router := mux.NewRouter()
			router.HandleFunc("/event/{"+constants.EVENT_ID_KEY+"}/attendees", func(w http.ResponseWriter, r *http.Request) {
				GetEventAttendeesPage(w, r).ServeHTTP(w, r)
			})

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("Handler returned unexpected body: expected to contain %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

// =============================================================================
// Tests for GetPricingPage handler
// =============================================================================

func TestGetPricingPage(t *testing.T) {
	// Save original environment variables
	originalStripeGrowth := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalStripeSeed := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalStripeGrowth)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalStripeSeed)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	// Set up test environment variables
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	tests := []struct {
		name               string
		userInfo           constants.UserInfo
		expectedStatusCode int
		shouldContain      []string
		shouldNotContain   []string
	}{
		{
			name: "Success - Logged in user",
			userInfo: constants.UserInfo{
				Sub:   "user123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			expectedStatusCode: http.StatusOK,
			shouldContain: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Free",
				"$15",
				"$50",
				"Get Started",
			},
			shouldNotContain: []string{},
		},
		{
			name:               "Success - Not logged in",
			userInfo:           constants.UserInfo{},
			expectedStatusCode: http.StatusOK,
			shouldContain: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Free",
				"$15",
				"$50",
				"Get Started",
			},
			shouldNotContain: []string{},
		},
		{
			name: "Success - Displays subscription features",
			userInfo: constants.UserInfo{
				Sub:   "user456",
				Email: "user2@example.com",
			},
			expectedStatusCode: http.StatusOK,
			shouldContain: []string{
				"Custom subdomain",
				"Host Events",
				"Custom registration forms",
				"Custom theme and branding",
				"Syndicate and re-publish events",
				"API Access",
			},
			shouldNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest("GET", "/pricing", nil)

			// Add user info to context if provided
			ctx := req.Context()
			if tt.userInfo.Sub != "" {
				ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			}
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handler := GetPricingPage(w, req)
			handler(w, req)

			// Verify status code
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			// Verify response body contains expected strings
			body := w.Body.String()
			for _, expected := range tt.shouldContain {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected response to contain '%s'", expected)
				}
			}

			// Verify response body doesn't contain unexpected strings
			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(body, unexpected) {
					t.Errorf("Expected response to NOT contain '%s'", unexpected)
				}
			}

			// Verify Content-Type is HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("Expected Content-Type to contain 'text/html', got '%s'", contentType)
			}
		})
	}
}

func TestGetPricingPage_WithQueryParams(t *testing.T) {
	// Save original environment variables
	originalStripeGrowth := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalStripeSeed := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalStripeGrowth)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalStripeSeed)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	tests := []struct {
		name               string
		queryParams        string
		userInfo           constants.UserInfo
		expectedStatusCode int
		shouldContain      []string
	}{
		{
			name:        "Success - With error query param",
			queryParams: "?error=checkout_failed",
			userInfo: constants.UserInfo{
				Sub: "user123",
			},
			expectedStatusCode: http.StatusOK,
			shouldContain: []string{
				"Plans and Pricing",
				// The error handling is done in JavaScript, so we just verify the page renders
			},
		},
		{
			name:        "Success - With success query param",
			queryParams: "?success=true",
			userInfo: constants.UserInfo{
				Sub: "user456",
			},
			expectedStatusCode: http.StatusOK,
			shouldContain: []string{
				"Plans and Pricing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query params
			url := "/pricing" + tt.queryParams
			req := httptest.NewRequest("GET", url, nil)

			// Add user info to context
			ctx := req.Context()
			if tt.userInfo.Sub != "" {
				ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			}
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			handler := GetPricingPage(w, req)
			handler(w, req)

			// Verify status code
			if w.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, w.Code)
			}

			// Verify expected content
			body := w.Body.String()
			for _, expected := range tt.shouldContain {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected response to contain '%s'", expected)
				}
			}
		})
	}
}

func TestGetPricingPage_RendersCorrectPlanIDs(t *testing.T) {
	// This test verifies that the correct Stripe plan IDs are embedded in the page
	originalStripeGrowth := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalStripeSeed := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalStripeGrowth)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalStripeSeed)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	testGrowthPlan := "price_1234567890growth"
	testSeedPlan := "price_1234567890seed"

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", testGrowthPlan)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", testSeedPlan)
	os.Setenv("APEX_URL", "https://test.example.com")

	req := httptest.NewRequest("GET", "/pricing", nil)
	ctx := context.WithValue(req.Context(), "userInfo", constants.UserInfo{Sub: "user123"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler := GetPricingPage(w, req)
	handler(w, req)

	// Verify the response contains the plan IDs in the JavaScript
	body := w.Body.String()
	if !strings.Contains(body, testGrowthPlan) {
		t.Errorf("Expected response to contain Growth plan ID '%s'", testGrowthPlan)
	}
	if !strings.Contains(body, testSeedPlan) {
		t.Errorf("Expected response to contain Seed plan ID '%s'", testSeedPlan)
	}

	// Verify data attributes are set correctly
	if !strings.Contains(body, `data-growth-plan-id="`+testGrowthPlan+`"`) {
		t.Error("Expected response to contain data-growth-plan-id attribute with correct value")
	}
	if !strings.Contains(body, `data-seed-plan-id="`+testSeedPlan+`"`) {
		t.Error("Expected response to contain data-seed-plan-id attribute with correct value")
	}
}
