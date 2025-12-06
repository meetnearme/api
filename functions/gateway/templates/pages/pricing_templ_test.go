package pages

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestPricingPage(t *testing.T) {
	// Set up environment variables for the template
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	testCases := []struct {
		name                  string
		isLoggedIn            bool
		hasSeedSubscription   bool
		hasGrowthSubscription bool
		expectedStrings       []string
		notExpectedStrings    []string
		basicButtonText       string // "Registered" or "Get Started"
		seedButtonText        string // "Subscribed" or "Get Started"
		growthButtonText      string // "Subscribed" or "Get Started"
	}{
		{
			name:                  "Not logged in - No subscriptions",
			isLoggedIn:            false,
			hasSeedSubscription:   false,
			hasGrowthSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Get Started", // Should appear 3 times (Basic, Seed, Growth)
			},
			notExpectedStrings: []string{
				"Registered",
				"Subscribed",
			},
			basicButtonText:  "Get Started",
			seedButtonText:   "Get Started",
			growthButtonText: "Get Started",
		},
		{
			name:                  "Logged in - No subscriptions",
			isLoggedIn:            true,
			hasSeedSubscription:   false,
			hasGrowthSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Registered",  // Basic plan for logged-in users
				"Get Started", // Should appear for Seed and Growth
			},
			notExpectedStrings: []string{
				"Subscribed",
			},
			basicButtonText:  "Registered",
			seedButtonText:   "Get Started",
			growthButtonText: "Get Started",
		},
		{
			name:                  "Logged in - Seed subscription only",
			isLoggedIn:            true,
			hasSeedSubscription:   true,
			hasGrowthSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Registered",  // Basic plan
				"Subscribed",  // Seed plan (Growth also has access to Seed features)
				"Get Started", // Growth plan
			},
			notExpectedStrings: []string{},
			basicButtonText:    "Registered",
			seedButtonText:     "Subscribed",
			growthButtonText:   "Get Started",
		},
		{
			name:                  "Logged in - Growth subscription only",
			isLoggedIn:            true,
			hasSeedSubscription:   false,
			hasGrowthSubscription: true,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Registered", // Basic plan
				"Subscribed", // Seed plan (Growth includes Seed)
				"Subscribed", // Growth plan
			},
			notExpectedStrings: []string{
				"Get Started", // Should not appear since user has Growth subscription
			},
			basicButtonText:  "Registered",
			seedButtonText:   "Subscribed", // Growth subscription gives access to Seed tier
			growthButtonText: "Subscribed",
		},
		{
			name:                  "Logged in - Both subscriptions (edge case)",
			isLoggedIn:            true,
			hasSeedSubscription:   true,
			hasGrowthSubscription: true,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Seed Community",
				"Growth Community",
				"Registered", // Basic plan
				"Subscribed", // Seed plan
				"Subscribed", // Growth plan
			},
			notExpectedStrings: []string{
				"Get Started", // Should not appear since user has all subscriptions
			},
			basicButtonText:  "Registered",
			seedButtonText:   "Subscribed",
			growthButtonText: "Subscribed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the PricingPage component
			pricingPage := PricingPage(tc.isLoggedIn, tc.hasSeedSubscription, tc.hasGrowthSubscription)

			// Create a fake context
			fakeContext := context.Background()
			fakeContext = context.WithValue(fakeContext, constants.MNM_OPTIONS_CTX_KEY, map[string]string{
				"userId": "123",
				"--p":    "#000000",
			})

			// Create mock user info
			mockUserInfo := constants.UserInfo{
				Sub:  "user123",
				Name: "Test User",
			}

			// Wrap with Layout template
			layoutTemplate := Layout(constants.SitePages["pricing"], mockUserInfo, pricingPage, types.Event{}, false, fakeContext, []string{}, false)

			// Render the template
			var buf bytes.Buffer
			err := layoutTemplate.Render(fakeContext, &buf)
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			renderedContent := buf.String()

			// Verify expected strings are present
			for _, expected := range tc.expectedStrings {
				count := strings.Count(renderedContent, expected)
				if count == 0 {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
				}
			}

			// Verify not expected strings are absent
			for _, notExpected := range tc.notExpectedStrings {
				if strings.Contains(renderedContent, notExpected) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", notExpected)
				}
			}

			// Verify button texts
			// Count occurrences of each button text
			basicButtonCount := strings.Count(renderedContent, tc.basicButtonText)
			seedButtonCount := strings.Count(renderedContent, tc.seedButtonText)
			growthButtonCount := strings.Count(renderedContent, tc.growthButtonText)

			// Basic button should appear once
			if basicButtonCount == 0 {
				t.Errorf("Expected Basic plan button text '%s' to appear, but it didn't", tc.basicButtonText)
			}

			// Seed button should appear once
			if seedButtonCount == 0 {
				t.Errorf("Expected Seed plan button text '%s' to appear, but it didn't", tc.seedButtonText)
			}

			// Growth button should appear once
			if growthButtonCount == 0 {
				t.Errorf("Expected Growth plan button text '%s' to appear, but it didn't", tc.growthButtonText)
			}

			// Verify plan IDs are in the JavaScript
			if !strings.Contains(renderedContent, "price_growth_test") {
				t.Error("Expected rendered content to contain Growth plan ID 'price_growth_test'")
			}
			if !strings.Contains(renderedContent, "price_seed_test") {
				t.Error("Expected rendered content to contain Seed plan ID 'price_seed_test'")
			}

			// Verify subscription status logic in rendered content
			// If user has Growth subscription, Seed should show "Subscribed" (since Growth includes Seed features)
			if tc.hasGrowthSubscription && !strings.Contains(renderedContent, "Subscribed") {
				t.Error("Expected 'Subscribed' button for Seed plan when user has Growth subscription")
			}

			// If user has Seed subscription, Seed should show "Subscribed"
			if tc.hasSeedSubscription && !tc.hasGrowthSubscription && !strings.Contains(renderedContent, "Subscribed") {
				t.Error("Expected 'Subscribed' button for Seed plan when user has Seed subscription")
			}
		})
	}
}

func TestPricingPage_ButtonStates(t *testing.T) {
	// Set up environment variables
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	tests := []struct {
		name                   string
		isLoggedIn             bool
		hasSeedSubscription    bool
		hasGrowthSubscription  bool
		expectedBasicDisabled  bool
		expectedSeedDisabled   bool
		expectedGrowthDisabled bool
	}{
		{
			name:                   "Not logged in - all buttons enabled",
			isLoggedIn:             false,
			hasSeedSubscription:    false,
			hasGrowthSubscription:  false,
			expectedBasicDisabled:  false,
			expectedSeedDisabled:   false,
			expectedGrowthDisabled: false,
		},
		{
			name:                   "Logged in - Basic disabled, others enabled",
			isLoggedIn:             true,
			hasSeedSubscription:    false,
			hasGrowthSubscription:  false,
			expectedBasicDisabled:  true, // "Registered" button is disabled
			expectedSeedDisabled:   false,
			expectedGrowthDisabled: false,
		},
		{
			name:                   "Logged in with Seed - Basic and Seed disabled, Growth enabled",
			isLoggedIn:             true,
			hasSeedSubscription:    true,
			hasGrowthSubscription:  false,
			expectedBasicDisabled:  true,
			expectedSeedDisabled:   true, // "Subscribed" button is disabled
			expectedGrowthDisabled: false,
		},
		{
			name:                   "Logged in with Growth - all buttons disabled",
			isLoggedIn:             true,
			hasSeedSubscription:    false,
			hasGrowthSubscription:  true,
			expectedBasicDisabled:  true,
			expectedSeedDisabled:   true, // Growth includes Seed, so Seed shows "Subscribed"
			expectedGrowthDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricingPage := PricingPage(tt.isLoggedIn, tt.hasSeedSubscription, tt.hasGrowthSubscription)

			fakeContext := context.Background()
			fakeContext = context.WithValue(fakeContext, constants.MNM_OPTIONS_CTX_KEY, map[string]string{})

			mockUserInfo := constants.UserInfo{Sub: "user123"}
			layoutTemplate := Layout(constants.SitePages["pricing"], mockUserInfo, pricingPage, types.Event{}, false, fakeContext, []string{}, false)

			var buf bytes.Buffer
			err := layoutTemplate.Render(fakeContext, &buf)
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			renderedContent := buf.String()

			// Check for disabled button class
			// Disabled buttons have "pointer-events-none" class
			// Basic plan button
			basicButtonDisabled := strings.Contains(renderedContent, "Basic Community") && strings.Contains(renderedContent, "pointer-events-none")
			if basicButtonDisabled != tt.expectedBasicDisabled {
				t.Errorf("Basic button disabled state: expected %v, but got %v", tt.expectedBasicDisabled, basicButtonDisabled)
			}

			// Seed plan button
			// Check between Seed Community section and Growth Community section
			seedSectionStart := strings.Index(renderedContent, "Seed Community")
			growthSectionStart := strings.Index(renderedContent, "Growth Community")
			if seedSectionStart != -1 && growthSectionStart != -1 {
				seedSection := renderedContent[seedSectionStart:growthSectionStart]
				seedButtonDisabled := strings.Contains(seedSection, "pointer-events-none")
				if seedButtonDisabled != tt.expectedSeedDisabled {
					t.Errorf("Seed button disabled state: expected %v, but got %v", tt.expectedSeedDisabled, seedButtonDisabled)
				}
			}

			// Growth plan button
			// Check after Growth Community section
			if growthSectionStart != -1 {
				growthSection := renderedContent[growthSectionStart:]
				growthButtonDisabled := strings.Contains(growthSection, "pointer-events-none")
				if growthButtonDisabled != tt.expectedGrowthDisabled {
					t.Errorf("Growth button disabled state: expected %v, but got %v", tt.expectedGrowthDisabled, growthButtonDisabled)
				}
			}
		})
	}
}

func TestPricingPage_RendersPlanIDs(t *testing.T) {
	// Set up environment variables
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	testGrowthPlan := "price_1234567890growth"
	testSeedPlan := "price_1234567890seed"
	testApexURL := "https://test.example.com"

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", testGrowthPlan)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", testSeedPlan)
	os.Setenv("APEX_URL", testApexURL)

	pricingPage := PricingPage(false, false, false)

	fakeContext := context.Background()
	fakeContext = context.WithValue(fakeContext, constants.MNM_OPTIONS_CTX_KEY, map[string]string{})

	mockUserInfo := constants.UserInfo{}
	layoutTemplate := Layout(constants.SitePages["pricing"], mockUserInfo, pricingPage, types.Event{}, false, fakeContext, []string{}, false)

	var buf bytes.Buffer
	err := layoutTemplate.Render(fakeContext, &buf)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	renderedContent := buf.String()

	// Verify plan IDs are in the JavaScript data attributes
	if !strings.Contains(renderedContent, testGrowthPlan) {
		t.Errorf("Expected rendered content to contain Growth plan ID '%s'", testGrowthPlan)
	}
	if !strings.Contains(renderedContent, testSeedPlan) {
		t.Errorf("Expected rendered content to contain Seed plan ID '%s'", testSeedPlan)
	}
	if !strings.Contains(renderedContent, testApexURL) {
		t.Errorf("Expected rendered content to contain APEX URL '%s'", testApexURL)
	}
}
