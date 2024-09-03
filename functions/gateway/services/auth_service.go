package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	domain        = flag.String("domain", os.Getenv("ZITADEL_INSTANCE_URL"), "your ZITADEL instance domain (in the form: https://<instance>.zitadel.cloud or https://<yourdomain>)")
	key           = flag.String("key", os.Getenv("ZITADEL_ENCRYPTION_KEY"), "encryption key")
	clientID      = flag.String("clientID", os.Getenv("ZITADEL_CLIENT_ID"), "clientID provided by ZITADEL")
	redirectURI   = flag.String("redirectURI", string(os.Getenv("APEX_URL")+"/auth/callback"), "redirect URI registered with ZITADEL")
	authorizeURI  = flag.String("authorizeURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_URL")+"/oauth/v2/authorize"), "Zitadel authorizeURL")
	tokenURI      = flag.String("tokenURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_URL")+"/oauth/v2/token"), "Zitadel endpoint to exchange code challenge and verifier for token")
	endSessionURI = flag.String("endSessionURI", string(os.Getenv("ZITADEL_INSTANCE_URL")+"/oidc/v1/end_session"), "Zitadel logout URI")
	mw            *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	authN         *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	once          sync.Once
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

func GenerateCodeChallengeAndVerifier() (string, string, error) {
	codeVerifier, err := randomBytesInHex(32) // Length in bytes
	if err != nil {
		return "", "", err
	}

	sha2 := sha256.New()
	io.WriteString(sha2, codeVerifier)

	codeChallenge := base64.RawURLEncoding.EncodeToString(sha2.Sum(nil))

	return codeChallenge, codeVerifier, nil
}

func BuildAuthorizeRequest(codeChallenge string) (*url.URL, error) {
	authURL, err := url.Parse(*authorizeURI)
	if err != nil {
		log.Printf("Failed to parse Zitadel authorize URI: %v", err)
		return nil, err
	}

	query := authURL.Query()
	query.Set("client_id", *clientID)
	query.Set("redirect_uri", *redirectURI)
	query.Set("response_type", "code") // 'code' for authorization code grant
	query.Set("scope", "oidc profile email offline_access")
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")

	authURL.RawQuery = query.Encode()
	log.Printf("Auth URL: %v", authURL)

	return authURL, nil
}

func GetAuthToken(code string, codeVerifier string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", *redirectURI)
	data.Set("client_id", *clientID)
	data.Set("code_verifier", codeVerifier)

	resp, err := http.PostForm(*tokenURI, data)
	if err != nil {
		log.Printf("Failed to get tokens: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Handle the token response
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Handle the token response
	log.Printf("Tokens: %v", result)

	return result, nil
}
