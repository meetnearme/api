package services

import (
	"os"
	"testing"

	"github.com/stripe/stripe-go/v83"
)

// TestStripeServiceInitialization tests the Stripe service initialization
func TestStripeServiceInitialization(t *testing.T) {
	t.Run("InitStripe_WithValidKey", func(t *testing.T) {
		// Save original environment
		originalKey := os.Getenv("STRIPE_SECRET_KEY")
		originalPubKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")

		defer func() {
			if originalKey != "" {
				os.Setenv("STRIPE_SECRET_KEY", originalKey)
			} else {
				os.Unsetenv("STRIPE_SECRET_KEY")
			}
			if originalPubKey != "" {
				os.Setenv("STRIPE_PUBLISHABLE_KEY", originalPubKey)
			} else {
				os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
			}
		}()

		// Set test keys
		os.Setenv("STRIPE_SECRET_KEY", "sk_test_123456789")
		os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_123456789")

		// Initialize Stripe
		InitStripe()

		// Get client
		client := GetStripeClient()
		if client == nil {
			t.Error("Expected client to be initialized")
		}

		// Verify client has expected services
		if client.V1CheckoutSessions == nil {
			t.Error("Expected V1CheckoutSessions service to be available")
		}

		if client.V1Subscriptions == nil {
			t.Error("Expected V1Subscriptions service to be available")
		}

		if client.V1Customers == nil {
			t.Error("Expected V1Customers service to be available")
		}

		if client.V1Products == nil {
			t.Error("Expected V1Products service to be available")
		}

		if client.V1Prices == nil {
			t.Error("Expected V1Prices service to be available")
		}
	})

	t.Run("InitStripe_WithEmptyKey", func(t *testing.T) {
		// Save original environment
		originalKey := os.Getenv("STRIPE_SECRET_KEY")
		originalPubKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")

		defer func() {
			if originalKey != "" {
				os.Setenv("STRIPE_SECRET_KEY", originalKey)
			} else {
				os.Unsetenv("STRIPE_SECRET_KEY")
			}
			if originalPubKey != "" {
				os.Setenv("STRIPE_PUBLISHABLE_KEY", originalPubKey)
			} else {
				os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
			}
		}()

		// Set empty keys
		os.Setenv("STRIPE_SECRET_KEY", "")
		os.Setenv("STRIPE_PUBLISHABLE_KEY", "")

		// Initialize Stripe
		InitStripe()

		// Get client
		client := GetStripeClient()
		if client == nil {
			t.Error("Expected client to be initialized even with empty keys")
		}
	})

	t.Run("GetStripeKeyPair_EnvironmentVariables", func(t *testing.T) {
		// Save original environment
		originalKey := os.Getenv("STRIPE_SECRET_KEY")
		originalPubKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")

		defer func() {
			if originalKey != "" {
				os.Setenv("STRIPE_SECRET_KEY", originalKey)
			} else {
				os.Unsetenv("STRIPE_SECRET_KEY")
			}
			if originalPubKey != "" {
				os.Setenv("STRIPE_PUBLISHABLE_KEY", originalPubKey)
			} else {
				os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
			}
		}()

		// Test with both keys set
		os.Setenv("STRIPE_SECRET_KEY", "sk_test_secret")
		os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_public")

		pubKey, secretKey := GetStripeKeyPair()
		if pubKey != "pk_test_public" {
			t.Errorf("Expected publishable key pk_test_public, got %s", pubKey)
		}
		if secretKey != "sk_test_secret" {
			t.Errorf("Expected secret key sk_test_secret, got %s", secretKey)
		}

		// Test with missing secret key
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_public")

		pubKey, secretKey = GetStripeKeyPair()
		if pubKey != "pk_test_public" {
			t.Errorf("Expected publishable key pk_test_public, got %s", pubKey)
		}
		if secretKey != "" {
			t.Errorf("Expected empty secret key, got %s", secretKey)
		}

		// Test with missing publishable key
		os.Setenv("STRIPE_SECRET_KEY", "sk_test_secret")
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")

		pubKey, secretKey = GetStripeKeyPair()
		if pubKey != "" {
			t.Errorf("Expected empty publishable key, got %s", pubKey)
		}
		if secretKey != "sk_test_secret" {
			t.Errorf("Expected secret key sk_test_secret, got %s", secretKey)
		}
	})

	t.Run("GetStripeCheckoutWebhookSecret_EnvironmentVariable", func(t *testing.T) {
		// Save original environment
		originalSecret := os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")

		defer func() {
			if originalSecret != "" {
				os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", originalSecret)
			} else {
				os.Unsetenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
			}
		}()

		// Test with webhook secret set
		os.Setenv("STRIPE_CHECKOUT_WEBHOOK_SECRET", "whsec_test123")

		secret := GetStripeCheckoutWebhookSecret()
		if secret != "whsec_test123" {
			t.Errorf("Expected webhook secret whsec_test123, got %s", secret)
		}

		// Test with missing webhook secret
		os.Unsetenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")

		secret = GetStripeCheckoutWebhookSecret()
		if secret != "" {
			t.Errorf("Expected empty webhook secret, got %s", secret)
		}
	})
}

// TestStripeClientCompatibility tests that our client usage is compatible with Stripe SDK v83.0.0
func TestStripeClientCompatibility(t *testing.T) {
	t.Run("Client_Services_Available", func(t *testing.T) {
		// Create a test client
		client := stripe.NewClient("sk_test_123456789")

		// Test that all expected services are available
		services := map[string]interface{}{
			"V1CheckoutSessions":      client.V1CheckoutSessions,
			"V1Subscriptions":         client.V1Subscriptions,
			"V1Customers":             client.V1Customers,
			"V1Products":              client.V1Products,
			"V1Prices":                client.V1Prices,
			"V1BillingPortalSessions": client.V1BillingPortalSessions,
			"V1WebhookEndpoints":      client.V1WebhookEndpoints,
		}

		for serviceName, service := range services {
			if service == nil {
				t.Errorf("Expected %s service to be available", serviceName)
			}
		}
	})

	t.Run("Client_ParameterTypes_Compatible", func(t *testing.T) {
		// Test that we can create parameter types without compilation errors

		// Test CheckoutSessionCreateParams
		checkoutParams := &stripe.CheckoutSessionCreateParams{
			SuccessURL: stripe.String("https://example.com/success"),
			CancelURL:  stripe.String("https://example.com/cancel"),
			Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		}

		if checkoutParams.SuccessURL == nil {
			t.Error("Failed to create CheckoutSessionCreateParams")
		}

		// Test CheckoutSessionCreateLineItemParams
		lineItemParams := &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(2000),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String("Test Product"),
				},
			},
		}

		if lineItemParams.Quantity == nil {
			t.Error("Failed to create CheckoutSessionCreateLineItemParams")
		}

		// Test CustomerCreateParams
		customerParams := &stripe.CustomerCreateParams{
			Email: stripe.String("test@example.com"),
			Metadata: map[string]string{
				"external_id": "user_123",
			},
		}

		if customerParams.Email == nil {
			t.Error("Failed to create CustomerCreateParams")
		}

		// Test SubscriptionCreateParams
		subscriptionParams := &stripe.SubscriptionCreateParams{
			Customer: stripe.String("cus_test123"),
			Items: []*stripe.SubscriptionCreateItemParams{
				{
					Price: stripe.String("price_test123"),
				},
			},
		}

		if subscriptionParams.Customer == nil {
			t.Error("Failed to create SubscriptionCreateParams")
		}
	})

	t.Run("Client_EnumTypes_Compatible", func(t *testing.T) {
		// Test that enum types are compatible

		// Test CheckoutSessionMode
		modes := []stripe.CheckoutSessionMode{
			stripe.CheckoutSessionModePayment,
			stripe.CheckoutSessionModeSubscription,
			stripe.CheckoutSessionModeSetup,
		}

		expectedModes := []string{"payment", "subscription", "setup"}
		for i, mode := range modes {
			if string(mode) != expectedModes[i] {
				t.Errorf("Expected mode %s, got %s", expectedModes[i], string(mode))
			}
		}

		// Test SubscriptionStatus
		statuses := []stripe.SubscriptionStatus{
			stripe.SubscriptionStatusActive,
			stripe.SubscriptionStatusCanceled,
			stripe.SubscriptionStatusPastDue,
			stripe.SubscriptionStatusTrialing,
		}

		expectedStatuses := []string{"active", "canceled", "past_due", "trialing"}
		for i, status := range statuses {
			if string(status) != expectedStatuses[i] {
				t.Errorf("Expected status %s, got %s", expectedStatuses[i], string(status))
			}
		}

		// Test Currency
		currencies := []stripe.Currency{
			stripe.CurrencyUSD,
			stripe.CurrencyEUR,
			stripe.CurrencyGBP,
		}

		expectedCurrencies := []string{"usd", "eur", "gbp"}
		for i, currency := range currencies {
			if string(currency) != expectedCurrencies[i] {
				t.Errorf("Expected currency %s, got %s", expectedCurrencies[i], string(currency))
			}
		}
	})
}

// TestStripeSDKBreakingChanges tests for potential breaking changes in future SDK versions
func TestStripeSDKBreakingChanges(t *testing.T) {
	t.Run("Client_Initialization_Pattern", func(t *testing.T) {
		// This test will fail if Stripe changes the client initialization pattern

		// Test that we can create a client with a key
		client := stripe.NewClient("sk_test_123456789")
		if client == nil {
			t.Error("Client initialization failed - potential breaking change")
		}

		// Test that the client has the expected structure
		if client.V1CheckoutSessions == nil {
			t.Error("V1CheckoutSessions service missing - potential breaking change")
		}
	})

	t.Run("Parameter_Structure_Stability", func(t *testing.T) {
		// This test will fail if Stripe changes parameter structures

		// Test CheckoutSessionCreateParams structure
		params := &stripe.CheckoutSessionCreateParams{}

		// These fields should exist and be settable
		params.SuccessURL = stripe.String("https://example.com/success")
		params.CancelURL = stripe.String("https://example.com/cancel")
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
		params.ClientReferenceID = stripe.String("ref_123")

		// Verify fields are set
		if params.SuccessURL == nil || *params.SuccessURL != "https://example.com/success" {
			t.Error("SuccessURL field structure changed - potential breaking change")
		}

		if params.CancelURL == nil || *params.CancelURL != "https://example.com/cancel" {
			t.Error("CancelURL field structure changed - potential breaking change")
		}

		if params.Mode == nil || *params.Mode != "payment" {
			t.Error("Mode field structure changed - potential breaking change")
		}

		if params.ClientReferenceID == nil || *params.ClientReferenceID != "ref_123" {
			t.Error("ClientReferenceID field structure changed - potential breaking change")
		}
	})

	t.Run("Service_Method_Availability", func(t *testing.T) {
		// This test will fail if Stripe removes or renames service methods

		client := stripe.NewClient("sk_test_123456789")

		// Test that critical service methods exist
		// Note: We can't call the actual methods without making API calls,
		// but we can verify the services exist

		if client.V1CheckoutSessions == nil {
			t.Error("V1CheckoutSessions service missing - potential breaking change")
		}

		if client.V1Subscriptions == nil {
			t.Error("V1Subscriptions service missing - potential breaking change")
		}

		if client.V1Customers == nil {
			t.Error("V1Customers service missing - potential breaking change")
		}

		if client.V1Products == nil {
			t.Error("V1Products service missing - potential breaking change")
		}

		if client.V1Prices == nil {
			t.Error("V1Prices service missing - potential breaking change")
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkStripeClientInitialization(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := stripe.NewClient("sk_test_123456789")
		_ = client
	}
}

func BenchmarkStripeParameterCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params := &stripe.CheckoutSessionCreateParams{
			SuccessURL: stripe.String("https://example.com/success"),
			CancelURL:  stripe.String("https://example.com/cancel"),
			Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		}
		_ = params
	}
}
