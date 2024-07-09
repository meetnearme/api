package services

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	// flags to be provided for running the example server
	domain      = os.Getenv("ZITADEL_INSTANCE_URL")
	key         = os.Getenv("ZITADEL_ENCRYPTION_KEY")
	clientID    = os.Getenv("ZITADEL_CLIENT_ID")
	redirectURI = os.Getenv("ZITADEL_REDIRECT_URI")
	mw   *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	authN *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	once sync.Once
)

func init() {
	once.Do(initMW)
}

func initMW() {
	ctx := context.Background()

	// Initiate the authentication by providing a zitadel configuration and handler.
	// This example will use OIDC/OAuth2 PKCE Flow, therefore you will also need to initialize that with the generated client_id:
	var err	error
	authN, err = authentication.New(ctx, zitadel.New(domain), key,
		openid.DefaultAuthentication(clientID, redirectURI, key),
	)
	if err != nil {
		log.Println("zitadel sdk could not initialize:" + err.Error())
		return
	}

	// Initialize the middleware by providing the sdk
	mw = authentication.Middleware(authN)
	if mw == nil {
		log.Println("middleware is nil")
	}
	if authN == nil {
		log.Println("authN is nil")
	} else {
			log.Println("middleware and authenticator initialized successfully")
	}
}

func GetAuthMw() (*authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]], *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]) {
	return mw, authN
}
