package main

// TODO: test "endTime" and add to UI

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"

	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/handlers/dynamodb_handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

type AuthType string

const (
	None               AuthType = "none"
	Check              AuthType = "check"
	Require            AuthType = "require"
	RequireServiceUser AuthType = "require_service_user"
)

type Route struct {
	Path    string
	Method  string
	Handler func(http.ResponseWriter, *http.Request) http.HandlerFunc
	Auth    AuthType
}

var Routes []Route

func init() {
	Routes = []Route{
		{"/auth/login", "GET", handlers.HandleLogin, None},
		{"/auth/callback", "GET", handlers.HandleCallback, None},
		{"/auth/logout", "GET", handlers.HandleLogout, None},
		{helpers.SitePages["home"].Slug, "GET", handlers.GetHomeOrUserPage, Check},
		{helpers.SitePages["about"].Slug, "GET", handlers.GetAboutPage, Check},
		{helpers.SitePages["user"].Slug, "GET", handlers.GetHomeOrUserPage, Check},
		{helpers.SitePages["add-event-source"].Slug, "GET", handlers.GetAddEventSourcePage, Require},
		{helpers.SitePages["admin"].Slug, "GET", handlers.GetAdminPage, Require},
		{helpers.SitePages["settings"].Slug, "GET", handlers.GetProfileSettingsPage, Require},
		{helpers.SitePages["add-event"].Slug, "GET", handlers.GetAddOrEditEventPage, Require},
		{helpers.SitePages["edit-event"].Slug, "GET", handlers.GetAddOrEditEventPage, Require},
		{helpers.SitePages["attendees-event"].Slug, "GET", handlers.GetEventAttendeesPage, Require},
		{helpers.SitePages["map-embed"].Slug, "GET", handlers.GetMapEmbedPage, Check},
		{helpers.SitePages["privacy-policy"].Slug, "GET", handlers.GetPrivacyPolicyPage, Check},
		{helpers.SitePages["data-request"].Slug, "GET", handlers.GetDataRequestPage, Check},
		{helpers.SitePages["terms-of-service"].Slug, "GET", handlers.GetTermsOfServicePage, Check},
		// TODO: sometimes `Check` will fail to retrieve the user info, this is different
		// from `Require` which always creates a new session if the user isn't logged in...
		// the complexity is we might want "in the middle", which would be "auto-refresh
		// the session, but DO NOT redirect to /login if the user's session is expired'"
		// session duration might be a Zitadel configuration issue
		{helpers.SitePages["event-detail"].Slug, "GET", handlers.GetEventDetailsPage, Check},
		// Below for competition engagement modules
		// {helpers.SitePages["competitions"].Slug, "GET", handlers.GetCompetitionsPage, Check},
		{helpers.SitePages["competition-edit"].Slug, "GET", handlers.GetAddOrEditCompetitionPage, Require},
		{helpers.SitePages["competition-new"].Slug, "GET", handlers.GetAddOrEditCompetitionPage, Require},

		// API routes

		// == START == need to expose these via permanent key for headless clients
		{"/api/event{trailingslash:\\/?}", "POST", handlers.PostEventHandler, Require},
		{"/api/events{trailingslash:\\/?}", "POST", handlers.PostBatchEventsHandler, Require},
		{"/api/events{trailingslash:\\/?}", "GET", handlers.SearchEventsHandler, None},
		{"/api/events{trailingslash:\\/?}", "PUT", handlers.BulkUpdateEventsHandler, Require},
		{"/api/events/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetOneEventHandler, None},
		{"/api/events/{" + helpers.EVENT_ID_KEY + "}", "PUT", handlers.UpdateOneEventHandler, Require},
		{"/api/events", "DELETE", handlers.BulkDeleteEventsHandler, Require},
		{"/api/event-reg-purch{trailingslash:\\/?}", "PUT", handlers.UpdateEventRegPurchHandler, Require},
		{"/api/event-reg-purch/{" + helpers.EVENT_ID_KEY + "}", "PUT", handlers.UpdateEventRegPurchHandler, Require},
		{"/api/locations{trailingslash:\\/?}", "GET", handlers.SearchLocationsHandler, None},
		//  == END == need to expose these via permanent key for headless clients
		{"/api/auth/users/update-mnm-options{trailingslash:\\/?}", "POST", handlers.SetMnmOptions, Require},
		{"/api/auth/users/update-interests{trailingslash:\\/?}", "POST", handlers.UpdateUserInterests, Require},
		{"/api/auth/users/update-about{trailingslash:\\/?}", "POST", handlers.UpdateUserAbout, Require},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo{trailingslash:\\/?}", "POST", handlers.GeoLookup, None},
		{"/api/user-search{trailingslash:\\/?}", "GET", handlers.SearchUsersHandler, Require},
		{"/api/users{trailingslash:\\/?}", "GET", handlers.GetUsersHandler, None},
		{"/api/html/events{trailingslash:\\/?}", "GET", handlers.GetEventsPartial, None},
		{"/api/html/event-series-form/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetEventAdminChildrenPartial, None},
		{"/api/html/seshu/session/submit{trailingslash:\\/?}", "POST", handlers.SubmitSeshuSession, Require},
		{"/api/html/seshu/session/location{trailingslash:\\/?}", "PUT", handlers.GeoThenPatchSeshuSession, Require},
		{"/api/html/seshu/session/events{trailingslash:\\/?}", "PUT", handlers.SubmitSeshuEvents, Require},
		{"/api/html/competition-config/owner/{" + helpers.USER_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionConfigsHtmlByPrimaryOwnerHandler, None},

		// // Purchasables routes
		{"/api/purchasables/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreatePurchasableHandler, Require},   // Create a new purchasable
		{"/api/purchasables/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasableHandler, None},          // Get all purchasables
		{"/api/purchasables/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "PUT", dynamodb_handlers.UpdatePurchasableHandler, Require},    // Update an existing purchasable
		{"/api/purchasables/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeletePurchasableHandler, Require}, // Delete a purchasable

		// RegistrationFields
		{"/api/registration-fields/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreateRegistrationFieldsHandler, Require},
		{"/api/registration-fields/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetRegistrationFieldsByEventIDHandler, None},
		{"/api/registration-fields/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "PUT", dynamodb_handlers.UpdateRegistrationFieldsHandler, Require},
		{"/api/registration-fields/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeleteRegistrationFieldsHandler, Require},

		// Purchases
		{"/api/purchases/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreatePurchaseHandler, Require},                     // Create a new event Purchase
		{"/api/purchases/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}/{created_at:[0-9]+}", "GET", dynamodb_handlers.GetPurchaseByPkHandler, Require}, // Get a specific event Purchase
		{"/api/purchases/event/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasesByEventIDHandler, Require},                                 // Get all event Purchases
		{"/api/purchases/user/{user_id:[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasesByUserIDHandler, Require},
		{"/api/purchases/has-for-event", "POST", dynamodb_handlers.HasPurchaseForEventHandler, Require},                                                                   // User has a purchase for one of two events
		{"/api/purchases/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}/{created_at:[0-9]+}", "PUT", dynamodb_handlers.UpdatePurchaseHandler, None}, // Update an existing event Purchase
		{"/api/purchases/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeletePurchaseHandler, None},

		// Competition Config
		{"/api/competition-config", "PUT", dynamodb_handlers.UpdateCompetitionConfigHandler, Require},
		{"/api/competition-config/owner", "GET", dynamodb_handlers.GetCompetitionConfigsByPrimaryOwnerHandler, Require},
		{"/api/competition-config/owner/{" + helpers.USER_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionConfigsByPrimaryOwnerHandler, None},
		{"/api/competition-config/{" + helpers.COMPETITIONS_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionConfigByIdHandler, Require},
		// verify below is correct with brian, was not accessing userId from context
		{"/api/competition-config/{" + helpers.COMPETITIONS_ID_KEY + "}", "PUT", dynamodb_handlers.UpdateCompetitionConfigHandler, Require},
		{"/api/competition-config/{" + helpers.COMPETITIONS_ID_KEY + "}", "DELETE", dynamodb_handlers.DeleteCompetitionConfigHandler, Require},

		// Competition Round
		{"/api/competition-round/{" + helpers.COMPETITIONS_ID_KEY + "}", "PUT", dynamodb_handlers.PutCompetitionRoundsHandler, Require}, // creation or update
		{"/api/competition-round/competition-sum/{" + helpers.COMPETITIONS_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionRoundsScoreSums, None},
		{"/api/competition-round/competition/{" + helpers.COMPETITIONS_ID_KEY + "}", "GET", dynamodb_handlers.GetAllCompetitionRoundsHandler, None},                                   // Gets all rounds for a competition using begins_with
		{"/api/competition-round/event/{" + helpers.EVENT_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionRoundsByEventIdHandler, Require},                                       // This gets a single round item by the event id it is associated with
		{"/api/competition-round/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.ROUND_NUMBER_KEY + "}", "GET", dynamodb_handlers.GetCompetitionRoundByPrimaryKeyHandler, Require}, // This gets a single round item by its own id
		{"/api/competition-round/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.ROUND_NUMBER_KEY + "}", "DELETE", dynamodb_handlers.DeleteCompetitionRoundHandler, Require},
		// summing, ending point for leader board needed here

		// Competition Waiting Room
		{"/api/waiting-room/{" + helpers.COMPETITIONS_ID_KEY + "}", "PUT", dynamodb_handlers.PutCompetitionWaitingRoomParticipantHandler, Require},
		{"/api/waiting-room/{" + helpers.COMPETITIONS_ID_KEY + "}", "GET", dynamodb_handlers.GetCompetitionWaitingRoomParticipantsHandler, Require},
		{"/api/waiting-room/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.USER_ID_KEY + "}", "DELETE", dynamodb_handlers.DeleteCompetitionWaitingRoomParticipantHandler, Require},

		// // Competition Vote
		{"/api/votes/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.ROUND_NUMBER_KEY + "}", "PUT", dynamodb_handlers.PutCompetitionVoteHandler, Require},
		{"/api/votes/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.ROUND_NUMBER_KEY + "}", "GET", dynamodb_handlers.GetCompetitionVotesByRoundHandler, Require},
		{"/api/votes/tally-votes/{" + helpers.COMPETITIONS_ID_KEY + "}/{" + helpers.ROUND_NUMBER_KEY + "}", "GET", dynamodb_handlers.GetCompetitionVotesTallyForRoundHandler, Require},
		{"/api/votes", "DELETE", dynamodb_handlers.DeleteCompetitionVoteHandler, Require},

		// Checkout Session
		{"/api/checkout/{" + helpers.EVENT_ID_KEY + ":[0-9a-fA-F-]+}", "POST", handlers.CreateCheckoutSessionHandler, Check},
		{"/api/webhook/checkout", "POST", handlers.HandleCheckoutWebhookHandler, None},
	}
}

type AuthConfig struct {
	AuthDomain     string
	AllowedDomains []string
	CookieDomain   string
}

type App struct {
	Router     *mux.Router
	AuthZ      *authorization.Authorizer[*oauth.IntrospectionContext]
	AuthConfig *AuthConfig
}

func NewApp() *App {
	app := &App{
		Router: mux.NewRouter(),
	}
	app.Router.Use(stateRedirectMiddleware)
	app.Router.Use(withContext)
	app.Router.Use(WithDerivedOptionsFromReq)
	app.InitializeAuth()
	log.Printf("App created: %+v", app)

	defer func() {
		app.InitStripe()
	}()
	return app
}

func (app *App) InitializeAuth() {
	services.InitAuth()
	app.AuthZ = services.GetAuthMw()
	app.AuthConfig = &AuthConfig{
		AuthDomain:     os.Getenv("ZITADEL_INSTANCE_HOST"),
		AllowedDomains: []string{strings.Replace(os.Getenv("APEX_URL"), "https://", "", 1), strings.Replace(os.Getenv("APEX_URL"), "https://", "*.", 1)},
		CookieDomain:   strings.Replace(os.Getenv("APEX_URL"), "https://", "", 1),
	}
}

func (app *App) SetupRoutes(routes []Route) {
	for _, route := range routes {
		app.addRoute(route)
	}
}

func (app *App) InitStripe() {
	services.InitStripe()
}

func (app *App) addRoute(route Route) {
	var handler http.HandlerFunc
	var accessTokenCookie *http.Cookie
	var refreshTokenCookie *http.Cookie
	var err error
	var refreshTokenCookieErr error

	switch route.Auth {
	case Require:
		handler = func(w http.ResponseWriter, r *http.Request) {
			var accessToken string

			// First check Authorization header
			authHeader := r.Header.Get("Authorization")
			redirectUrl := r.URL.String()

			if strings.HasPrefix(authHeader, "Bearer ") {
				accessToken = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				// Fall back to cookie-based auth
				accessTokenCookie, err = r.Cookie(helpers.MNM_ACCESS_TOKEN_COOKIE_NAME)
				if err != nil {
					refreshTokenCookie, refreshTokenCookieErr = r.Cookie(helpers.MNM_REFRESH_TOKEN_COOKIE_NAME)
					if refreshTokenCookieErr != nil {
						state := base64.URLEncoding.EncodeToString([]byte(redirectUrl))
						loginURL := fmt.Sprintf("/auth/login?state=%s&redirect=%s", state, url.QueryEscape(redirectUrl))
						http.Redirect(w, r, loginURL, http.StatusFound)
						return
					}

					tokens, refreshAccessTokenErr := services.RefreshAccessToken(refreshTokenCookie.Value)
					if refreshAccessTokenErr != nil {
						log.Printf("Require middleware refresh / access token error: %+v", refreshAccessTokenErr)
						loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(redirectUrl))
						http.Redirect(w, r, loginURL, http.StatusFound)
						return
					}

					// Store the access token and refresh token securely
					newAccessToken, ok := tokens[helpers.MNM_ACCESS_TOKEN_COOKIE_NAME].(string)
					if !ok {
						log.Printf("Failed to get access token in require middleware: %+v", refreshAccessTokenErr)
						loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(redirectUrl))
						http.Redirect(w, r, loginURL, http.StatusFound)
						return
					}

					refreshToken, ok := tokens[helpers.MNM_REFRESH_TOKEN_COOKIE_NAME].(string)
					if !ok {
						log.Printf("Failed to get refresh token in require middleware: %+v", refreshAccessTokenErr)
						loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(redirectUrl))
						http.Redirect(w, r, loginURL, http.StatusFound)
						return
					}

					idTokenHint, ok := tokens[helpers.MNM_ID_TOKEN_COOKIE_NAME].(string)
					if !ok {
						log.Printf("Failed to get id_token in require middleware: %+v", refreshAccessTokenErr)
						loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(redirectUrl))
						http.Redirect(w, r, loginURL, http.StatusFound)
						return
					}

					// Store tokens in cookies
					services.SetSubdomainCookie(w, helpers.MNM_ACCESS_TOKEN_COOKIE_NAME, newAccessToken, false, 0)
					services.SetSubdomainCookie(w, helpers.MNM_REFRESH_TOKEN_COOKIE_NAME, refreshToken, false, 0)
					services.SetSubdomainCookie(w, helpers.MNM_ID_TOKEN_COOKIE_NAME, idTokenHint, false, 0)
					accessToken = newAccessToken
					http.Redirect(w, r, redirectUrl, http.StatusFound)
					return
				}
				accessToken = accessTokenCookie.Value
			}

			// Use the Authorizer to introspect the access token
			authCtx, err := app.AuthZ.CheckAuthorization(r.Context(), "Bearer "+accessToken)
			if err != nil {
				log.Printf("Authorization Failed: %v", err)
				if strings.HasPrefix(authHeader, "Bearer ") {
					loginURL := fmt.Sprintf("/auth/login?redirect=%s", url.QueryEscape(redirectUrl))
					http.Redirect(w, r, loginURL, http.StatusFound)
				} else {
					// Store the original host and URL in the state parameter
					state := base64.URLEncoding.EncodeToString([]byte(redirectUrl))
					loginURL := fmt.Sprintf("/auth/login?state=%s&redirect=%s", state, url.QueryEscape(redirectUrl))
					http.Redirect(w, r, loginURL, http.StatusFound)
				}
				return
			}

			claims := authCtx.Claims
			roleClaims, userMetaClaims := services.ExtractClaimsMeta(claims)

			userInfo := helpers.UserInfo{}
			data, err := json.MarshalIndent(authCtx, "", "	")
			if err != nil {
				http.Redirect(w, r, "/auth/login"+"?redirect="+redirectUrl, http.StatusFound)
				return
			}
			err = json.Unmarshal(data, &userInfo)
			if err != nil {
				http.Redirect(w, r, "/auth/login"+"?redirect="+redirectUrl, http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), "userInfo", userInfo)

			if roleClaims != nil {
				ctx = context.WithValue(ctx, "roleClaims", roleClaims)
			}
			if userMetaClaims != nil {
				ctx = context.WithValue(ctx, "userMetaClaims", userMetaClaims)
			}
			r = r.WithContext(ctx)

			// After successful auth, check for state parameter to handle subdomain redirect
			state := r.URL.Query().Get("state")
			if state != "" {
				decodedState, err := base64.URLEncoding.DecodeString(state)
				if err == nil {
					parts := strings.Split(string(decodedState), "|")
					if len(parts) == 2 {
						originalHost := parts[0]
						originalURL := parts[1]

						// Extract subdomain from original host
						hostParts := strings.Split(originalHost, ".")
						apexDomain := strings.Join(hostParts[len(hostParts)-2:], ".")
						subdomain := ""
						if len(hostParts) > 2 {
							subdomain = strings.Join(hostParts[:len(hostParts)-2], ".")
						}

						// Build the final URL with subdomain
						var finalURL string
						if subdomain != "" {
							finalURL = fmt.Sprintf("https://%s.%s%s", subdomain, apexDomain, originalURL)
						} else {
							finalURL = fmt.Sprintf("https://%s%s", apexDomain, originalURL)
						}

						http.Redirect(w, r, finalURL, http.StatusFound)
						return
					}
				}
			}

			route.Handler(w, r).ServeHTTP(w, r)
		}
	case Check:
		handler = func(w http.ResponseWriter, r *http.Request) {
			// Get the access token from cookies
			accessTokenCookie, err = r.Cookie(helpers.MNM_ACCESS_TOKEN_COOKIE_NAME)
			if err != nil {
				route.Handler(w, r).ServeHTTP(w, r)
				return
			}

			accessToken := "Bearer " + accessTokenCookie.Value

			// Use the Authorizer to introspect the access token
			authCtx, err := app.AuthZ.CheckAuthorization(r.Context(), accessToken)
			if err != nil {
				route.Handler(w, r).ServeHTTP(w, r)
				return
			}

			claims := authCtx.Claims
			roleClaims, userMetaClaims := services.ExtractClaimsMeta(claims)

			userInfo := helpers.UserInfo{}
			data, err := json.MarshalIndent(authCtx, "", "	")
			if err != nil {
				route.Handler(w, r).ServeHTTP(w, r)
				return
			}

			err = json.Unmarshal(data, &userInfo)
			if err != nil {
				route.Handler(w, r).ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), "userInfo", userInfo)
			if roleClaims != nil {
				ctx = context.WithValue(ctx, "roleClaims", roleClaims)
			}
			if userMetaClaims != nil {
				ctx = context.WithValue(ctx, "userMetaClaims", userMetaClaims)
			}
			r = r.WithContext(ctx)
			route.Handler(w, r).ServeHTTP(w, r)
		}
	case RequireServiceUser:
		handler = func(w http.ResponseWriter, r *http.Request) {
			accessTokenCookie, err = r.Cookie(helpers.MNM_ACCESS_TOKEN_COOKIE_NAME)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			tokenString := accessTokenCookie.Value

			jwks, err := services.FetchJWKS()
			if err != nil {
				http.Error(w, "Error fetching JWKS", http.StatusInternalServerError)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}

				kid, ok := token.Header["kid"].(string)
				if !ok {
					return nil, fmt.Errorf("kid not found in token header")
				}

				return services.GetPublicKey(jwks, kid)
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Extract claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// Extract roles, metadata, and user information
				userID := claims["sub"]

				log.Printf("Claims: %v", claims)
				log.Printf("User ID: %v", userID)

				// Add extracted information to the request context
				ctx := r.Context()
				ctx = context.WithValue(ctx, "userID", userID)
				r = r.WithContext(ctx)
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			route.Handler(w, r).ServeHTTP(w, r)
		}
	default:
		handler = func(w http.ResponseWriter, r *http.Request) {
			route.Handler(w, r).ServeHTTP(w, r)
		}
	}

	app.Router.HandleFunc(route.Path, handler).Methods(route.Method).Name(route.Path)
}

func (app *App) SetupNotFoundHandler() {
	app.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Not found", r.RequestURI)
		http.Error(w, fmt.Sprintf("Not found: %s", r.RequestURI), http.StatusNotFound)
	})
}

// Middleware to inject context into the request
func withContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Add a dummy APIGatewayV2HTTPRequest for testing
		if _, ok := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest); !ok {
			ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: r.Method,
						Path:   r.URL.Path,
					},
				},
			})
		}
		// Add context to request
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// Global middleware function for final_redirect_uri state parameter handling
func stateRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		stateParam := r.URL.Query().Get("state")
		if stateParam != "" {
			// Try to decode the state parameter
			decodedBytes, err := base64.URLEncoding.DecodeString(stateParam)
			if err == nil {
				decodedState := string(decodedBytes)
				if strings.Contains(decodedState, helpers.FINAL_REDIRECT_URI_KEY+"=") {
					parts := strings.Split(decodedState, helpers.FINAL_REDIRECT_URI_KEY+"=")
					if len(parts) > 1 {
						redirectURI := parts[1]
						// Remove any additional parameters if present
						if idx := strings.Index(redirectURI, "&"); idx != -1 {
							redirectURI = redirectURI[:idx]
						}
						redirectURI, err = url.QueryUnescape(redirectURI)
						if err != nil {
							log.Printf("Failed to unescape redirectURI: %v", err)
							http.Redirect(w, r, os.Getenv("APEX_URL"), http.StatusFound)
							return
						}
						http.Redirect(w, r, redirectURI, http.StatusFound)
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func WithDerivedOptionsFromReq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mnmOptions := map[string]string{}
		mnmOptionsHeaderVal := strings.Trim(r.Header.Get("X-Mnm-Options"), "\"")
		if strings.Contains(mnmOptionsHeaderVal, "=") {
			parts := strings.Split(mnmOptionsHeaderVal, ";")
			for _, part := range parts {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					key := strings.Trim(kv[0], " \"") // trim spaces and quotes
					value := strings.Trim(kv[1], " \"")
					if slices.Contains(helpers.AllowedMnmOptionsKeys, key) {
						mnmOptions[key] = value
					}
				} else {
					log.Printf("kv length != 2: '%d'", len(kv))
				}
			}
		} else {
			mnmOptions["userId"] = strings.Trim(mnmOptionsHeaderVal, " \"")
		}
		ctx := context.WithValue(r.Context(), helpers.MNM_OPTIONS_CTX_KEY, mnmOptions)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	flag.Parse()
	app := NewApp()
	app.InitializeAuth()
	app.SetupNotFoundHandler()

	// This is the package level instance of Db in handlers
	_ = transport.GetDB()

	app.SetupRoutes(Routes)

	adapter := gorillamux.NewV2(app.Router)

	lambda.Start(func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, request)
		return adapter.ProxyWithContext(ctx, request)
	})
}
