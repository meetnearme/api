package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	queryParams := r.URL.Query()
	redirectQueryParam := queryParams.Get("redirect")

	apexURL := os.Getenv("APEX_URL")
	if apexURL == "" {
		log.Println("APEX_URL not configured")
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("APEX_URL not configured"), http.StatusInternalServerError, false)
		}
	}

	codeChallenge, codeVerifier, err := services.GenerateCodeChallengeAndVerifier()
	if err != nil {
		log.Println("Failed to generate code challenge and verifier", err)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Failed to generate code challenge and verifier"), http.StatusInternalServerError, false)
		}
	}

	services.SetSubdomainCookie(w, helpers.PKCE_VERIFIER_COOKIE_NAME, codeVerifier, false, 600)

	authURL, err := services.BuildAuthorizeRequest(codeChallenge, redirectQueryParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to authorize request: %+v", err)
		log.Println(msg)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte(msg), http.StatusBadRequest, false)(w, r)
		}
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
		return func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, location, http.StatusFound)
		}
	}

	// NOTE: We get the code verifier from stored cookies beacuse
	// distributed ephemeral lambda environments can have a conflict
	// where the lambda instance issuing the auth request can be
	// different from the lambda instance receiving the callback
	verifierCookie, err := r.Cookie(helpers.PKCE_VERIFIER_COOKIE_NAME)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Invalid or expired authorization session"), http.StatusBadRequest, false)(w, r)
		}
	}

	codeVerifier := verifierCookie.Value
	if codeVerifier == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Invalid or expired authorization session"), http.StatusBadRequest, false)(w, r)
		}
	}

	// PKCE verifiier token is now consumed, clear it
	services.ClearSubdomainCookie(w, helpers.PKCE_VERIFIER_COOKIE_NAME)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code is missing", http.StatusBadRequest)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Authorization code is missing"), http.StatusBadRequest, false)(w, r)
		}
	}

	zitadelRes, err := services.GetAuthToken(code, codeVerifier)
	if err != nil {
		log.Printf("Authentication Failed: %v", err)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Authentication failed"), http.StatusUnauthorized, false)(w, r)
		}
	}

	// Store the access token and refresh token securely
	accessToken, ok := zitadelRes["access_token"].(string)
	if !ok {
		if zitadelRes["error"] != "" {
			msg := fmt.Sprintf("Failed to get access tokens, error from zitadel: %+v", zitadelRes["error"])
			if zitadelRes["error_description"] != "" {
				msg += fmt.Sprintf(", error_description: %+v", zitadelRes["error_description"])
			}
			log.Printf(msg)
			return func(w http.ResponseWriter, r *http.Request) {
				transport.SendHtmlErrorPage([]byte(msg), http.StatusUnauthorized, false)(w, r)
			}
		}
		log.Printf("Failed to get access tokens, error from zitadel: %+v", zitadelRes)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Failed to get access token"), http.StatusInternalServerError, false)(w, r)
		}
	}

	refreshToken, ok := zitadelRes["refresh_token"].(string)
	if !ok {
		log.Printf("Refresh token error: %v", ok)
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SendHtmlErrorPage([]byte("Failed to get refresh token"), http.StatusInternalServerError, false)(w, r)
		}
	}
	idTokenHint, ok := zitadelRes["id_token"].(string)
	if !ok {
		log.Printf("id_token_hint not found in query params")
	}

	var userRedirectURL string = "/"
	var cookieDomain string = ""
	log.Printf("432: cookieDomain: %v", cookieDomain)
	if appState != "" {
		userRedirectURL = appState
		// Parse the redirect URL to get the host
		if parsedURL, err := url.Parse(appState); err == nil && parsedURL.Host != "" {
			cookieDomain = parsedURL.Host
		}
	}

	// Store tokens in cookies
	services.SetSubdomainCookie(w, helpers.MNM_ACCESS_TOKEN_COOKIE_NAME, accessToken, false, 0)
	services.SetSubdomainCookie(w, helpers.MNM_REFRESH_TOKEN_COOKIE_NAME, refreshToken, false, 0)
	services.SetSubdomainCookie(w, helpers.MNM_ID_TOKEN_COOKIE_NAME, idTokenHint, false, 0)

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, userRedirectURL, http.StatusFound)
	}
}

func HandleLogout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services.HandleLogout(w, r)
	}
}
