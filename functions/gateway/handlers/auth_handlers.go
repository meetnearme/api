package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/meetnearme/api/functions/gateway/services"
)

var codeChallenge, codeVerifier, err = services.GenerateCodeChallengeAndVerifier()

func HandleLogin(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	queryParams := r.URL.Query()
	redirectQueryParam := queryParams.Get("redirect")
	log.Printf("399: Redirect user: %s", redirectQueryParam)
	log.Printf("400: Query params: %v", queryParams)
	// Extract subdomain from host
	host := r.Host
	parts := strings.Split(host, ".")
	var subdomain string

	apexURL := os.Getenv("APEX_URL")
	if apexURL == "" {
		http.Error(w, "APEX_URL not configured", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	parsedApex, err := url.Parse(apexURL)
	if err != nil {
		http.Error(w, "Invalid APEX_URL", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	baseDomain := strings.Split(parsedApex.Host, ".")
	log.Printf("410: Base domain: %v", baseDomain)
	if len(baseDomain) < 2 {
		http.Error(w, "Invalid APEX_URL format", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	// Find where the base domain starts
	baseIndex := len(parts)
	log.Printf("415: Base index: %v", baseIndex)
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == baseDomain[0] { // This will match "example" from example.com
			baseIndex = i
			break
		}
	}

	if baseIndex > 0 {
		subdomain = strings.Join(parts[:baseIndex], ".")
		log.Printf("420: Subdomain: %v", subdomain)
	}

	authURL, err := services.BuildAuthorizeRequest(codeChallenge, redirectQueryParam)
	if err != nil {
		http.Error(w, "Failed to authorize request", http.StatusBadRequest)
		return http.HandlerFunc(nil)
	}
	log.Printf("425: Auth URL: %v", authURL)

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, authURL.String(), http.StatusFound)
	}
}

func HandleCallback(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	sessionId := r.URL.Query().Get("id")
	appState := r.URL.Query().Get("state")

	if sessionId != "" {
		location := r.Header.Get("Location")
		http.Redirect(w, r, location, http.StatusFound)
		return http.HandlerFunc(nil)
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code is missing", http.StatusBadRequest)
		return http.HandlerFunc(nil)
	}

	tokens, err := services.GetAuthToken(code, codeVerifier)
	if err != nil {
		log.Printf("Authentication Failed: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return http.HandlerFunc(nil)
	}

	// Store the access token and refresh token securely
	accessToken, ok := tokens["access_token"].(string)
	if !ok {
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	refreshToken, ok := tokens["refresh_token"].(string)
	if !ok {
		fmt.Printf("Refresh token error: %v", ok)
		http.Error(w, "Failed to get refresh token", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	log.Printf("430: App state: %v", appState)
	log.Printf("431: request URL sent to callback: %v", r.URL)
	var userRedirectURL string = "/"
	var cookieDomain string = ""

	if appState != "" {
		userRedirectURL = appState
		// Parse the redirect URL to get the host
		if parsedURL, err := url.Parse(appState); err == nil && parsedURL.Host != "" {
			cookieDomain = parsedURL.Host
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "access_token",
		Value:  accessToken,
		Path:   "/",
		Domain: cookieDomain,
	})

	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		Value:  refreshToken,
		Path:   "/",
		Domain: cookieDomain,
	})

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, userRedirectURL, http.StatusFound)
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services.HandleLogout(w, r)
	}
}
