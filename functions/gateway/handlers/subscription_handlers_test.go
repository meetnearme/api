package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestCreateCustomerPortalSession(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	// Set up test environment
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")
	os.Setenv("APEX_URL", "https://example.com")

	// Set up logging transport
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock Stripe server for customer search/creation and portal session
	mockStripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK STRIPE HIT: %s %s", r.Method, r.URL.Path)

		switch {
		case strings.Contains(r.URL.Path, "/v1/customers/search"):
			t.Logf("   └─ Handling customer search")
			// Mock customer search - return no customers (will trigger creation)
			mockSearchResponse := map[string]interface{}{
				"object":   "search_result",
				"data":     []interface{}{},
				"has_more": false,
			}
			responseBytes, _ := json.Marshal(mockSearchResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case r.URL.Path == "/v1/customers" && r.Method == "POST":
			t.Logf("   └─ Handling customer creation")
			// Mock customer creation
			mockCustomerResponse := map[string]interface{}{
				"id":    "cus_test_123",
				"email": "test@example.com",
				"name":  "Test User",
				"metadata": map[string]interface{}{
					"zitadel_user_id": "zitadel_user_123",
				},
			}
			responseBytes, _ := json.Marshal(mockCustomerResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case r.URL.Path == "/v1/billing_portal/sessions":
			t.Logf("   └─ Handling billing portal session creation")
			// Mock portal session creation
			mockSessionResponse := map[string]interface{}{
				"id":  "bps_test_session_123",
				"url": "https://billing.stripe.com/p/session/test_session_123",
			}
			responseBytes, _ := json.Marshal(mockSessionResponse)
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

	tests := []struct {
		name             string
		method           string
		userInfo         constants.UserInfo
		queryParams      map[string]string
		expectedStatus   int
		expectedRedirect string
		expectedError    bool
	}{
		{
			name:   "successful portal session creation with default return URL",
			method: "POST",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			queryParams:      map[string]string{},
			expectedStatus:   http.StatusSeeOther,
			expectedRedirect: "https://billing.stripe.com/p/session/test_session_123",
			expectedError:    false,
		},
		{
			name:   "successful portal session with custom return URL",
			method: "POST",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			queryParams: map[string]string{
				"return_url": "/admin/subscriptions",
			},
			expectedStatus:   http.StatusSeeOther,
			expectedRedirect: "https://billing.stripe.com/p/session/test_session_123",
			expectedError:    false,
		},
		{
			name:   "successful portal session with subscription ID and default flow type",
			method: "POST",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			queryParams: map[string]string{
				"subscription_id": "sub_test_123",
			},
			expectedStatus:   http.StatusSeeOther,
			expectedRedirect: "https://billing.stripe.com/p/session/test_session_123",
			expectedError:    false,
		},
		{
			name:   "successful portal session with subscription cancel flow",
			method: "POST",
			userInfo: constants.UserInfo{
				Sub:   "zitadel_user_123",
				Email: "test@example.com",
				Name:  "Test User",
			},
			queryParams: map[string]string{
				"subscription_id": "sub_test_123",
				"flow_type":       constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_CANCEL,
			},
			expectedStatus:   http.StatusSeeOther,
			expectedRedirect: "https://billing.stripe.com/p/session/test_session_123",
			expectedError:    false,
		},
		{
			name:           "missing user ID returns unauthorized",
			method:         "POST",
			userInfo:       constants.UserInfo{},
			queryParams:    map[string]string{},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(tt.method, "/api/customer-portal/session", nil)

			// Add query parameters
			q := req.URL.Query()
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}
			req.URL.RawQuery = q.Encode()

			// Add context with user info
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", tt.userInfo)
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			err := CreateCustomerPortalSession(w, req)

			// Check error
			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify response
			result := w.Result()
			defer result.Body.Close()

			if result.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, result.StatusCode)
			}

			// For redirects, verify location header
			if tt.expectedStatus == http.StatusSeeOther {
				location := result.Header.Get("Location")
				if location != tt.expectedRedirect {
					t.Errorf("Expected redirect to '%s', got '%s'", tt.expectedRedirect, location)
				}
			}
		})
	}
}
