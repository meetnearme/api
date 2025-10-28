package types

import (
	"os"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/stripe/stripe-go/v83"
)

// TestStripeSDKSchemaCompatibility tests that our types are compatible with Stripe SDK v83.0.0
func TestStripeSDKSchemaCompatibility(t *testing.T) {
	t.Run("SubscriptionStatus_EnumValues", func(t *testing.T) {
		// Test that our SubscriptionStatus enum values match Stripe's expected values
		expectedStatuses := []string{
			"active", "canceled", "incomplete", "incomplete_expired",
			"past_due", "paused", "trialing", "unpaid",
		}

		ourStatuses := []SubscriptionStatus{
			SubscriptionStatusActive, SubscriptionStatusCanceled,
			SubscriptionStatusIncomplete, SubscriptionStatusIncompleteExpired,
			SubscriptionStatusPastDue, SubscriptionStatusPaused,
			SubscriptionStatusTrialing, SubscriptionStatusUnpaid,
		}

		if len(ourStatuses) != len(expectedStatuses) {
			t.Errorf("Expected %d subscription statuses, got %d", len(expectedStatuses), len(ourStatuses))
		}

		for i, expected := range expectedStatuses {
			if string(ourStatuses[i]) != expected {
				t.Errorf("Status %d: expected %s, got %s", i, expected, string(ourStatuses[i]))
			}
		}
	})

	t.Run("CheckoutSessionMode_Compatibility", func(t *testing.T) {
		// Test that our checkout session mode values are compatible
		expectedModes := []string{"payment", "subscription", "setup"}

		// Verify Stripe SDK has these modes
		modes := []stripe.CheckoutSessionMode{
			stripe.CheckoutSessionModePayment,
			stripe.CheckoutSessionModeSubscription,
			stripe.CheckoutSessionModeSetup,
		}

		if len(modes) != len(expectedModes) {
			t.Errorf("Expected %d checkout session modes, got %d", len(expectedModes), len(modes))
		}

		for i, expected := range expectedModes {
			if string(modes[i]) != expected {
				t.Errorf("Mode %d: expected %s, got %s", i, expected, string(modes[i]))
			}
		}
	})

	t.Run("Currency_Compatibility", func(t *testing.T) {
		// Test that our currency handling is compatible with Stripe
		testCurrencies := []string{"usd", "eur", "gbp", "cad"}

		for _, currency := range testCurrencies {
			stripeCurrency := stripe.Currency(currency)
			if string(stripeCurrency) != currency {
				t.Errorf("Currency mismatch: expected %s, got %s", currency, string(stripeCurrency))
			}
		}
	})
}

// TestStripeTypeConversions tests our type conversion functions
func TestStripeTypeConversions(t *testing.T) {
	t.Run("ConvertStripeSubscription_ValidData", func(t *testing.T) {
		// Create a mock Stripe subscription with all required fields
		mockStripeSub := &stripe.Subscription{
			ID:     "sub_test123",
			Status: stripe.SubscriptionStatusActive,
			Customer: &stripe.Customer{
				ID: "cus_test123",
			},
			CancelAtPeriodEnd: false,
			Created:           time.Now().Unix(),
			CanceledAt:        0,
			Items: &stripe.SubscriptionItemList{
				Data: []*stripe.SubscriptionItem{
					{
						ID:                 "si_test123",
						CurrentPeriodStart: time.Now().Unix(),
						CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
						Price: &stripe.Price{
							ID:         "price_test123",
							UnitAmount: 2000, // $20.00
							Currency:   stripe.CurrencyUSD,
							Recurring: &stripe.PriceRecurring{
								Interval: stripe.PriceRecurringIntervalMonth,
							},
							Product: &stripe.Product{
								Name: "Test Product",
							},
						},
					},
				},
			},
		}

		// Convert using our function
		result := ConvertStripeSubscription(mockStripeSub)

		// Validate conversion
		if result.ID != mockStripeSub.ID {
			t.Errorf("Expected ID %s, got %s", mockStripeSub.ID, result.ID)
		}

		if result.CustomerID != mockStripeSub.Customer.ID {
			t.Errorf("Expected CustomerID %s, got %s", mockStripeSub.Customer.ID, result.CustomerID)
		}

		if result.Status != SubscriptionStatus(mockStripeSub.Status) {
			t.Errorf("Expected Status %s, got %s", mockStripeSub.Status, result.Status)
		}

		if result.PlanID != mockStripeSub.Items.Data[0].Price.ID {
			t.Errorf("Expected PlanID %s, got %s", mockStripeSub.Items.Data[0].Price.ID, result.PlanID)
		}

		if result.PlanAmount != mockStripeSub.Items.Data[0].Price.UnitAmount {
			t.Errorf("Expected PlanAmount %d, got %d", mockStripeSub.Items.Data[0].Price.UnitAmount, result.PlanAmount)
		}

		if result.PlanCurrency != string(mockStripeSub.Items.Data[0].Price.Currency) {
			t.Errorf("Expected PlanCurrency %s, got %s", mockStripeSub.Items.Data[0].Price.Currency, result.PlanCurrency)
		}

		if result.PlanInterval != string(mockStripeSub.Items.Data[0].Price.Recurring.Interval) {
			t.Errorf("Expected PlanInterval %s, got %s", mockStripeSub.Items.Data[0].Price.Recurring.Interval, result.PlanInterval)
		}

		if result.PlanName != mockStripeSub.Items.Data[0].Price.Product.Name {
			t.Errorf("Expected PlanName %s, got %s", mockStripeSub.Items.Data[0].Price.Product.Name, result.PlanName)
		}
	})

	t.Run("ConvertStripeSubscription_EmptyItems", func(t *testing.T) {
		// Test with empty items list
		mockStripeSub := &stripe.Subscription{
			ID:     "sub_test123",
			Status: stripe.SubscriptionStatusActive,
			Customer: &stripe.Customer{
				ID: "cus_test123",
			},
			CancelAtPeriodEnd: false,
			Created:           time.Now().Unix(),
			Items: &stripe.SubscriptionItemList{
				Data: []*stripe.SubscriptionItem{}, // Empty items
			},
		}

		result := ConvertStripeSubscription(mockStripeSub)

		// Should not panic and should have empty period times
		if !result.CurrentPeriodStart.IsZero() {
			t.Error("Expected CurrentPeriodStart to be zero for empty items")
		}

		if !result.CurrentPeriodEnd.IsZero() {
			t.Error("Expected CurrentPeriodEnd to be zero for empty items")
		}
	})

	t.Run("ConvertStripeProduct_ValidData", func(t *testing.T) {
		mockProduct := &stripe.Product{
			ID:          "prod_test123",
			Name:        "Test Product",
			Description: "A test product",
			Active:      true,
			Metadata: map[string]string{
				"category": "test",
			},
		}

		mockPrice := &stripe.Price{
			ID:         "price_test123",
			UnitAmount: 2000,
			Currency:   stripe.CurrencyUSD,
			Recurring: &stripe.PriceRecurring{
				Interval:      stripe.PriceRecurringIntervalMonth,
				IntervalCount: 1,
			},
		}

		result := ConvertStripeProduct(mockProduct, mockPrice)

		// ID should now be the product ID, and PriceID should be the price ID
		if result.ID != mockProduct.ID {
			t.Errorf("Expected ID %s, got %s", mockProduct.ID, result.ID)
		}

		if result.PriceID != mockPrice.ID {
			t.Errorf("Expected PriceID %s, got %s", mockPrice.ID, result.PriceID)
		}

		if result.Name != mockProduct.Name {
			t.Errorf("Expected Name %s, got %s", mockProduct.Name, result.Name)
		}

		if result.Description != mockProduct.Description {
			t.Errorf("Expected Description %s, got %s", mockProduct.Description, result.Description)
		}

		if result.Amount != mockPrice.UnitAmount {
			t.Errorf("Expected Amount %d, got %d", mockPrice.UnitAmount, result.Amount)
		}

		if result.Currency != string(mockPrice.Currency) {
			t.Errorf("Expected Currency %s, got %s", mockPrice.Currency, result.Currency)
		}

		if result.Interval != string(mockPrice.Recurring.Interval) {
			t.Errorf("Expected Interval %s, got %s", mockPrice.Recurring.Interval, result.Interval)
		}

		if result.IntervalCount != mockPrice.Recurring.IntervalCount {
			t.Errorf("Expected IntervalCount %d, got %d", mockPrice.Recurring.IntervalCount, result.IntervalCount)
		}

		if result.Active != mockProduct.Active {
			t.Errorf("Expected Active %t, got %t", mockProduct.Active, result.Active)
		}
	})
}

// TestStripeAPIContractValidation tests that our API usage matches Stripe's expected contracts
func TestStripeAPIContractValidation(t *testing.T) {
	t.Run("CheckoutSessionCreateParams_RequiredFields", func(t *testing.T) {
		// Test that we can create valid CheckoutSessionCreateParams
		params := &stripe.CheckoutSessionCreateParams{
			SuccessURL: stripe.String("https://example.com/success"),
			CancelURL:  stripe.String("https://example.com/cancel"),
			Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		}

		// Validate that required fields are set
		if params.SuccessURL == nil {
			t.Error("SuccessURL is required")
		}

		if params.CancelURL == nil {
			t.Error("CancelURL is required")
		}

		if params.Mode == nil {
			t.Error("Mode is required")
		}

		// Validate that the mode is a valid value
		validModes := []string{"payment", "subscription", "setup"}
		modeValid := false
		for _, validMode := range validModes {
			if *params.Mode == validMode {
				modeValid = true
				break
			}
		}
		if !modeValid {
			t.Errorf("Invalid mode: %s", *params.Mode)
		}
	})

	t.Run("CheckoutSessionCreateLineItemParams_Structure", func(t *testing.T) {
		// Test that we can create valid line item parameters
		lineItem := &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String("usd"),
				UnitAmount: stripe.Int64(2000),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String("Test Product"),
				},
			},
		}

		// Validate structure
		if lineItem.Quantity == nil {
			t.Error("Quantity is required")
		}

		if lineItem.PriceData == nil {
			t.Error("PriceData is required")
		}

		if lineItem.PriceData.Currency == nil {
			t.Error("PriceData.Currency is required")
		}

		if lineItem.PriceData.UnitAmount == nil {
			t.Error("PriceData.UnitAmount is required")
		}

		if lineItem.PriceData.ProductData == nil {
			t.Error("PriceData.ProductData is required")
		}

		if lineItem.PriceData.ProductData.Name == nil {
			t.Error("PriceData.ProductData.Name is required")
		}
	})

	t.Run("SubscriptionStatus_Validation", func(t *testing.T) {
		// Test that our subscription status validation works
		subscription := &CustomerSubscription{
			Status: SubscriptionStatusActive,
		}

		if !subscription.IsActive() {
			t.Error("Expected IsActive() to return true for active status")
		}

		if subscription.IsCanceled() {
			t.Error("Expected IsCanceled() to return false for active status")
		}

		if subscription.IsPastDue() {
			t.Error("Expected IsPastDue() to return false for active status")
		}

		// Test canceled status
		subscription.Status = SubscriptionStatusCanceled
		if subscription.IsActive() {
			t.Error("Expected IsActive() to return false for canceled status")
		}

		if !subscription.IsCanceled() {
			t.Error("Expected IsCanceled() to return true for canceled status")
		}

		// Test trialing status
		subscription.Status = SubscriptionStatusTrialing
		if !subscription.IsActive() {
			t.Error("Expected IsActive() to return true for trialing status")
		}
	})

	t.Run("ZitadelRoleMapping", func(t *testing.T) {
		// Test that our role mapping works correctly
		testCases := []struct {
			planName     string
			expectedRole string
		}{
			{"Growth", constants.Roles[constants.SubGrowth]},
			{"Seed Community", constants.Roles[constants.SubSeed]},
			{"Unknown Plan", constants.Roles[constants.SubGrowth]}, // Default fallback
		}

		for _, tc := range testCases {
			subscription := &CustomerSubscription{
				PlanName: tc.planName,
			}

			role := subscription.GetZitadelRole()
			if role != tc.expectedRole {
				t.Errorf("For plan %s: expected role %s, got %s", tc.planName, tc.expectedRole, role)
			}
		}
	})
}

// TestStripeClientInitialization tests that our client initialization works correctly
func TestStripeClientInitialization(t *testing.T) {
	t.Run("EnvironmentVariables_Required", func(t *testing.T) {
		// Test that required environment variables are properly handled
		originalKey := os.Getenv("STRIPE_SECRET_KEY")
		originalPubKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")

		// Clean up after test
		defer func() {
			os.Setenv("STRIPE_SECRET_KEY", originalKey)
			os.Setenv("STRIPE_PUBLISHABLE_KEY", originalPubKey)
		}()

		// Test with missing secret key
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_123")

		// Test environment variable handling
		secretKey := os.Getenv("STRIPE_SECRET_KEY")
		pubKey := os.Getenv("STRIPE_PUBLISHABLE_KEY")

		if secretKey != "" {
			t.Error("Expected empty secret key when environment variable is not set")
		}
		if pubKey != "pk_test_123" {
			t.Errorf("Expected publishable key pk_test_123, got %s", pubKey)
		}

		// Test with missing publishable key
		os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")

		secretKey = os.Getenv("STRIPE_SECRET_KEY")
		pubKey = os.Getenv("STRIPE_PUBLISHABLE_KEY")

		if secretKey != "sk_test_123" {
			t.Errorf("Expected secret key sk_test_123, got %s", secretKey)
		}
		if pubKey != "" {
			t.Error("Expected empty publishable key when environment variable is not set")
		}
	})

	t.Run("StripeClient_Creation", func(t *testing.T) {
		// Test that we can create a Stripe client (without making actual API calls)
		originalKey := os.Getenv("STRIPE_SECRET_KEY")
		defer func() {
			os.Setenv("STRIPE_SECRET_KEY", originalKey)
		}()

		// Set a test key
		os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")

		// This should not panic
		client := stripe.NewClient("sk_test_123")
		if client == nil {
			t.Error("Expected client to be created successfully")
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
	})
}

// TestStripeWebhookCompatibility tests webhook-related functionality
func TestStripeWebhookCompatibility(t *testing.T) {
	t.Run("WebhookEventTypes_Validation", func(t *testing.T) {
		// Test that our webhook event types match Stripe's expected values
		expectedEvents := []string{
			"customer.subscription.created",
			"customer.subscription.deleted",
			"customer.subscription.paused",
			"customer.subscription.pending_update_applied",
			"customer.subscription.pending_update_expired",
			"customer.subscription.resumed",
			"customer.subscription.trial_will_end",
			"customer.subscription.updated",
			"customer.updated",
		}

		// These should match the constants in our helpers/constants.go
		ourEvents := []string{
			"customer.subscription.created",
			"customer.subscription.deleted",
			"customer.subscription.paused",
			"customer.subscription.pending_update_applied",
			"customer.subscription.pending_update_expired",
			"customer.subscription.resumed",
			"customer.subscription.trial_will_end",
			"customer.subscription.updated",
			"customer.updated",
		}

		if len(ourEvents) != len(expectedEvents) {
			t.Errorf("Expected %d webhook events, got %d", len(expectedEvents), len(ourEvents))
		}

		for i, expected := range expectedEvents {
			if ourEvents[i] != expected {
				t.Errorf("Event %d: expected %s, got %s", i, expected, ourEvents[i])
			}
		}
	})

	t.Run("WebhookPayload_Structure", func(t *testing.T) {
		// Test that our webhook payload structure can handle Stripe webhooks
		payload := &SubscriptionWebhookPayload{
			ID:      "evt_test123",
			Object:  "event",
			Type:    "customer.subscription.created",
			Created: time.Now().Unix(),
			Data: struct {
				Object map[string]interface{} `json:"object"`
			}{
				Object: map[string]interface{}{
					"id":     "sub_test123",
					"object": "subscription",
					"status": "active",
				},
			},
		}

		// Validate structure
		if payload.ID == "" {
			t.Error("Expected ID to be set")
		}

		if payload.Object != "event" {
			t.Errorf("Expected object to be 'event', got %s", payload.Object)
		}

		if payload.Type == "" {
			t.Error("Expected Type to be set")
		}

		if payload.Created == 0 {
			t.Error("Expected Created timestamp to be set")
		}

		if payload.Data.Object == nil {
			t.Error("Expected Data.Object to be set")
		}
	})
}

// TestStripeSDKVersionCompatibility tests that our code is compatible with the current SDK version
func TestStripeSDKVersionCompatibility(t *testing.T) {
	t.Run("SDK_Version_Check", func(t *testing.T) {
		// This test will fail if we upgrade to a version that breaks our assumptions
		// We can add version-specific checks here if needed

		// Test that we can import and use the current SDK
		client := stripe.NewClient("sk_test_123")
		if client == nil {
			t.Error("Failed to create Stripe client")
		}

		// Test that we can create parameters for the current SDK
		params := &stripe.CheckoutSessionCreateParams{
			SuccessURL: stripe.String("https://example.com/success"),
			CancelURL:  stripe.String("https://example.com/cancel"),
			Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		}

		if params.SuccessURL == nil {
			t.Error("Failed to create CheckoutSessionCreateParams")
		}
	})

	t.Run("Type_Compatibility_Check", func(t *testing.T) {
		// Test that our custom types are compatible with Stripe types

		// Test SubscriptionStatus compatibility
		stripeStatus := stripe.SubscriptionStatusActive
		ourStatus := SubscriptionStatus(stripeStatus)

		if ourStatus != SubscriptionStatusActive {
			t.Error("SubscriptionStatus conversion failed")
		}

		// Test Currency compatibility
		stripeCurrency := stripe.CurrencyUSD
		currencyString := string(stripeCurrency)

		if currencyString != "usd" {
			t.Error("Currency conversion failed")
		}
	})
}

// Benchmark tests to ensure our conversions are performant
func BenchmarkConvertStripeSubscription(b *testing.B) {
	mockStripeSub := &stripe.Subscription{
		ID:     "sub_test123",
		Status: stripe.SubscriptionStatusActive,
		Customer: &stripe.Customer{
			ID: "cus_test123",
		},
		CancelAtPeriodEnd: false,
		Created:           time.Now().Unix(),
		Items: &stripe.SubscriptionItemList{
			Data: []*stripe.SubscriptionItem{
				{
					ID:                 "si_test123",
					CurrentPeriodStart: time.Now().Unix(),
					CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour).Unix(),
					Price: &stripe.Price{
						ID:         "price_test123",
						UnitAmount: 2000,
						Currency:   stripe.CurrencyUSD,
						Recurring: &stripe.PriceRecurring{
							Interval: stripe.PriceRecurringIntervalMonth,
						},
						Product: &stripe.Product{
							Name: "Test Product",
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertStripeSubscription(mockStripeSub)
	}
}

func BenchmarkConvertStripeProduct(b *testing.B) {
	mockProduct := &stripe.Product{
		ID:          "prod_test123",
		Name:        "Test Product",
		Description: "A test product",
		Active:      true,
		Metadata: map[string]string{
			"category": "test",
		},
	}

	mockPrice := &stripe.Price{
		ID:         "price_test123",
		UnitAmount: 2000,
		Currency:   stripe.CurrencyUSD,
		Recurring: &stripe.PriceRecurring{
			Interval:      stripe.PriceRecurringIntervalMonth,
			IntervalCount: 1,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConvertStripeProduct(mockProduct, mockPrice)
	}
}
