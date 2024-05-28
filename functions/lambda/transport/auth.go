package transport

import (
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/client"
	"github.com/clerk/clerk-sdk-go/v2/session"
	"github.com/clerk/clerk-sdk-go/v2/user"
)

type ClerkAuth struct {
	genericClient *client.Client
	userClient    *user.Client
	sessionClient *session.Client
}

func InitClerkAuth(config *clerk.ClientConfig) *ClerkAuth {
	genericClient := client.NewClient(config)
	userClient := user.NewClient(config)
	sessionClient := session.NewClient(config)

	return &ClerkAuth{
		genericClient: genericClient,
		userClient:    userClient,
		sessionClient: sessionClient,
	}
}
