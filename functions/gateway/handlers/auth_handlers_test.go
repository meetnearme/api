package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/constants"
)

func TestHandleLogin(t *testing.T) {
	// Save original environment variables
	originalApexURL := os.Getenv("APEX_URL")
	originalZitadelHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalClientID := os.Getenv("ZITADEL_CLIENT_ID")
	defer func() {
		os.Setenv("APEX_URL", originalApexURL)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelHost)
		os.Setenv("ZITADEL_CLIENT_ID", originalClientID)
	}()

	tests := []struct {
		name           string
		apexURL        string
		zitadelHost    string
		clientID       string
		redirectParam  string
		expectedStatus int
		expectRedirect bool
	}{
		{
			name:           "Successful login with redirect",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			redirectParam:  "/dashboard",
			expectedStatus: http.StatusFound,
			expectRedirect: true,
		},
		{
			name:           "Successful login without redirect",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			redirectParam:  "",
			expectedStatus: http.StatusFound,
			expectRedirect: true,
		},
		{
			name:           "Missing APEX_URL",
			apexURL:        "",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			redirectParam:  "",
			expectedStatus: http.StatusOK, // SendHtmlErrorPage always returns 200
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("APEX_URL", tt.apexURL)
			os.Setenv("ZITADEL_INSTANCE_HOST", tt.zitadelHost)
			os.Setenv("ZITADEL_CLIENT_ID", tt.clientID)

			// Create request
			url := "/auth/login"
			if tt.redirectParam != "" {
				url += "?redirect=" + tt.redirectParam
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Add AWS Lambda context (required for transport layer)
			ctx := req.Context()
			ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Call the handler - match the actual usage pattern
			handler := HandleLogin(w, req)
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Result().StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Result().StatusCode)
			}

			// Check for redirect
			if tt.expectRedirect {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("Expected redirect location header, got none")
				}
				if !strings.Contains(location, "authorize") {
					t.Errorf("Expected redirect to contain 'authorize', got %s", location)
				}
			}

			// Check for PKCE verifier cookie
			cookies := w.Result().Cookies()
			foundVerifier := false
			for _, cookie := range cookies {
				if cookie.Name == constants.PKCE_VERIFIER_COOKIE_NAME {
					foundVerifier = true
					break
				}
			}
			if tt.expectRedirect && !foundVerifier {
				t.Error("Expected PKCE verifier cookie, got none")
			}
		})
	}
}

func TestHandleCallback(t *testing.T) {
	// Save original environment variables
	originalApexURL := os.Getenv("APEX_URL")
	originalZitadelHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalClientID := os.Getenv("ZITADEL_CLIENT_ID")
	originalClientSecret := os.Getenv("ZITADEL_CLIENT_SECRET")
	defer func() {
		os.Setenv("APEX_URL", originalApexURL)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelHost)
		os.Setenv("ZITADEL_CLIENT_ID", originalClientID)
		os.Setenv("ZITADEL_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name           string
		sessionID      string
		state          string
		code           string
		verifierCookie string
		apexURL        string
		zitadelHost    string
		clientID       string
		clientSecret   string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Session ID redirect",
			sessionID:      "test-session",
			state:          "test-state",
			code:           "",
			verifierCookie: "",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusFound,
			expectError:    false,
		},
		{
			name:           "Missing verifier cookie",
			sessionID:      "",
			state:          "",
			code:           "test-code",
			verifierCookie: "",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusOK, // SendHtmlErrorPage always returns 200
			expectError:    true,
		},
		{
			name:           "Missing authorization code",
			sessionID:      "",
			state:          "",
			code:           "",
			verifierCookie: "test-verifier",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusBadRequest, // http.Error is called directly
			expectError:    true,
		},
		{
			name:           "Missing APEX_URL",
			sessionID:      "",
			state:          "",
			code:           "test-code",
			verifierCookie: "test-verifier",
			apexURL:        "",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusOK,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("APEX_URL", tt.apexURL)
			os.Setenv("ZITADEL_INSTANCE_HOST", tt.zitadelHost)
			os.Setenv("ZITADEL_CLIENT_ID", tt.clientID)
			os.Setenv("ZITADEL_CLIENT_SECRET", tt.clientSecret)

			// Create request with query parameters
			url := "/auth/callback"
			params := []string{}
			if tt.sessionID != "" {
				params = append(params, "id="+tt.sessionID)
			}
			if tt.state != "" {
				params = append(params, "state="+tt.state)
			}
			if tt.code != "" {
				params = append(params, "code="+tt.code)
			}
			if len(params) > 0 {
				url += "?" + strings.Join(params, "&")
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Add verifier cookie if provided
			if tt.verifierCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  constants.PKCE_VERIFIER_COOKIE_NAME,
					Value: tt.verifierCookie,
				})
			}

			// Add AWS Lambda context
			ctx := req.Context()
			ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Call the handler - match the actual usage pattern
			handler := HandleCallback(w, req)
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Result().StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Result().StatusCode)
			}

			// For session ID redirect, check for Location header
			if tt.sessionID != "" {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("Expected Location header for session ID redirect, got none")
				}
			}

			// For successful auth, check for token cookies
			if !tt.expectError && tt.code != "" && tt.verifierCookie != "" {
				cookies := w.Result().Cookies()
				expectedCookies := []string{
					constants.MNM_ACCESS_TOKEN_COOKIE_NAME,
					constants.MNM_REFRESH_TOKEN_COOKIE_NAME,
					constants.MNM_ID_TOKEN_COOKIE_NAME,
				}
				for _, expectedCookie := range expectedCookies {
					found := false
					for _, cookie := range cookies {
						if cookie.Name == expectedCookie {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected cookie %s, got none", expectedCookie)
					}
				}
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	// Save original environment variables
	originalApexURL := os.Getenv("APEX_URL")
	originalZitadelHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalClientID := os.Getenv("ZITADEL_CLIENT_ID")
	originalClientSecret := os.Getenv("ZITADEL_CLIENT_SECRET")
	defer func() {
		os.Setenv("APEX_URL", originalApexURL)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelHost)
		os.Setenv("ZITADEL_CLIENT_ID", originalClientID)
		os.Setenv("ZITADEL_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name           string
		apexURL        string
		zitadelHost    string
		clientID       string
		clientSecret   string
		expectedStatus int
	}{
		{
			name:           "Successful logout",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusFound, // Logout redirects
		},
		{
			name:           "Logout with missing APEX_URL",
			apexURL:        "",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			expectedStatus: http.StatusFound, // Logout redirects even with missing APEX_URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("APEX_URL", tt.apexURL)
			os.Setenv("ZITADEL_INSTANCE_HOST", tt.zitadelHost)
			os.Setenv("ZITADEL_CLIENT_ID", tt.clientID)
			os.Setenv("ZITADEL_CLIENT_SECRET", tt.clientSecret)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)

			// Add AWS Lambda context
			ctx := req.Context()
			ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Call the handler - match the actual usage pattern
			handler := HandleLogout(w, req)
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Result().StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Result().StatusCode)
			}

			// Check for redirect location
			location := w.Header().Get("Location")
			if location == "" {
				t.Error("Expected Location header for logout redirect, got none")
			}

			// For successful logout with APEX_URL, check that auth cookies are cleared
			if tt.apexURL != "" {
				cookies := w.Result().Cookies()
				authCookies := []string{
					constants.MNM_ACCESS_TOKEN_COOKIE_NAME,
					constants.MNM_REFRESH_TOKEN_COOKIE_NAME,
					constants.MNM_ID_TOKEN_COOKIE_NAME,
					constants.PKCE_VERIFIER_COOKIE_NAME,
				}
				for _, cookieName := range authCookies {
					found := false
					for _, cookie := range cookies {
						if cookie.Name == cookieName {
							found = true
							// Check that the cookie is being cleared (MaxAge < 0 or Expires in the past)
							if cookie.MaxAge >= 0 && !cookie.Expires.IsZero() && cookie.Expires.After(time.Now()) {
								t.Errorf("Expected cookie %s to be cleared, but it's not", cookieName)
							}
							break
						}
					}
					if !found {
						t.Errorf("Expected cookie %s to be present in response, got none", cookieName)
					}
				}
			}
		})
	}
}

func TestHandleRefresh(t *testing.T) {
	// Save original environment variables
	originalApexURL := os.Getenv("APEX_URL")
	originalZitadelHost := os.Getenv("ZITADEL_INSTANCE_HOST")
	originalClientID := os.Getenv("ZITADEL_CLIENT_ID")
	originalClientSecret := os.Getenv("ZITADEL_CLIENT_SECRET")
	originalGoEnv := os.Getenv("GO_ENV")
	defer func() {
		os.Setenv("APEX_URL", originalApexURL)
		os.Setenv("ZITADEL_INSTANCE_HOST", originalZitadelHost)
		os.Setenv("ZITADEL_CLIENT_ID", originalClientID)
		os.Setenv("ZITADEL_CLIENT_SECRET", originalClientSecret)
		os.Setenv("GO_ENV", originalGoEnv)
	}()

	tests := []struct {
		name           string
		refreshToken   string
		apexURL        string
		zitadelHost    string
		clientID       string
		clientSecret   string
		goEnv          string
		expectedStatus int
		expectError    bool
		expectCookies  bool
	}{
		{
			name:           "Successful refresh",
			refreshToken:   "test-refresh-token",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			goEnv:          constants.GO_TEST_ENV,
			expectedStatus: http.StatusOK,
			expectError:    false,
			expectCookies:  true,
		},
		{
			name:           "Missing refresh token cookie",
			refreshToken:   "",
			apexURL:        "https://example.com",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			goEnv:          constants.GO_TEST_ENV,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
			expectCookies:  false,
		},
		{
			name:           "Missing APEX_URL",
			refreshToken:   "test-refresh-token",
			apexURL:        "",
			zitadelHost:    "example.zitadel.cloud",
			clientID:       "test-client-id",
			clientSecret:   "test-client-secret",
			goEnv:          constants.GO_TEST_ENV,
			expectedStatus: http.StatusOK, // Still works in test mode
			expectError:    false,
			expectCookies:  false, // Cookies won't be set when APEX_URL is missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("APEX_URL", tt.apexURL)
			os.Setenv("ZITADEL_INSTANCE_HOST", tt.zitadelHost)
			os.Setenv("ZITADEL_CLIENT_ID", tt.clientID)
			os.Setenv("ZITADEL_CLIENT_SECRET", tt.clientSecret)
			os.Setenv("GO_ENV", tt.goEnv)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)

			// Add refresh token cookie if provided
			if tt.refreshToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  constants.MNM_REFRESH_TOKEN_COOKIE_NAME,
					Value: tt.refreshToken,
				})
			}

			// Add AWS Lambda context
			ctx := req.Context()
			ctx = context.WithValue(ctx, constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			// Call the handler - match the actual usage pattern
			handler := HandleRefresh(w, req)
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Result().StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Result().StatusCode)
			}

			// Check response body for success case
			if !tt.expectError {
				body := w.Body.String()
				if !strings.Contains(body, `"success": true`) {
					t.Errorf("Expected success response body, got %s", body)
				}
			}

			// Check for token cookies in successful cases
			if tt.expectCookies {
				cookies := w.Result().Cookies()
				expectedCookies := []string{
					constants.MNM_ACCESS_TOKEN_COOKIE_NAME,
					constants.MNM_REFRESH_TOKEN_COOKIE_NAME,
					constants.MNM_ID_TOKEN_COOKIE_NAME,
				}
				for _, expectedCookie := range expectedCookies {
					found := false
					for _, cookie := range cookies {
						if cookie.Name == expectedCookie {
							found = true
							// Check that the cookie has a value (not empty)
							if cookie.Value == "" {
								t.Errorf("Expected cookie %s to have a value, got empty", expectedCookie)
							}
							break
						}
					}
					if !found {
						t.Errorf("Expected cookie %s, got none", expectedCookie)
					}
				}
			}
		})
	}
}
