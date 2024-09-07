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

	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

var (
	domain        = flag.String("domain", os.Getenv("ZITADEL_INSTANCE_URL"), "your ZITADEL instance domain (in the form: https://<instance>.zitadel.cloud or https://<yourdomain>)")
	key           = flag.String("key", os.Getenv("ZITADEL_ENCRYPTION_KEY"), "encryption key")
	clientID      = flag.String("clientID", os.Getenv("ZITADEL_CLIENT_ID"), "clientID provided by ZITADEL")
	clientSecret  = flag.String("clientSecret", os.Getenv("ZITADEL_CLIENT_SECRET"), "clientSecret provided by ZITADEL")
	redirectURI   = flag.String("redirectURI", string(os.Getenv("APEX_URL")+"/auth/callback"), "redirect URI registered with ZITADEL")
	authorizeURI  = flag.String("authorizeURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_URL")+"/oauth/v2/authorize"), "Zitadel authorizeURL")
	tokenURI      = flag.String("tokenURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_URL")+"/oauth/v2/token"), "Zitadel endpoint to exchange code challenge and verifier for token")
	endSessionURI = flag.String("endSessionURI", string(os.Getenv("ZITADEL_INSTANCE_URL")+"/oidc/v1/end_session"), "Zitadel logout URI")
	authZ         *authorization.Authorizer[*oauth.IntrospectionContext]
	once          sync.Once
)

func InitAuth() {
	once.Do(func() {
		ctx := context.Background()

		// Initialize introspection authentication using client secret
		introspectionAuth := oauth.ClientIDSecretIntrospectionAuthentication(*clientID, *clientSecret)

		// Initialize the authZ with introspection authentication
		var err error
		authZ, err = authorization.New(ctx, zitadel.New(*domain), oauth.WithIntrospection[*oauth.IntrospectionContext](introspectionAuth))
		if err != nil {
			log.Fatalf("failed to initialize authorizer: %v", err)
		}
	})
}

// // AuthMiddleware checks the access token in cookies and introspects it.
// func AuthMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// Get the access token from cookies
// 		cookie, err := r.Cookie("access_token")
// 		if err != nil {
// 			http.Error(w, "Unauthorized: No access token", http.StatusUnauthorized)
// 			return
// 		}
// 		accessToken := cookie.Value

// 		// Use the Authorizer to introspect the access token
// 		authCtx, err := authZ.CheckAuthorization(r.Context(), accessToken)
// 		if err != nil {
// 			http.Error(w, "Unauthorized: Invalid access token", http.StatusUnauthorized)
// 			return
// 		}
// 	})
// }

func GetAuthMw() *authorization.Authorizer[*oauth.IntrospectionContext] {
	InitAuth()
	return authZ
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
