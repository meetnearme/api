package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"
	"github.com/zitadel/zitadel-go/v3/pkg/zitadel"
)

type JWKS struct {
	Keys []json.RawMessage `json:"keys"`
}

var (
	domain        = flag.String("domain", os.Getenv("ZITADEL_INSTANCE_HOST"), "your ZITADEL instance domain (in the form: https://<instance>.zitadel.cloud or https://<yourdomain>)")
	key           = flag.String("key", os.Getenv("ZITADEL_ENCRYPTION_KEY"), "encryption key")
	clientID      = flag.String("clientID", os.Getenv("ZITADEL_CLIENT_ID"), "clientID provided by ZITADEL")
	clientSecret  = flag.String("clientSecret", os.Getenv("ZITADEL_CLIENT_SECRET"), "clientSecret provided by ZITADEL")
	jwtClientID   = flag.String("jwtClientID", os.Getenv("ZITADEL_JWT_CLIENT_ID"), "Client ID for JWT app for service user auth in ZITADEL")
	apiPvtKeyID   = flag.String("apiPvtKeyID", os.Getenv("API_PRIVATE_KEY_ID"), "Private key ID for JWT app for service user auth in ZITADEL")
	apiPvtKey     = flag.String("apiPvtKey", os.Getenv("API_PRIVATE_KEY"), "Private key ID for JWT app for service user auth in ZITADEL")
	projectID     = flag.String("projectID", os.Getenv("ZITADEL_PROJECT_ID"), "zitadel project ID for MeetNearMe")
	redirectURI   = flag.String("redirectURI", string(os.Getenv("APEX_URL")+"/auth/callback"), "redirect URI registered with ZITADEL")
	loginPageURI  = flag.String("loginPageURI", string(os.Getenv("APEX_URL")), "App login page URI")
	authorizeURI  = flag.String("authorizeURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_HOST")+"/oauth/v2/authorize"), "Zitadel authorizeURL")
	jwksURI       = flag.String("jwksURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_HOST")+"/oauth/v2/keys"), "Zitadel endpoint to get public keys")
	tokenURI      = flag.String("tokenURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_HOST")+"/oauth/v2/token"), "Zitadel endpoint to exchange code challenge and verifier for token")
	endSessionURI = flag.String("endSessionURI", string("https://"+os.Getenv("ZITADEL_INSTANCE_HOST")+"/oidc/v1/end_session"), "Zitadel logout URI")
	authZ         *authorization.Authorizer[*oauth.IntrospectionContext]
	once          sync.Once
)

func InitAuth() {
	once.Do(func() {
		ctx := context.Background()

		introspectionAuth := oauth.ClientIDSecretIntrospectionAuthentication(*clientID, *clientSecret)

		var err error
		authZ, err = authorization.New(ctx, zitadel.New(*domain), oauth.WithIntrospection[*oauth.IntrospectionContext](introspectionAuth))
		if err != nil {
			log.Fatalf("failed to initialize authorizer: %v", err)
		}
	})
}

func GetAuthMw() *authorization.Authorizer[*oauth.IntrospectionContext] {
	InitAuth()
	return authZ
}

// Extract and format roles from the claims.
func ExtractClaimsMeta(claims map[string]interface{}) ([]helpers.RoleClaim, map[string]interface{}) {
	var userMeta map[string]interface{}
	var roles []helpers.RoleClaim

	if metadataMap, ok := claims[helpers.AUTH_METADATA_KEY].(map[string]interface{}); ok {
		userMeta = metadataMap
	}

	roleKey := strings.Replace(helpers.AUTH_ROLE_CLAIMS_KEY, "<project-id>", *projectID, 1)
	// Check if the claims contain the specified key
	if roleMap, ok := claims[roleKey].(map[string]interface{}); ok {
		for role, projects := range roleMap {
			// Iterate over the project map for each role
			if projectMap, ok := projects.(map[string]interface{}); ok {
				for projectID, projectName := range projectMap {
					// Add the role, project ID, and project name to the list
					roles = append(roles, helpers.RoleClaim{
						Role:        role,
						ProjectID:   projectID,
						ProjectName: fmt.Sprintf("%v", projectName), // Ensure it's a string
					})
				}
			}
		}
	}

	return roles, userMeta
}

func RandomBytesInHex(count int) (string, error) {
	buf := make([]byte, count)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func GenerateCodeChallengeAndVerifier() (string, string, error) {
	codeVerifier, err := RandomBytesInHex(32) // Length in bytes
	if err != nil {
		return "", "", err
	}

	sha2 := sha256.New()
	io.WriteString(sha2, codeVerifier)

	codeChallenge := base64.RawURLEncoding.EncodeToString(sha2.Sum(nil))

	return codeChallenge, codeVerifier, nil
}

func BuildAuthorizeRequest(codeChallenge string, userRedirectURL string) (*url.URL, error) {
	authURL, err := url.Parse(*authorizeURI)
	if err != nil {
		return nil, err
	}

	query := authURL.Query()
	query.Set("client_id", *clientID)
	query.Set("redirect_uri", *redirectURI)
	query.Set("response_type", "code") // code for authorization grant
	query.Set("scope", "openid oidc profile email offline_access "+helpers.AUTH_METADATA_KEY)
	query.Set("code_challenge", codeChallenge)
	query.Set("code_challenge_method", "S256")
	query.Set("state", userRedirectURL)

	authURL.RawQuery = query.Encode()

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

	return result, nil
}

func RefreshAccessToken(refreshToken string) (map[string]interface{}, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("redirect_uri", *redirectURI)
	data.Set("client_id", *clientID)
	data.Set("client_secret", *clientSecret)

	resp, err := http.PostForm(*tokenURI, data)
	if err != nil {
		log.Printf("Failed to refresh access_token: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Handle the token response
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return result, nil
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	var redirectURL *url.URL
	redirectURLStr := r.URL.Query().Get(helpers.POST_LOGOUT_REDIRECT_URI_KEY)
	decodedRedirectURL, err := url.QueryUnescape(redirectURLStr)
	if err != nil {
		log.Printf("Failed to decode redirect URL: %v", err)
	}
	parsedRedirectURL, err := url.Parse(decodedRedirectURL)
	if err != nil {
		log.Printf("Failed to parse redirect URL: %v", err)
		// fallback to default logout URL
		redirectURL, err = url.Parse(*endSessionURI)
		if err != nil {
			log.Printf("Failed to parse Zitadel End Session URI: %v", err)
			return
		}
	} else {
		redirectURL = parsedRedirectURL
	}

	// Clear legacy cookies, we can remove this when we know cookies have expired for everyone
	// today is 2025-03-05, so we could remove this logic perhaps after 2025-06-05, depending on zitadel
	ClearSubdomainCookie(w, "access_token")
	ClearSubdomainCookie(w, "refresh_token")
	ClearSubdomainCookie(w, helpers.MNM_ID_TOKEN_COOKIE_NAME)

	// Clear application cookies
	ClearSubdomainCookie(w, helpers.PKCE_VERIFIER_COOKIE_NAME)
	ClearSubdomainCookie(w, helpers.MNM_ACCESS_TOKEN_COOKIE_NAME)
	ClearSubdomainCookie(w, helpers.MNM_REFRESH_TOKEN_COOKIE_NAME)

	// Get id_token_hint from cookies if available
	var idTokenHint string
	if c, err := r.Cookie(helpers.MNM_ID_TOKEN_COOKIE_NAME); err == nil && c != nil {
		idTokenHint = c.Value
	}

	postLogoutURI, err := url.Parse(*loginPageURI)
	if err != nil {
		log.Printf("Failed to parse Zitadel End Session URI: %v", err)
		return
	}
	logoutURL, err := url.Parse(*endSessionURI)
	if err != nil {
		log.Printf("Failed to parse End Session URI: %v", err)
		return
	}
	query := postLogoutURI.Query()
	query.Set(helpers.POST_LOGOUT_REDIRECT_URI_KEY, os.Getenv("APEX_URL"))
	if *clientID != "" {
		query.Set("client_id", *clientID)
	}
	// Create a proper state parameter that includes the final_redirect_uri
	stateValue := helpers.FINAL_REDIRECT_URI_KEY + "="
	if redirectURL != nil {
		stateValue += url.QueryEscape(redirectURL.String())
	} else {
		stateValue += url.QueryEscape(os.Getenv("APEX_URL"))
	}

	// Encode the state for security
	encodedState := base64.URLEncoding.EncodeToString([]byte(stateValue))

	// Set state in the query parameters
	query.Set("state", encodedState)

	// Add id_token_hint if available
	if idTokenHint != "" {
		query.Set("id_token_hint", idTokenHint)
	}

	logoutURL.RawQuery = query.Encode()
	http.Redirect(w, r, logoutURL.String(), http.StatusFound)
}

func ClearSubdomainCookie(w http.ResponseWriter, cookieName string) {
	SetSubdomainCookie(w, cookieName, "", true, 0)
}

func FetchJWKS() (*JWKS, error) {
	resp, err := http.Get(*jwksURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	return &jwks, nil
}

func GetPublicKey(jwks *JWKS, kid string) (*rsa.PublicKey, error) {
	for _, key := range jwks.Keys {
		var k map[string]interface{}
		if err := json.Unmarshal(key, &k); err != nil {
			return nil, err
		}

		if k["kid"] == kid && k["kty"] == "RSA" {
			nStr, _ := k["n"].(string)
			eStr, _ := k["e"].(string)

			nBytes, err := jwt.DecodeSegment(nStr)
			if err != nil {
				return nil, err
			}

			eBytes, err := jwt.DecodeSegment(eStr)
			if err != nil {
				return nil, err
			}

			e := 0
			for _, b := range eBytes {
				e = e*256 + int(b)
			}

			pubKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(nBytes),
				E: e,
			}

			return pubKey, nil
		}
	}

	return nil, fmt.Errorf("no matching RSA key found in JWKS")
}

func SetSubdomainCookie(w http.ResponseWriter, cookieName string, cookieValue string, clearing bool, ttl int) {
	apexDomain := os.Getenv("APEX_URL")
	if apexDomain == "" {
		log.Print("ERR: APEX_URL is not set, cannot set cookies")
		return
	}

	subdomainCookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
	}

	if apexDomain == "http://localhost:8000" {
		apexDomain = strings.Replace(apexDomain, "http://", "", 1)
		subdomainCookie.Secure = false
		subdomainCookie.Domain = ""
	} else {
		apexDomain = strings.Replace(apexDomain, "https://", "", 1)
		subdomainCookie.Secure = true
		subdomainCookieWildcard := "." + apexDomain
		subdomainCookie.Domain = subdomainCookieWildcard
	}

	if clearing {
		subdomainCookie.Expires = time.Unix(0, 0)
		subdomainCookie.Value = ""
		if ttl > 0 {
			subdomainCookie.MaxAge = ttl
		} else {
			subdomainCookie.MaxAge = 0
		}
	}

	http.SetCookie(w, subdomainCookie)
}

