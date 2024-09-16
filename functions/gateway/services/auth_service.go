package services

import (
	"context"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	domain      = flag.String("domain", os.Getenv("ZITADEL_INSTANCE_HOST"), "your ZITADEL instance domain (in the form: https://<instance>.zitadel.cloud or https://<yourdomain>)")
	key         = flag.String("key", os.Getenv("ZITADEL_ENCRYPTION_KEY"), "encryption key")
	clientID    = flag.String("clientID", os.Getenv("ZITADEL_CLIENT_ID"), "clientID provided by ZITADEL")
	redirectURI = flag.String("redirectURI", string( os.Getenv("APEX_URL") + "/auth/callback"), "redirect URI registered with ZITADEL")
	mw   *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	authN *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	once sync.Once
)

func InitAuth() {
	once.Do(func() {
		ctx := context.Background()

		log.Printf("Initializing authentication with domain: %s, clientID: %s, redirectURI: %s", *domain, *clientID, *redirectURI)

		var err error
		authN, err = authentication.New(ctx, zitadel.New(*domain), *key,
			openid.DefaultAuthentication(*clientID, *redirectURI, *key),
		)
		if err != nil {
			log.Printf("Failed to initialize Zitadel authentication: %v", err)
			return
		}

		mw = authentication.Middleware[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]](authN)

		if mw == nil {
			log.Println("Warning: middleware (mw) is nil after initialization")
		}

		if authN == nil {
			log.Println("Warning: authenticator (authN) is nil after initialization")
		}
	})
}

func GetAuthMw() (*authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]], *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]) {
	InitAuth()
	return mw, authN
}
