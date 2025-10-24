package services

import (
	"fmt"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/billingportal/session"
	"github.com/stripe/stripe-go/v83/customer"
	"github.com/stripe/stripe-go/v83/price"
	"github.com/stripe/stripe-go/v83/product"
	"github.com/stripe/stripe-go/v83/subscription"
)

// StripeSubscriptionService implements the StripeSubscriptionServiceInterface
type StripeSubscriptionService struct {
	client *stripe.Client
}

// NewStripeSubscriptionService creates a new instance of StripeSubscriptionService
func NewStripeSubscriptionService() interfaces.StripeSubscriptionServiceInterface {
	return &StripeSubscriptionService{
		client: GetStripeClient(),
	}
}

// GetSubscriptionPlans fetches the specific subscription plans configured in environment variables
func (s *StripeSubscriptionService) GetSubscriptionPlans() ([]*types.SubscriptionPlan, error) {
	// Get the specific plan IDs from environment variables
	growthPlanID, seedPlanID := GetStripeSubscriptionPlanIDs()

	// Use channels for parallel fetching
	type planResult struct {
		plan *types.SubscriptionPlan
		err  error
		name string
	}

	results := make(chan planResult, 2)
	var wg sync.WaitGroup

	// Fetch Growth plan in parallel
	if growthPlanID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan, err := s.getPlanByID(growthPlanID)
			results <- planResult{plan: plan, err: err, name: "Growth"}
		}()
	}

	// Fetch Seed plan in parallel
	if seedPlanID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan, err := s.getPlanByID(seedPlanID)
			results <- planResult{plan: plan, err: err, name: "Seed"}
		}()
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	var plans []*types.SubscriptionPlan
	var errors []error

	// Collect results from all goroutines
	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Errorf("error fetching %s plan: %w", result.name, result.err))
		} else if result.plan != nil {
			plans = append(plans, result.plan)
		}
	}

	// Return error if any plan failed to fetch
	if len(errors) > 0 {
		return nil, fmt.Errorf("failed to fetch subscription plans: %v", errors)
	}

	return plans, nil
}

// getPlanByID fetches a specific subscription plan by its price ID
func (s *StripeSubscriptionService) getPlanByID(priceID string) (*types.SubscriptionPlan, error) {
	// Get the price details
	price, err := price.Get(priceID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching price %s: %w", priceID, err)
	}

	// Only include recurring prices (subscriptions)
	if price.Recurring == nil {
		return nil, fmt.Errorf("price %s is not a recurring subscription", priceID)
	}

	// Get the product details
	product, err := product.Get(price.Product.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching product %s: %w", price.Product.ID, err)
	}

	// Convert to our subscription plan type
	plan := types.ConvertStripeProduct(product, price)
	return plan, nil
}

// GetCustomerSubscriptions gets active subscriptions for a customer
func (s *StripeSubscriptionService) GetCustomerSubscriptions(customerID string) ([]*types.CustomerSubscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("all"), // Get all subscriptions regardless of status
	}

	iter := subscription.List(params)
	var subscriptions []*types.CustomerSubscription

	for iter.Next() {
		sub := iter.Subscription()
		customerSub := types.ConvertStripeSubscription(sub)
		subscriptions = append(subscriptions, customerSub)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error fetching subscriptions for customer %s: %w", customerID, err)
	}

	return subscriptions, nil
}

// CreateCustomerPortalSession creates a customer portal session for subscription management
func (s *StripeSubscriptionService) CreateCustomerPortalSession(customerID, returnURL string) (*types.CustomerPortalSession, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("error creating customer portal session: %w", err)
	}

	return &types.CustomerPortalSession{
		ID:        session.ID,
		URL:       session.URL,
		ReturnURL: returnURL,
	}, nil
}

// SearchCustomerByExternalID searches for a customer by Zitadel user ID in metadata
func (s *StripeSubscriptionService) SearchCustomerByExternalID(externalID string) (*stripe.Customer, error) {
	// For now, we'll use a simple approach - list customers and filter by metadata
	// This is not ideal for production but works for the current implementation
	// TODO: Implement proper customer search when Stripe API supports it
	params := &stripe.CustomerListParams{}

	iter := customer.List(params)
	for iter.Next() {
		customer := iter.Customer()
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["external_id"]; exists && extID == externalID {
				return customer, nil
			}
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error searching for customer with external_id %s: %w", externalID, err)
	}

	return nil, nil // Customer not found
}

// UpdateCustomerMetadata updates customer metadata with Zitadel user ID
func (s *StripeSubscriptionService) UpdateCustomerMetadata(customerID, externalID string) error {
	params := &stripe.CustomerParams{
		Metadata: map[string]string{
			"external_id": externalID,
		},
	}

	_, err := customer.Update(customerID, params)
	if err != nil {
		return fmt.Errorf("error updating customer metadata: %w", err)
	}

	return nil
}

// CreateCustomer creates a new Stripe customer with external_id metadata
func (s *StripeSubscriptionService) CreateCustomer(externalID, email, name string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"external_id": externalID,
		},
	}

	customer, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("error creating customer: %w", err)
	}

	return customer, nil
}

// GetOrCreateCustomerByExternalID gets an existing customer or creates a new one
func (s *StripeSubscriptionService) GetOrCreateCustomerByExternalID(externalID, email, name string) (*stripe.Customer, error) {
	// First, try to find existing customer
	existingCustomer, err := s.SearchCustomerByExternalID(externalID)
	if err != nil {
		return nil, err
	}

	if existingCustomer != nil {
		return existingCustomer, nil
	}

	// Customer doesn't exist, create a new one
	return s.CreateCustomer(externalID, email, name)
}
