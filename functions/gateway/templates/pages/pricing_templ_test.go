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
	originalEnterprisePlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", originalEnterprisePlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", "price_enterprise_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	testCases := []struct {
		name                      string
		isLoggedIn                bool
		hasSeedSubscription       bool
		hasGrowthSubscription     bool
		hasEnterpriseSubscription bool
		expectedStrings           []string
		notExpectedStrings        []string
		basicButtonText           string // "Registered" or "Get Started"
		// seedButtonText         string // Seed tier is currently commented out in pricing.templ
		growthButtonText     string // "Subscribed" or "Get Started"
		enterpriseButtonText string // "Subscribed" or "Get Started"
	}{
		{
			name:                      "Not logged in - No subscriptions",
			isLoggedIn:                false,
			hasSeedSubscription:       false,
			hasGrowthSubscription:     false,
			hasEnterpriseSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				// "Seed Community", // Seed tier is currently commented out in pricing.templ
				"Growth Community",
				"Enterprise Community",
				"Get Started", // Should appear for Basic, Growth, Enterprise
			},
			notExpectedStrings: []string{
				"Registered",
				"Subscribed",
			},
			basicButtonText:      "Get Started",
			growthButtonText:     "Get Started",
			enterpriseButtonText: "Get Started",
		},
		{
			name:                      "Logged in - No subscriptions",
			isLoggedIn:                true,
			hasSeedSubscription:       false,
			hasGrowthSubscription:     false,
			hasEnterpriseSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				// "Seed Community", // Seed tier is currently commented out in pricing.templ
				"Growth Community",
				"Enterprise Community",
				"Registered",  // Basic plan for logged-in users
				"Get Started", // Should appear for Growth and Enterprise
			},
			notExpectedStrings: []string{
				"Subscribed",
			},
			basicButtonText:      "Registered",
			growthButtonText:     "Get Started",
			enterpriseButtonText: "Get Started",
		},
		{
			name:                      "Logged in - Growth subscription only",
			isLoggedIn:                true,
			hasSeedSubscription:       false,
			hasGrowthSubscription:     true,
			hasEnterpriseSubscription: false,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				// "Seed Community", // Seed tier is currently commented out in pricing.templ
				"Growth Community",
				"Enterprise Community",
				"Registered",  // Basic plan
				"Subscribed",  // Growth plan
				"Get Started", // Enterprise plan
			},
			notExpectedStrings:   []string{},
			basicButtonText:      "Registered",
			growthButtonText:     "Subscribed",
			enterpriseButtonText: "Get Started",
		},
		{
			name:                      "Logged in - Enterprise subscription only",
			isLoggedIn:                true,
			hasSeedSubscription:       false,
			hasGrowthSubscription:     false,
			hasEnterpriseSubscription: true,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Growth Community",
				"Enterprise Community",
				"Registered",  // Basic plan
				"Subscribed",  // Enterprise plan
				"Get Started", // Growth plan
			},
			notExpectedStrings:   []string{},
			basicButtonText:      "Registered",
			growthButtonText:     "Get Started",
			enterpriseButtonText: "Subscribed",
		},
		{
			name:                      "Logged in - Growth and Enterprise subscriptions",
			isLoggedIn:                true,
			hasSeedSubscription:       false,
			hasGrowthSubscription:     true,
			hasEnterpriseSubscription: true,
			expectedStrings: []string{
				"Plans and Pricing",
				"Basic Community",
				"Growth Community",
				"Enterprise Community",
				"Registered", // Basic plan
				"Subscribed", // Growth and Enterprise plans
			},
			notExpectedStrings: []string{
				"Get Started", // Should not appear since user has both Growth and Enterprise
			},
			basicButtonText:      "Registered",
			growthButtonText:     "Subscribed",
			enterpriseButtonText: "Subscribed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the PricingPage component
			pricingPage := PricingPage(tc.isLoggedIn, tc.hasSeedSubscription, tc.hasGrowthSubscription, tc.hasEnterpriseSubscription)

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
			// seedButtonCount := strings.Count(renderedContent, tc.seedButtonText) // Seed tier is currently commented out in pricing.templ
			growthButtonCount := strings.Count(renderedContent, tc.growthButtonText)
			enterpriseButtonCount := strings.Count(renderedContent, tc.enterpriseButtonText)

			// Basic button should appear once
			if basicButtonCount == 0 {
				t.Errorf("Expected Basic plan button text '%s' to appear, but it didn't", tc.basicButtonText)
			}

			// Seed button should appear once
			// Seed tier is currently commented out in pricing.templ
			// if seedButtonCount == 0 {
			// 	t.Errorf("Expected Seed plan button text '%s' to appear, but it didn't", tc.seedButtonText)
			// }

			// Growth button should appear once
			if growthButtonCount == 0 {
				t.Errorf("Expected Growth plan button text '%s' to appear, but it didn't", tc.growthButtonText)
			}

			// Enterprise button should appear once
			if enterpriseButtonCount == 0 {
				t.Errorf("Expected Enterprise plan button text '%s' to appear, but it didn't", tc.enterpriseButtonText)
			}

			// Verify plan IDs are in the JavaScript
			if !strings.Contains(renderedContent, "price_growth_test") {
				t.Error("Expected rendered content to contain Growth plan ID 'price_growth_test'")
			}
			// Note: Seed plan ID is still present in the template (for data attribute) even though Seed tier is hidden
			if !strings.Contains(renderedContent, "price_seed_test") {
				t.Error("Expected rendered content to contain Seed plan ID 'price_seed_test'")
			}
			if !strings.Contains(renderedContent, "price_enterprise_test") {
				t.Error("Expected rendered content to contain Enterprise plan ID 'price_enterprise_test'")
			}

			// Verify subscription status logic in rendered content
			// If user has Growth subscription, Growth should show "Subscribed"
			if tc.hasGrowthSubscription && !strings.Contains(renderedContent, "Subscribed") {
				t.Error("Expected 'Subscribed' button for Growth plan when user has Growth subscription")
			}

			// If user has Enterprise subscription, Enterprise should show "Subscribed"
			if tc.hasEnterpriseSubscription && !strings.Contains(renderedContent, "Subscribed") {
				t.Error("Expected 'Subscribed' button for Enterprise plan when user has Enterprise subscription")
			}

			// Seed tier subscription checks - currently commented out as Seed tier is hidden in pricing.templ
			// If user has Seed subscription, Seed should show "Subscribed"
			// if tc.hasSeedSubscription && !tc.hasGrowthSubscription && !strings.Contains(renderedContent, "Subscribed") {
			// 	t.Error("Expected 'Subscribed' button for Seed plan when user has Seed subscription")
			// }
		})
	}
}

func TestPricingPage_ButtonStates(t *testing.T) {
	// Set up environment variables
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalEnterprisePlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", originalEnterprisePlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", "price_growth_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", "price_seed_test")
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", "price_enterprise_test")
	os.Setenv("APEX_URL", "https://test.example.com")

	tests := []struct {
		name                       string
		isLoggedIn                 bool
		hasSeedSubscription        bool
		hasGrowthSubscription      bool
		hasEnterpriseSubscription  bool
		expectedBasicDisabled      bool
		expectedGrowthDisabled     bool
		expectedEnterpriseDisabled bool
	}{
		{
			name:                       "Not logged in - all buttons enabled",
			isLoggedIn:                 false,
			hasSeedSubscription:        false,
			hasGrowthSubscription:      false,
			hasEnterpriseSubscription:  false,
			expectedBasicDisabled:      false,
			expectedGrowthDisabled:     false,
			expectedEnterpriseDisabled: false,
		},
		{
			name:                       "Logged in - Basic disabled, others enabled",
			isLoggedIn:                 true,
			hasSeedSubscription:        false,
			hasGrowthSubscription:      false,
			hasEnterpriseSubscription:  false,
			expectedBasicDisabled:      true, // "Registered" button is disabled
			expectedGrowthDisabled:     false,
			expectedEnterpriseDisabled: false,
		},
		{
			name:                       "Logged in with Growth - Basic and Growth disabled, Enterprise enabled",
			isLoggedIn:                 true,
			hasSeedSubscription:        false,
			hasGrowthSubscription:      true,
			hasEnterpriseSubscription:  false,
			expectedBasicDisabled:      true,
			expectedGrowthDisabled:     true, // "Subscribed" button is disabled
			expectedEnterpriseDisabled: false,
		},
		{
			name:                       "Logged in with Enterprise - Basic and Enterprise disabled, Growth enabled",
			isLoggedIn:                 true,
			hasSeedSubscription:        false,
			hasGrowthSubscription:      false,
			hasEnterpriseSubscription:  true,
			expectedBasicDisabled:      true,
			expectedGrowthDisabled:     false,
			expectedEnterpriseDisabled: true,
		},
		{
			name:                       "Logged in with Growth and Enterprise - all buttons disabled",
			isLoggedIn:                 true,
			hasSeedSubscription:        false,
			hasGrowthSubscription:      true,
			hasEnterpriseSubscription:  true,
			expectedBasicDisabled:      true,
			expectedGrowthDisabled:     true,
			expectedEnterpriseDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricingPage := PricingPage(tt.isLoggedIn, tt.hasSeedSubscription, tt.hasGrowthSubscription, tt.hasEnterpriseSubscription)

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

			// Growth plan button
			// Check between Growth Community section and Enterprise Community section
			growthSectionStart := strings.Index(renderedContent, "Growth Community")
			enterpriseSectionStart := strings.Index(renderedContent, "Enterprise Community")
			if growthSectionStart != -1 && enterpriseSectionStart != -1 {
				growthSection := renderedContent[growthSectionStart:enterpriseSectionStart]
				growthButtonDisabled := strings.Contains(growthSection, "pointer-events-none")
				if growthButtonDisabled != tt.expectedGrowthDisabled {
					t.Errorf("Growth button disabled state: expected %v, but got %v", tt.expectedGrowthDisabled, growthButtonDisabled)
				}
			}

			// Enterprise plan button
			// Check after Enterprise Community section
			if enterpriseSectionStart != -1 {
				enterpriseSection := renderedContent[enterpriseSectionStart:]
				enterpriseButtonDisabled := strings.Contains(enterpriseSection, "pointer-events-none")
				if enterpriseButtonDisabled != tt.expectedEnterpriseDisabled {
					t.Errorf("Enterprise button disabled state: expected %v, but got %v", tt.expectedEnterpriseDisabled, enterpriseButtonDisabled)
				}
			}
		})
	}
}

func TestPricingPage_RendersPlanIDs(t *testing.T) {
	// Set up environment variables
	originalGrowthPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	originalSeedPlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	originalEnterprisePlan := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE")
	originalApexURL := os.Getenv("APEX_URL")

	defer func() {
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", originalGrowthPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", originalSeedPlan)
		os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", originalEnterprisePlan)
		os.Setenv("APEX_URL", originalApexURL)
	}()

	testGrowthPlan := "price_1234567890growth"
	testSeedPlan := "price_1234567890seed"
	testEnterprisePlan := "price_1234567890enterprise"
	testApexURL := "https://test.example.com"

	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH", testGrowthPlan)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_SEED", testSeedPlan)
	os.Setenv("STRIPE_SUBSCRIPTION_PLAN_ENTERPRISE", testEnterprisePlan)
	os.Setenv("APEX_URL", testApexURL)

	pricingPage := PricingPage(false, false, false, false)

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
