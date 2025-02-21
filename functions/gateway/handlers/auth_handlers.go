package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/meetnearme/api/functions/gateway/services"
)

var codeChallenge, codeVerifier, err = services.GenerateCodeChallengeAndVerifier()

func HandleLogin(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	queryParams := r.URL.Query()
	redirectQueryParam := queryParams.Get("redirect")

	apexURL := os.Getenv("APEX_URL")
	if apexURL == "" {
		http.Error(w, "APEX_URL not configured", http.StatusInternalServerError)
		return http.HandlerFunc(nil)
	}

	authURL, err := services.BuildAuthorizeRequest(codeChallenge, redirectQueryParam)
	if err != nil {
		http.Error(w, "Failed to authorize request", http.StatusBadRequest)
		return http.HandlerFunc(nil)
	}

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

	var userRedirectURL string = "/"
	if appState != "" {
		userRedirectURL = appState
	}

	// Store tokens in cookies
	subdomainAccessToken, apexAccessToken := services.GetContextualCookie("access_token", accessToken)
	subdomainRefreshToken, apexRefreshToken := services.GetContextualCookie("refresh_token", refreshToken)
	http.SetCookie(w, subdomainAccessToken)
	http.SetCookie(w, apexAccessToken)
	http.SetCookie(w, subdomainRefreshToken)
	http.SetCookie(w, apexRefreshToken)

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, userRedirectURL, http.StatusFound)
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services.HandleLogout(w, r)
	}
}
