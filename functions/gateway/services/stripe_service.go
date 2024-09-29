package services

import (
	"os"

	"github.com/stripe/stripe-go/v79/client"
)

var sc = &client.API{}

func InitStripe() {
	once.Do(func() {
		_, priv := GetStripeKeyPair()
		sc.Init(priv, nil) // the second parameter overrides the backends used if needed for mocking
	})
}

func GetStripeKeyPair() (publishableKey string, privateKey string) {
	sstStage := os.Getenv("SST_STAGE")
	if sstStage == "prod" {
		return os.Getenv("PROD_STRIPE_PUBLISHABLE_KEY"), os.Getenv("PROD_STRIPE_SECRET_KEY")
	} else {
		return os.Getenv("DEV_STRIPE_PUBLISHABLE_KEY"), os.Getenv("DEV_STRIPE_SECRET_KEY")
	}
}

func GetStripeClient() *client.API {
	return sc
}
