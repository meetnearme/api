package types

import (
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/stripe/stripe-go/v83"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive            SubscriptionStatus = "active"
	SubscriptionStatusCanceled          SubscriptionStatus = "canceled"
	SubscriptionStatusIncomplete        SubscriptionStatus = "incomplete"
	SubscriptionStatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	SubscriptionStatusPastDue           SubscriptionStatus = "past_due"
	SubscriptionStatusPaused            SubscriptionStatus = "paused"
	SubscriptionStatusTrialing          SubscriptionStatus = "trialing"
	SubscriptionStatusUnpaid            SubscriptionStatus = "unpaid"
)

// CustomerSubscription wraps Stripe subscription data for our application
type CustomerSubscription struct {
	ID                 string             `json:"id"`
	CustomerID         string             `json:"customer_id"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
	CancelAt           *time.Time         `json:"cancel_at,omitempty"`   // When the subscription will be canceled
	CanceledAt         *time.Time         `json:"canceled_at,omitempty"` // When the subscription was canceled
	PlanID             string             `json:"plan_id"`
	PlanName           string             `json:"plan_name"`
	PlanAmount         int64              `json:"plan_amount"`
	PlanCurrency       string             `json:"plan_currency"`
	PlanInterval       string             `json:"plan_interval"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Created int64                  `json:"created"`
	Data    map[string]interface{} `json:"data"`
}

// SubscriptionPlan represents a subscription plan from Stripe
type SubscriptionPlan struct {
	ID            string            `json:"id"`       // Product ID
	PriceID       string            `json:"price_id"` // Price ID for checkout
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Amount        int64             `json:"amount"`
	Currency      string            `json:"currency"`
	Interval      string            `json:"interval"`
	IntervalCount int64             `json:"interval_count"`
	Active        bool              `json:"active"`
	Metadata      map[string]string `json:"metadata"`
}

// CustomerPortalSession represents a Stripe customer portal session
type CustomerPortalSession struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	ReturnURL string `json:"return_url"`
}

// SubscriptionWebhookPayload represents the payload structure for subscription webhooks
type SubscriptionWebhookPayload struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Type    string `json:"type"`
	Created int64  `json:"created"`
	Data    struct {
		Object map[string]interface{} `json:"object"`
	} `json:"data"`
}

// ConvertStripeSubscription converts a Stripe subscription to our CustomerSubscription type
func ConvertStripeSubscription(stripeSub *stripe.Subscription) *CustomerSubscription {
	subscription := &CustomerSubscription{
		ID:                stripeSub.ID,
		CustomerID:        stripeSub.Customer.ID,
		Status:            SubscriptionStatus(stripeSub.Status),
		CancelAtPeriodEnd: stripeSub.CancelAtPeriodEnd,
		CreatedAt:         time.Unix(stripeSub.Created, 0),
		UpdatedAt:         time.Unix(stripeSub.Created, 0), // Use Created as fallback since Updated doesn't exist
	}

	// Handle cancel_at (scheduled cancellation timestamp)
	if stripeSub.CancelAt > 0 {
		cancelAt := time.Unix(stripeSub.CancelAt, 0)
		subscription.CancelAt = &cancelAt
	}

	// Handle canceled_at (actual cancellation timestamp)
	if stripeSub.CanceledAt > 0 {
		canceledAt := time.Unix(stripeSub.CanceledAt, 0)
		subscription.CanceledAt = &canceledAt
	}

	// Extract plan information from the first item
	if len(stripeSub.Items.Data) > 0 {
		item := stripeSub.Items.Data[0]

		// Get current period information from the subscription item
		subscription.CurrentPeriodStart = time.Unix(item.CurrentPeriodStart, 0)
		subscription.CurrentPeriodEnd = time.Unix(item.CurrentPeriodEnd, 0)

		if item.Price != nil {
			subscription.PlanID = item.Price.ID
			subscription.PlanAmount = item.Price.UnitAmount
			subscription.PlanCurrency = string(item.Price.Currency)
			subscription.PlanInterval = string(item.Price.Recurring.Interval)

			// Get plan name from product
			if item.Price.Product != nil {
				subscription.PlanName = item.Price.Product.Name
			}
		}
	}

	return subscription
}

// ConvertStripeProduct converts a Stripe product to our SubscriptionPlan type
func ConvertStripeProduct(product *stripe.Product, price *stripe.Price) *SubscriptionPlan {
	plan := &SubscriptionPlan{
		ID:          product.ID, // Product ID
		PriceID:     price.ID,   // Price ID for checkout
		Name:        product.Name,
		Description: product.Description,
		Amount:      price.UnitAmount,
		Currency:    string(price.Currency),
		Active:      product.Active,
		Metadata:    product.Metadata,
	}

	if price.Recurring != nil {
		plan.Interval = string(price.Recurring.Interval)
		plan.IntervalCount = price.Recurring.IntervalCount
	}

	return plan
}

// IsActive returns true if the subscription is currently active
func (s *CustomerSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsCanceled returns true if the subscription is canceled
func (s *CustomerSubscription) IsCanceled() bool {
	return s.Status == SubscriptionStatusCanceled
}

// IsPastDue returns true if the subscription is past due
func (s *CustomerSubscription) IsPastDue() bool {
	return s.Status == SubscriptionStatusPastDue
}

// IsScheduledToCancel returns true if the subscription is scheduled to cancel
// This checks both cancel_at_period_end (boolean) and cancel_at (timestamp)
func (s *CustomerSubscription) IsScheduledToCancel() bool {
	return s.CancelAtPeriodEnd || (s.CancelAt != nil && s.CancelAt.After(time.Now()))
}

// GetZitadelRole returns the corresponding Zitadel role for this subscription plan
func (s *CustomerSubscription) GetZitadelRole() string {
	// Map subscription plans to Zitadel roles based on plan name
	switch s.PlanName {
	case "Growth":
		return constants.Roles[constants.SubGrowth]
	case "Seed Community":
		return constants.Roles[constants.SubSeed]
	default:
		return constants.Roles[constants.SubGrowth] // Default to Growth role
	}
}
