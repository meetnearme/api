package services

import (
	"os"
	"sync"

	"github.com/stripe/stripe-go/v83"
)

var sc *stripe.Client
var stripeOnce sync.Once

func InitStripe() {
	stripeOnce.Do(func() {
		_, priv := GetStripeKeyPair()
		sc = stripe.NewClient(priv)
	})
}

func GetStripeKeyPair() (publishableKey string, privateKey string) {
	return os.Getenv("STRIPE_PUBLISHABLE_KEY"), os.Getenv("STRIPE_SECRET_KEY")
}

func GetStripeClient() *stripe.Client {
	return sc
}

// ResetStripeClient resets the Stripe client (useful for testing)
func ResetStripeClient() {
	_, priv := GetStripeKeyPair()
	sc = stripe.NewClient(priv)
}

func GetStripeCheckoutWebhookSecret() string {
	return os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
}

// GetStripeSubscriptionPlanIDs returns the subscription plan IDs for the current environment
func GetStripeSubscriptionPlanIDs() (growthPlanID, seedPlanID string) {
	growthPlanID = os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	seedPlanID = os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	return growthPlanID, seedPlanID
}
