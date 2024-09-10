package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/meetnearme/api/functions/gateway/services"
)

var codeChallenge, codeVerifier, err = services.GenerateCodeChallengeAndVerifier()

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	redirectUser := queryParams.Get("redirect")
	log.Printf("Redirect to: %v", redirectUser)
	authURL, err := services.BuildAuthorizeRequest(codeChallenge, redirectUser)
	if err != nil {
		http.Error(w, "Failed to authorize request", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func HandleCallback(w http.ResponseWriter, r *http.Request) {
	sessionId := r.URL.Query().Get("id")
	appState := r.URL.Query().Get("state")

	if sessionId != "" {
		location := r.Header.Get("Location")
		http.Redirect(w, r, location, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code is missing", http.StatusBadRequest)
		return
	}

	tokens, err := services.GetAuthToken(code, codeVerifier)
	if err != nil {
		log.Printf("Authentication Failed: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Store the access token and refresh token securely
	accessToken, ok := tokens["access_token"].(string)
	if !ok {
		http.Error(w, "Failed to get access token", http.StatusInternalServerError)
		return
	}

	refreshToken, ok := tokens["refresh_token"].(string)
	if !ok {
		fmt.Printf("Refresh token error: %v", ok)
		http.Error(w, "Failed to get refresh token", http.StatusInternalServerError)
		return
	}

	// Store tokens in a session or secure cookie
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
}
