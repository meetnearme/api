package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/meetnearme/api/functions/gateway/services"
)

var codeChallenge, codeVerifier, err = services.GenerateCodeChallengeAndVerifier()

func HandleLogin(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	queryParams := r.URL.Query()
	redirectUser := queryParams.Get("redirect")
	authURL, err := services.BuildAuthorizeRequest(codeChallenge, redirectUser)
	if err != nil {
		http.Error(w, "Failed to authorize request", http.StatusBadRequest)
		return http.HandlerFunc(nil)
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
	return http.HandlerFunc(nil)
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

	http.SetCookie(w, &http.Cookie{
		Name:  "access_token",
		Value: accessToken,
		Path:  "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:  "refresh_token",
		Value: refreshToken,
		Path:  "/",
	})

	var userRedirectURL string = "/"
	if appState != "" {
		userRedirectURL = appState
	}
	http.Redirect(w, r, userRedirectURL, http.StatusFound)
	return http.HandlerFunc(nil)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	services.HandleLogout(w, r)
	return http.HandlerFunc(nil)
}
