package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

// customRoundTripper intercepts HTTP requests and redirects Stripe API calls to our mock server
type customRoundTripper struct {
	transport http.RoundTripper
	mockURL   string
}

func (c *customRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// If this is a Stripe API request, redirect it to our mock server
	if strings.Contains(req.URL.Host, "api.stripe.com") {
		mockURL, _ := url.Parse(c.mockURL)
		req.URL.Scheme = mockURL.Scheme
		req.URL.Host = mockURL.Host
	}

	// Use the underlying transport to make the request
	return c.transport.RoundTrip(req)
}

func TestGetStripeSubscriptionPlanIDs(t *testing.T) {
	var testGrowthPlanID = "test_growth_plan_id"
	var testSeedPlanID = "test_seed_plan_id"

	var originalGrowthPlanID = os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	var originalSeedPlanID = os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", testGrowthPlanID)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", testSeedPlanID)

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlanID)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlanID)
	}()

	growthPlanID, seedPlanID := GetStripeSubscriptionPlanIDs()

	// In a real test, you might want to verify specific values
	if growthPlanID != testGrowthPlanID {
		t.Errorf("Expected growth plan ID %s, got '%s'", testGrowthPlanID, growthPlanID)
	}
	if seedPlanID != testSeedPlanID {
		t.Errorf("Expected seed plan ID %s, got '%s'", testSeedPlanID, seedPlanID)
	}
}

func TestStripeSubscriptionService_GetZitadelRole(t *testing.T) {
	tests := []struct {
		name         string
		planName     string
		expectedRole string
	}{
		{
			name:         "Growth plan maps to subGrowth",
			planName:     "Growth",
			expectedRole: constants.Roles[constants.SubGrowth],
		},
		{
			name:         "Seed Community plan maps to subSeed",
			planName:     "Seed Community",
			expectedRole: constants.Roles[constants.SubSeed],
		},
		{
			name:         "Unknown plan defaults to subGrowth",
			planName:     "Unknown Plan",
			expectedRole: constants.Roles[constants.SubGrowth],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := &types.CustomerSubscription{
				PlanName: tt.planName,
			}

			role := subscription.GetZitadelRole()
			if role != tt.expectedRole {
				t.Errorf("GetZitadelRole() = %v, want %v", role, tt.expectedRole)
			}
		})
	}
}

func TestStripeSubscriptionService_IsActive(t *testing.T) {
	tests := []struct {
		name           string
		status         types.SubscriptionStatus
		expectedActive bool
	}{
		{
			name:           "Active subscription is active",
			status:         types.SubscriptionStatusActive,
			expectedActive: true,
		},
		{
			name:           "Trialing subscription is active",
			status:         types.SubscriptionStatusTrialing,
			expectedActive: true,
		},
		{
			name:           "Canceled subscription is not active",
			status:         types.SubscriptionStatusCanceled,
			expectedActive: false,
		},
		{
			name:           "Past due subscription is not active",
			status:         types.SubscriptionStatusPastDue,
			expectedActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := &types.CustomerSubscription{
				Status: tt.status,
			}

			active := subscription.IsActive()
			if active != tt.expectedActive {
				t.Errorf("IsActive() = %v, want %v", active, tt.expectedActive)
			}
		})
	}
}

func TestStripeSubscriptionService_IsCanceled(t *testing.T) {
	tests := []struct {
		name             string
		status           types.SubscriptionStatus
		expectedCanceled bool
	}{
		{
			name:             "Canceled subscription is canceled",
			status:           types.SubscriptionStatusCanceled,
			expectedCanceled: true,
		},
		{
			name:             "Active subscription is not canceled",
			status:           types.SubscriptionStatusActive,
			expectedCanceled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := &types.CustomerSubscription{
				Status: tt.status,
			}

			canceled := subscription.IsCanceled()
			if canceled != tt.expectedCanceled {
				t.Errorf("IsCanceled() = %v, want %v", canceled, tt.expectedCanceled)
			}
		})
	}
}

func TestStripeSubscriptionService_IsPastDue(t *testing.T) {
	tests := []struct {
		name            string
		status          types.SubscriptionStatus
		expectedPastDue bool
	}{
		{
			name:            "Past due subscription is past due",
			status:          types.SubscriptionStatusPastDue,
			expectedPastDue: true,
		},
		{
			name:            "Active subscription is not past due",
			status:          types.SubscriptionStatusActive,
			expectedPastDue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription := &types.CustomerSubscription{
				Status: tt.status,
			}

			pastDue := subscription.IsPastDue()
			if pastDue != tt.expectedPastDue {
				t.Errorf("IsPastDue() = %v, want %v", pastDue, tt.expectedPastDue)
			}
		})
	}
}

func TestStripeSubscriptionService_GetSubscriptionPlans(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")

	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")

	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
	}()

	// Set up logging transport for debugging
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock Stripe server using httptest.NewServer (handles binding automatically)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK STRIPE HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/prices/price_growth_test":
			t.Logf("   └─ Handling Growth plan price request")
			mockPriceResponse := map[string]interface{}{
				"id":          "price_growth_test",
				"object":      "price",
				"active":      true,
				"currency":    "usd",
				"unit_amount": 2000,
				"recurring": map[string]interface{}{
					"interval": "month",
				},
				"product": "prod_growth_test",
			}
			responseBytes, _ := json.Marshal(mockPriceResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case "/v1/prices/price_seed_test":
			t.Logf("   └─ Handling Seed plan price request")
			mockPriceResponse := map[string]interface{}{
				"id":          "price_seed_test",
				"object":      "price",
				"active":      true,
				"currency":    "usd",
				"unit_amount": 1000,
				"recurring": map[string]interface{}{
					"interval": "month",
				},
				"product": "prod_seed_test",
			}
			responseBytes, _ := json.Marshal(mockPriceResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case "/v1/products/prod_growth_test":
			t.Logf("   └─ Handling Growth product request")
			mockProductResponse := map[string]interface{}{
				"id":          "prod_growth_test",
				"object":      "product",
				"name":        "Growth",
				"description": "Growth subscription plan",
				"active":      true,
			}
			responseBytes, _ := json.Marshal(mockProductResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		case "/v1/products/prod_seed_test":
			t.Logf("   └─ Handling Seed product request")
			mockProductResponse := map[string]interface{}{
				"id":          "prod_seed_test",
				"object":      "product",
				"name":        "Seed Community",
				"description": "Seed Community subscription plan",
				"active":      true,
			}
			responseBytes, _ := json.Marshal(mockProductResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED STRIPE PATH: %s", r.URL.Path)
			t.Errorf("mock Stripe server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Set up environment variables
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")

	// Create a custom RoundTripper that intercepts Stripe API calls and redirects them to our mock server
	customTransport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			// Redirect Stripe API calls to our mock server
			if strings.Contains(req.URL.Host, "api.stripe.com") {
				mockURL, _ := url.Parse(mockServer.URL)
				return mockURL, nil
			}
			return nil, nil
		},
	}

	// Create a custom RoundTripper that modifies the request URL
	customRoundTripper := &customRoundTripper{
		transport: customTransport,
		mockURL:   mockServer.URL,
	}

	// Override the default HTTP transport to intercept Stripe requests
	http.DefaultTransport = customRoundTripper

	// Reset the Stripe client so it uses the new transport
	ResetStripeClient()

	// Create service and test the parallelized GetSubscriptionPlans method
	service := NewStripeSubscriptionService()

	plans, err := service.GetSubscriptionPlans()

	if err != nil {
		t.Errorf("GetSubscriptionPlans() failed: %v", err)
		return
	}

	// Verify we got the expected number of plans
	if len(plans) != 2 {
		t.Errorf("Expected 2 plans, got %d", len(plans))
		return
	}

	// Verify each plan has the expected structure
	planMap := make(map[string]*types.SubscriptionPlan)
	for _, plan := range plans {
		planMap[plan.Name] = plan
	}

	// Check Growth plan
	if growthPlan, exists := planMap["Growth"]; exists {
		if growthPlan.ID != "prod_growth_test" {
			t.Errorf("Expected Growth plan ID 'prod_growth_test', got '%s'", growthPlan.ID)
		}
		if growthPlan.PriceID != "price_growth_test" {
			t.Errorf("Expected Growth plan PriceID 'price_growth_test', got '%s'", growthPlan.PriceID)
		}
		if growthPlan.Amount != 2000 {
			t.Errorf("Expected Growth plan amount 2000, got %d", growthPlan.Amount)
		}
		t.Logf("✅ Growth plan: ID=%s, PriceID=%s, Name=%s, Amount=%d", growthPlan.ID, growthPlan.PriceID, growthPlan.Name, growthPlan.Amount)
	} else {
		t.Error("Growth plan not found in results")
	}

	// Check Seed plan
	if seedPlan, exists := planMap["Seed Community"]; exists {
		if seedPlan.ID != "prod_seed_test" {
			t.Errorf("Expected Seed plan ID 'prod_seed_test', got '%s'", seedPlan.ID)
		}
		if seedPlan.PriceID != "price_seed_test" {
			t.Errorf("Expected Seed plan PriceID 'price_seed_test', got '%s'", seedPlan.PriceID)
		}
		if seedPlan.Amount != 1000 {
			t.Errorf("Expected Seed plan amount 1000, got %d", seedPlan.Amount)
		}
		t.Logf("✅ Seed plan: ID=%s, PriceID=%s, Name=%s, Amount=%d", seedPlan.ID, seedPlan.PriceID, seedPlan.Name, seedPlan.Amount)
	} else {
		t.Error("Seed Community plan not found in results")
	}

	t.Logf("✅ Successfully retrieved %d subscription plans in parallel", len(plans))
}

func TestStripeSubscriptionService_GetCustomerSubscriptions(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
	}()

	// Set up logging transport for debugging
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock Stripe server using httptest.NewServer (handles binding automatically)
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK STRIPE HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/subscriptions":
			t.Logf("   └─ Handling subscriptions list request")
			if r.Method != "GET" {
				t.Errorf("expected method GET for /v1/subscriptions, got %s", r.Method)
			}

			// Check query parameters
			customerID := r.URL.Query().Get("customer")
			status := r.URL.Query().Get("status")

			if customerID != "cus_test_customer" {
				t.Errorf("expected customer=cus_test_customer, got %s", customerID)
			}
			if status != "all" {
				t.Errorf("expected status=all, got %s", status)
			}

			// Mock subscription response
			mockSubscriptionsResponse := map[string]interface{}{
				"object": "list",
				"data": []interface{}{
					map[string]interface{}{
						"id":                   "sub_test_subscription_1",
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
									"id":                   "si_test_item_1",
									"object":               "subscription_item",
									"current_period_start": 1234567890,
									"current_period_end":   1234567890 + 2592000,
									"price": map[string]interface{}{
										"id":      "price_growth_test",
										"object":  "price",
										"product": "prod_growth_test",
										"recurring": map[string]interface{}{
											"interval": "month",
										},
									},
								},
							},
						},
						"current_period_start": 1234567890,
						"current_period_end":   1234567890 + 2592000, // 30 days later
					},
					map[string]interface{}{
						"id":                   "sub_test_subscription_2",
						"object":               "subscription",
						"status":               "trialing",
						"customer":             "cus_test_customer",
						"cancel_at_period_end": false,
						"created":              1234567890,
						"canceled_at":          nil,
						"items": map[string]interface{}{
							"object": "list",
							"data": []interface{}{
								map[string]interface{}{
									"id":                   "si_test_item_2",
									"object":               "subscription_item",
									"current_period_start": 1234567890,
									"current_period_end":   1234567890 + 2592000,
									"price": map[string]interface{}{
										"id":      "price_seed_test",
										"object":  "price",
										"product": "prod_seed_test",
										"recurring": map[string]interface{}{
											"interval": "month",
										},
									},
								},
							},
						},
						"current_period_start": 1234567890,
						"current_period_end":   1234567890 + 2592000, // 30 days later
					},
				},
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
	defer mockServer.Close()

	// Set up environment variables
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")

	// Create a custom RoundTripper that intercepts Stripe API calls and redirects them to our mock server
	customTransport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			// Redirect Stripe API calls to our mock server
			if strings.Contains(req.URL.Host, "api.stripe.com") {
				mockURL, _ := url.Parse(mockServer.URL)
				return mockURL, nil
			}
			return nil, nil
		},
	}

	// Create a custom RoundTripper that modifies the request URL
	customRoundTripper := &customRoundTripper{
		transport: customTransport,
		mockURL:   mockServer.URL,
	}

	// Override the default HTTP transport to intercept Stripe requests
	http.DefaultTransport = customRoundTripper

	// Create service and test GetCustomerSubscriptions
	service := NewStripeSubscriptionService()

	subscriptions, err := service.GetCustomerSubscriptions("cus_test_customer")

	if err != nil {
		t.Errorf("GetCustomerSubscriptions() failed: %v", err)
		return
	}

	// Verify we got the expected number of subscriptions
	if len(subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subscriptions))
		return
	}

	// Verify each subscription has the expected structure
	subscriptionMap := make(map[string]*types.CustomerSubscription)
	for _, sub := range subscriptions {
		subscriptionMap[sub.ID] = sub
	}

	// Check first subscription (active)
	if sub1, exists := subscriptionMap["sub_test_subscription_1"]; exists {
		if sub1.Status != "active" {
			t.Errorf("Expected first subscription status 'active', got '%s'", sub1.Status)
		}
		if sub1.CustomerID != "cus_test_customer" {
			t.Errorf("Expected customer ID 'cus_test_customer', got '%s'", sub1.CustomerID)
		}
		t.Logf("✅ Subscription 1: ID=%s, Status=%s, CustomerID=%s", sub1.ID, sub1.Status, sub1.CustomerID)
	} else {
		t.Error("First subscription not found in results")
	}

	// Check second subscription (trialing)
	if sub2, exists := subscriptionMap["sub_test_subscription_2"]; exists {
		if sub2.Status != "trialing" {
			t.Errorf("Expected second subscription status 'trialing', got '%s'", sub2.Status)
		}
		if sub2.CustomerID != "cus_test_customer" {
			t.Errorf("Expected customer ID 'cus_test_customer', got '%s'", sub2.CustomerID)
		}
		t.Logf("✅ Subscription 2: ID=%s, Status=%s, CustomerID=%s", sub2.ID, sub2.Status, sub2.CustomerID)
	} else {
		t.Error("Second subscription not found in results")
	}

}

func TestStripeSubscriptionService_CreateCustomerPortalSession(t *testing.T) {
	// Save original environment variables
	originalStripeKey := os.Getenv("STRIPE_SECRET_KEY")
	defer func() {
		os.Setenv("STRIPE_SECRET_KEY", originalStripeKey)
	}()

	// Set up logging transport for debugging
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Create mock Stripe server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK STRIPE HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/billing_portal/sessions":
			t.Logf("   └─ Handling billing portal session creation")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/billing_portal/sessions, got %s", r.Method)
			}

			// Parse form data to check parameters
			r.ParseForm()
			customerID := r.Form.Get("customer")
			returnURL := r.Form.Get("return_url")
			flowType := r.Form.Get("flow_data[type]")
			subscriptionID := r.Form.Get("flow_data[subscription_update][subscription]")
			if subscriptionID == "" {
				subscriptionID = r.Form.Get("flow_data[subscription_cancel][subscription]")
			}

			if customerID == "" {
				t.Error("expected customer parameter")
			}

			// Mock portal session response
			mockSessionResponse := map[string]interface{}{
				"id":  "bps_test_session_123",
				"url": "https://billing.stripe.com/p/session/test_session_123",
			}

			responseBytes, _ := json.Marshal(mockSessionResponse)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

			t.Logf("   └─ Created portal session: customer=%s, return_url=%s, flow_type=%s, subscription=%s",
				customerID, returnURL, flowType, subscriptionID)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED STRIPE PATH: %s", r.URL.Path)
			t.Errorf("mock Stripe server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Set up environment variables
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_mock_key")

	// Create a custom RoundTripper that intercepts Stripe API calls
	customTransport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			if strings.Contains(req.URL.Host, "api.stripe.com") {
				mockURL, _ := url.Parse(mockServer.URL)
				return mockURL, nil
			}
			return nil, nil
		},
	}

	customRoundTripper := &customRoundTripper{
		transport: customTransport,
		mockURL:   mockServer.URL,
	}

	http.DefaultTransport = customRoundTripper
	ResetStripeClient()

	// Create service
	service := NewStripeSubscriptionService()

	t.Run("create portal session without subscription ID", func(t *testing.T) {
		session, err := service.CreateCustomerPortalSession(
			"cus_test_customer",
			"https://example.com/return",
			"",
			"",
		)

		if err != nil {
			t.Errorf("CreateCustomerPortalSession() failed: %v", err)
			return
		}

		if session == nil {
			t.Error("Expected session, got nil")
			return
		}

		if session.ID != "bps_test_session_123" {
			t.Errorf("Expected session ID 'bps_test_session_123', got '%s'", session.ID)
		}

		if session.URL != "https://billing.stripe.com/p/session/test_session_123" {
			t.Errorf("Expected session URL 'https://billing.stripe.com/p/session/test_session_123', got '%s'", session.URL)
		}

		if session.ReturnURL != "https://example.com/return" {
			t.Errorf("Expected return URL 'https://example.com/return', got '%s'", session.ReturnURL)
		}

		t.Logf("✅ Portal session created: ID=%s, URL=%s", session.ID, session.URL)
	})

	t.Run("create portal session with subscription ID and default flow type", func(t *testing.T) {
		session, err := service.CreateCustomerPortalSession(
			"cus_test_customer",
			"https://example.com/return",
			"sub_test_subscription",
			"", // Empty flow type should default to subscription_update
		)

		if err != nil {
			t.Errorf("CreateCustomerPortalSession() failed: %v", err)
			return
		}

		if session == nil {
			t.Error("Expected session, got nil")
			return
		}

		t.Logf("✅ Portal session with subscription created: ID=%s, URL=%s", session.ID, session.URL)
	})

	t.Run("create portal session with subscription cancel flow", func(t *testing.T) {
		session, err := service.CreateCustomerPortalSession(
			"cus_test_customer",
			"https://example.com/return",
			"sub_test_subscription",
			constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_CANCEL,
		)

		if err != nil {
			t.Errorf("CreateCustomerPortalSession() failed: %v", err)
			return
		}

		if session == nil {
			t.Error("Expected session, got nil")
			return
		}

		t.Logf("✅ Portal session with cancel flow created: ID=%s, URL=%s", session.ID, session.URL)
	})

	t.Run("create portal session with subscription update flow", func(t *testing.T) {
		session, err := service.CreateCustomerPortalSession(
			"cus_test_customer",
			"https://example.com/return",
			"sub_test_subscription",
			constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE,
		)

		if err != nil {
			t.Errorf("CreateCustomerPortalSession() failed: %v", err)
			return
		}

		if session == nil {
			t.Error("Expected session, got nil")
			return
		}

		t.Logf("✅ Portal session with update flow created: ID=%s, URL=%s", session.ID, session.URL)
	})

	t.Run("create portal session with payment method update flow (no subscription ID)", func(t *testing.T) {
		session, err := service.CreateCustomerPortalSession(
			"cus_test_customer",
			"https://example.com/return",
			"", // No subscription ID needed for payment method update
			constants.STRIPE_PORTAL_FLOW_PAYMENT_METHOD_UPDATE,
		)

		if err != nil {
			t.Errorf("CreateCustomerPortalSession() failed: %v", err)
			return
		}

		if session == nil {
			t.Error("Expected session, got nil")
			return
		}

		if session.ID != "bps_test_session_123" {
			t.Errorf("Expected session ID 'bps_test_session_123', got '%s'", session.ID)
		}

		if session.URL != "https://billing.stripe.com/p/session/test_session_123" {
			t.Errorf("Expected session URL 'https://billing.stripe.com/p/session/test_session_123', got '%s'", session.URL)
		}

		t.Logf("✅ Portal session with payment method update flow created: ID=%s, URL=%s", session.ID, session.URL)
	})
}
