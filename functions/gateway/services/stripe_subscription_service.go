package services

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/stripe/stripe-go/v83"
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
	growthPlanID, seedPlanID, enterprisePlanID := GetStripeSubscriptionPlanIDs()

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

	if enterprisePlanID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			plan, err := s.getPlanByID(enterprisePlanID)
			results <- planResult{plan: plan, err: err, name: "Enterprise"}
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

// getPlanByID fetches a specific subscription plan by its price ID or product ID
// Note: The environment variable contains price IDs, so we need to handle both cases
func (s *StripeSubscriptionService) getPlanByID(priceOrProductID string) (*types.SubscriptionPlan, error) {
	// First, try to retrieve it as a price
	priceObj, err := s.client.V1Prices.Retrieve(context.Background(), priceOrProductID, nil)
	if err != nil || priceObj == nil {
		// If it fails, it might be a product ID - try to fetch as product
		productObj, productErr := s.client.V1Products.Retrieve(context.Background(), priceOrProductID, nil)
		if productErr != nil || productObj == nil {
			return nil, fmt.Errorf("error fetching price/product %s: %w", priceOrProductID, err)
		}

		// Product was found, get its default price
		if productObj.DefaultPrice == nil || productObj.DefaultPrice.ID == "" {
			return nil, fmt.Errorf("product %s has no default price", priceOrProductID)
		}

		// Get the price details
		priceObj, err = s.client.V1Prices.Retrieve(context.Background(), productObj.DefaultPrice.ID, nil)
		if err != nil || priceObj == nil {
			return nil, fmt.Errorf("error fetching price %s: %w", productObj.DefaultPrice.ID, err)
		}

		// Convert to our subscription plan type
		plan := types.ConvertStripeProduct(productObj, priceObj)
		return plan, nil
	}

	// Price was found, now get the product
	if priceObj.Product == nil {
		return nil, fmt.Errorf("price %s has no associated product", priceOrProductID)
	}

	// Product.ID returns the product ID string
	productID := priceObj.Product.ID
	productObj, err := s.client.V1Products.Retrieve(context.Background(), productID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching product %s: %w", productID, err)
	}

	// Only include recurring prices (subscriptions)
	if priceObj.Recurring == nil {
		return nil, fmt.Errorf("price %s is not a recurring subscription", priceObj.ID)
	}

	// Convert to our subscription plan type
	plan := types.ConvertStripeProduct(productObj, priceObj)
	return plan, nil
}

// GetCustomerSubscriptions gets active subscriptions for a customer
func (s *StripeSubscriptionService) GetCustomerSubscriptions(customerID string) ([]*types.CustomerSubscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("all"), // Get all subscriptions regardless of status
	}

	ctx := context.Background()
	subs := s.client.V1Subscriptions.List(ctx, params)
	var subscriptions []*types.CustomerSubscription

	// Iterate over the sequence
	for sub, err := range subs {
		if err != nil {
			return nil, fmt.Errorf("error fetching subscriptions for customer %s: %w", customerID, err)
		}
		customerSub := types.ConvertStripeSubscription(sub)
		subscriptions = append(subscriptions, customerSub)
	}

	return subscriptions, nil
}

// CreateCustomerPortalSession creates a customer portal session for subscription management
// If subscriptionID is provided, creates a deep link to manage that specific subscription
// flowType can be: constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_CANCEL, STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE,
// STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE_CONFIRM, or STRIPE_PORTAL_FLOW_PAYMENT_METHOD_UPDATE
// If flowType is empty and subscriptionID is provided, defaults to STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE
func (s *StripeSubscriptionService) CreateCustomerPortalSession(customerID, returnURL string, subscriptionID, flowType string) (*types.CustomerPortalSession, error) {
	params := &stripe.BillingPortalSessionCreateParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	// Create flow data for deep linking:
	// 1. If subscriptionID is provided (for subscription-specific actions)
	// 2. If flowType is payment_method_update (customer-level action, doesn't need subscription ID)
	if subscriptionID != "" || flowType == constants.STRIPE_PORTAL_FLOW_PAYMENT_METHOD_UPDATE {
		// Default flow type if subscription ID provided but flow type not specified
		if subscriptionID != "" && flowType == "" {
			flowType = constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE
		}

		params.FlowData = &stripe.BillingPortalSessionCreateFlowDataParams{
			Type: stripe.String(flowType),
		}

		// Set subscription-specific flow data based on flow type
		switch flowType {
		case constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_CANCEL:
			if subscriptionID != "" {
				params.FlowData.SubscriptionCancel = &stripe.BillingPortalSessionCreateFlowDataSubscriptionCancelParams{
					Subscription: stripe.String(subscriptionID),
				}
			}
		case constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE:
			if subscriptionID != "" {
				params.FlowData.SubscriptionUpdate = &stripe.BillingPortalSessionCreateFlowDataSubscriptionUpdateParams{
					Subscription: stripe.String(subscriptionID),
				}
			}
		case constants.STRIPE_PORTAL_FLOW_SUBSCRIPTION_UPDATE_CONFIRM:
			if subscriptionID != "" {
				params.FlowData.SubscriptionUpdateConfirm = &stripe.BillingPortalSessionCreateFlowDataSubscriptionUpdateConfirmParams{
					Subscription: stripe.String(subscriptionID),
				}
			}
		case constants.STRIPE_PORTAL_FLOW_PAYMENT_METHOD_UPDATE:
			// payment_method_update doesn't need a subscription ID
			// Flow data type is sufficient - it will deep link to payment method update page
		}
	}

	session, err := s.client.V1BillingPortalSessions.Create(context.Background(), params)
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

	customers := s.client.V1Customers.Search(context.Background(), &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf("metadata['zitadel_user_id']:'%s'", externalID),
		},
	})
	for cust := range customers {
		if cust.Metadata != nil {
			if extID, exists := cust.Metadata["zitadel_user_id"]; exists && extID == externalID {
				return cust, nil
			}
		}
	}

	return nil, nil // Customer not found
}

// UpdateCustomerMetadata updates customer metadata with Zitadel user ID
func (s *StripeSubscriptionService) UpdateCustomerMetadata(customerID, externalID string) error {
	params := &stripe.CustomerUpdateParams{
		Metadata: map[string]string{
			"zitadel_user_id": externalID,
		},
	}

	_, err := s.client.V1Customers.Update(context.Background(), customerID, params)
	if err != nil {
		return fmt.Errorf("error updating customer metadata: %w", err)
	}

	return nil
}

// CreateCustomer creates a new Stripe customer with external_id metadata
func (s *StripeSubscriptionService) CreateCustomer(externalID, email, name string) (*stripe.Customer, error) {
	log.Printf("Creating Stripe customer with email: %s, name: %s, zitadel_user_id: %s", email, name, externalID)
	params := &stripe.CustomerCreateParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
		Metadata: map[string]string{
			"zitadel_user_id": externalID,
		},
	}

	cust, err := s.client.V1Customers.Create(context.Background(), params)
	if err != nil {
		log.Printf("Failed to create Stripe customer: %v", err)
		return nil, fmt.Errorf("error creating customer: %w", err)
	}
	log.Printf("Successfully created Stripe customer %s", cust.ID)

	return cust, nil
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
