package services

import (
	"os"

	"github.com/stripe/stripe-go/v80/client"
)

var sc = &client.API{}

func InitStripe() {
	once.Do(func() {
		_, priv := GetStripeKeyPair()
		sc.Init(priv, nil) // the second parameter overrides the backends used if needed for mocking
	})
}

func GetStripeKeyPair() (publishableKey string, privateKey string) {
	return os.Getenv("STRIPE_PUBLISHABLE_KEY"), os.Getenv("STRIPE_SECRET_KEY")
}

func GetStripeClient() *client.API {
	return sc
}

func GetStripeCheckoutWebhookSecret() string {
	return os.Getenv("STRIPE_CHECKOUT_WEBHOOK_SECRET")
}
