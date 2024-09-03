package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"sync"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	domain      = flag.String("domain", os.Getenv("ZITADEL_INSTANCE_URL"), "your ZITADEL instance domain (in the form: https://<instance>.zitadel.cloud or https://<yourdomain>)")
	key         = flag.String("key", os.Getenv("ZITADEL_ENCRYPTION_KEY"), "encryption key")
	clientID    = flag.String("clientID", os.Getenv("ZITADEL_CLIENT_ID"), "clientID provided by ZITADEL")
	redirectURI = flag.String("redirectURI", string(os.Getenv("APEX_URL")+"/auth/callback"), "redirect URI registered with ZITADEL")
	mw          *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	authN       *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	once        sync.Once
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

func randomBytesInHex(count int) (string, error) {
	buf := make([]byte, count)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

// GenerateCodeVerifier generates a cryptographically secure random string
func GenerateCodeVerifier() (string, error) {
	verifier, err := randomBytesInHex(32) // Length in bytes
	if err != nil {
		return "", err
	}
	return verifier, nil
	// Encode with URL encoding and remove padding
}

// GenerateCodeChallenge generates a code challenge from the code verifier
func GenerateCodeChallenge(codeVerifier string) string {
	sha2 := sha256.New()
	io.WriteString(sha2, codeVerifier)
	codeChallenge := base64.RawURLEncoding.EncodeToString(sha2.Sum(nil))
	return codeChallenge
}
