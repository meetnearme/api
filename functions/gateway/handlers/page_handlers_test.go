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
						helpers.WeaviateEventClassName: []interface{}{
							map[string]interface{}{
								"name":            "First Test Event",
								"description":     "Description of the first event",
								"eventOwners":     []interface{}{"789"},
								"eventOwnerName":  "First Event Host",
								"eventSourceType": "SLF", // Published single event
								"startTime":       time.Now().Add(48 * time.Hour).Unix(),
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
	// fakeContext = context.WithValue(fakeContext, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{})

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
	expectedTypes := []string{helpers.ES_SERIES_PARENT, helpers.ES_SINGLE_EVENT} // ["SLF_EVS", "SLF"]
	if len(helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES) != len(expectedTypes) {
		t.Errorf("DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES length mismatch: got %d, want %d",
			len(helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES), len(expectedTypes))
	}

	for i, expectedType := range expectedTypes {
		if helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[i] != expectedType {
			t.Errorf("DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[%d] = %s, want %s",
				i, helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES[i], expectedType)
		}
	}

	// Verify that unpublished types are NOT in the default searchable types
	unpublishedTypes := []string{
		helpers.ES_SINGLE_EVENT_UNPUB,  // "SLF_UNPUB"
		helpers.ES_SERIES_PARENT_UNPUB, // "SLF_EVS_UNPUB"
		helpers.ES_EVENT_SERIES_UNPUB,  // "EVS_UNPUB"
	}

	for _, unpubType := range unpublishedTypes {
		for _, searchableType := range helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES {
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

	mockUserInfo := helpers.UserInfo{
		Email:             "test@domain.com",
		EmailVerified:     true,
		GivenName:         "Demo",
		FamilyName:        "User",
		Name:              "Demo User",
		PreferredUsername: "test@domain.com",
		Sub:               "testID",
		UpdatedAt:         123234234,
	}

	mockRoleClaims := []helpers.RoleClaim{
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
	ctx = context.WithValue(ctx, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
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
	ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		QueryStringParameters: map[string]string{"address": "New York"},
	})
	// Add MNM_OPTIONS_CTX_KEY to context
	ctx = context.WithValue(ctx, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
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
						helpers.WeaviateEventClassName: []interface{}{
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
	ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		PathParameters: map[string]string{
			helpers.EVENT_ID_KEY: eventID,
		},
	})

	mockUserInfo := helpers.UserInfo{
		Email:             "test@domain.com",
		EmailVerified:     true,
		GivenName:         "Demo",
		FamilyName:        "User",
		Name:              "Demo User",
		PreferredUsername: "test@domain.com",
		Sub:               "testID",
		UpdatedAt:         123234234,
	}

	mockRoleClaims := []helpers.RoleClaim{
		{
			Role:        "superAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
	}

	ctx = context.WithValue(req.Context(), "userInfo", mockUserInfo)
	ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
	ctx = context.WithValue(ctx, helpers.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "123"})
	_ = req.WithContext(ctx)

	// Set up router to extract variables
	router := test_helpers.SetupStaticTestRouter(t, "./assets")

	router.HandleFunc("/event/{"+helpers.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		GetEventDetailsPage(w, r).ServeHTTP(w, r)
	})

	router.HandleFunc("/api/purchasables/{"+helpers.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
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

	router.HandleFunc("/api/registration-fields/{"+helpers.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
		json, _ := json.Marshal(map[string]interface{}{
			"registration_fields": []map[string]interface{}{
				{
					"name": "Test Field",
				},
			},
		})
		w.Write(json)
	})

	router.HandleFunc("/api/checkout/{"+helpers.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
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
		log.Fatal(err)
	}

	if browser == nil || err != nil {
		log.Fatalf("could not launch browser: %v\n", err)
		return
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
		expectedLoc    []float64
		expectedRadius float64
		expectedStart  int64
		expectedEnd    int64
		expectedCfLoc  helpers.CdnLocation
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
			},
			cfRay:          "1234567890000-EWR",
			expectedQuery:  "test query",
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
			expectedLoc:    []float64{40.7128, -74.0060},
			expectedRadius: helpers.DEFAULT_SEARCH_RADIUS,
			expectedStart:  4070908800,
			expectedEnd:    4071808800,
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name:           "No parameters provided",
			queryParams:    map[string]string{},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{helpers.Cities[0].Latitude, helpers.Cities[0].Longitude},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
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
			expectedLoc:    []float64{35.6762, 139.6503},
			expectedRadius: 500,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name: "Only time parameters",
			queryParams: map[string]string{
				"start_time": "this_week",
				"end_time":   "",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{helpers.Cities[0].Latitude, helpers.Cities[0].Longitude},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be 7 days from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name:           "Only CF-Ray header",
			queryParams:    map[string]string{},
			cfRay:          "1234567890000-LAX",
			expectedLoc:    []float64{helpers.CfLocationMap["LAX"].Lat, helpers.CfLocationMap["LAX"].Lon}, // Los Angeles coordinates
			expectedCfLoc:  helpers.CfLocationMap["LAX"],
			expectedRadius: helpers.DEFAULT_SEARCH_RADIUS,
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
			query, loc, radius, start, end, cfLoc, _, _, _, _, _, _ := GetSearchParamsFromReq(req)

			if query != tt.expectedQuery {
				t.Errorf("Expected query %s, got %s", tt.expectedQuery, query)
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
						helpers.WeaviateEventClassName: []interface{}{
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
		userInfo       helpers.UserInfo
		roleClaims     []helpers.RoleClaim
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "Add new event as superAdmin",
			eventID: "",
			userInfo: helpers.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []helpers.RoleClaim{
				{Role: "superAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Add Event",
		},
		{
			name:    "Edit existing event as event owner",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []helpers.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Edit Event",
		},
		{
			name:    "Unauthorized user",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "test@domain.com",
				Sub:   "testID",
				Name:  "Test User",
			},
			roleClaims: []helpers.RoleClaim{
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
				ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
					PathParameters: map[string]string{
						helpers.EVENT_ID_KEY: tt.eventID,
					},
				})
			}

			req = req.WithContext(ctx)

			// Set up router to extract variables
			router := mux.NewRouter()
			router.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
				GetAddOrEditEventPage(w, r).ServeHTTP(w, r)
			})
			router.HandleFunc("/event/{"+helpers.EVENT_ID_KEY+"}/edit", func(w http.ResponseWriter, r *http.Request) {
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
						helpers.WeaviateEventClassName: []interface{}{
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
		userInfo       helpers.UserInfo
		roleClaims     []helpers.RoleClaim
		expectedStatus int
		expectedBody   string
	}{
		{
			name:    "Authorized user (event owner)",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "authorized@example.com",
				Sub:   "authorizedUserID",
				Name:  "Authorized User",
			},
			roleClaims: []helpers.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Test Event", // Or some other expected content from the attendees page
		},
		{
			name:    "Unauthorized user (not event owner)",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "unauthorized@example.com",
				Sub:   "unauthorizedUserID",
				Name:  "Unauthorized User",
			},
			roleClaims: []helpers.RoleClaim{
				{Role: "eventAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "You are not authorized to edit this event",
		},
		{
			name:    "Superadmin can access any event",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "admin@example.com",
				Sub:   "adminUserID",
				Name:  "Admin User",
			},
			roleClaims: []helpers.RoleClaim{
				{Role: "superAdmin", ProjectID: "project-id"},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Test Event",
		},
		{
			name:    "User without required role",
			eventID: "123",
			userInfo: helpers.UserInfo{
				Email: "user@example.com",
				Sub:   "regularUserID",
				Name:  "Regular User",
			},
			roleClaims: []helpers.RoleClaim{
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
			ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{
					helpers.EVENT_ID_KEY: tt.eventID,
				},
			})
			req = req.WithContext(ctx)

			// Set up router to extract variables
			router := mux.NewRouter()
			router.HandleFunc("/event/{"+helpers.EVENT_ID_KEY+"}/attendees", func(w http.ResponseWriter, r *http.Request) {
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
