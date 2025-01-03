package main

// TODO: test "endTime" and add to UI

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
		{helpers.SitePages["home"].Slug, "GET", handlers.GetHomePage, Check},
		{helpers.SitePages["about"].Slug, "GET", handlers.GetAboutPage, Check},
		{helpers.SitePages["add-event-source"].Slug, "GET", handlers.GetAddEventSourcePage, Require},
		{helpers.SitePages["profile"].Slug, "GET", handlers.GetProfilePage, Require},
		{helpers.SitePages["settings"].Slug, "GET", handlers.GetProfileSettingsPage, Require},
		{helpers.SitePages["add-event"].Slug, "GET", handlers.GetAddOrEditEventPage, Require},
		{helpers.SitePages["edit-event"].Slug, "GET", handlers.GetAddOrEditEventPage, Require},
		{helpers.SitePages["attendees-event"].Slug, "GET", handlers.GetEventAttendeesPage, Require},
		{helpers.SitePages["map-embed"].Slug, "GET", handlers.GetMapEmbedPage, None},
		// TODO: sometimes `Check` will fail to retrieve the user info, this is different
		// from `Require` which always creates a new session if the user isn't logged in...
		// the complexity is we might want "in the middle", which would be "auto-refresh
		// the session, but DO NOT redirect to /login if the user's session is expired'"
		// session duration might be a ZFvitadel configuration issue
		{helpers.SitePages["event-detail"].Slug, "GET", handlers.GetEventDetailsPage, Check},

		// API routes

		// == START == need to expose these via permanent key for headless clients
		{"/api/event", "POST", handlers.PostEventHandler, Require},
		{"/api/events", "POST", handlers.PostBatchEventsHandler, Require},
		{"/api/events", "GET", handlers.SearchEventsHandler, None},
		{"/api/events", "PUT", handlers.BulkUpdateEventsHandler, Require},
		{"/api/events/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetOneEventHandler, None},
		{"/api/events/{" + helpers.EVENT_ID_KEY + "}", "PUT", handlers.UpdateOneEventHandler, Require},
		{"/api/locations", "GET", handlers.SearchLocationsHandler, None},
		//  == END == need to expose these via permanent key for headless clients
		{"/api/auth/users/set-subdomain", "POST", handlers.SetUserSubdomain, Require},
		{"/api/auth/users/update-interests", "POST", handlers.UpdateUserInterests, Require},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo", "POST", handlers.GeoLookup, None},
		{"/api/user-search", "GET", handlers.SearchUsersHandler, Require},
		{"/api/users", "GET", handlers.GetUsersHandler, Require},
		{"/api/html/events", "GET", handlers.GetEventsPartial, None},
		{"/api/html/seshu/session/submit", "POST", handlers.SubmitSeshuSession, Require},
		{"/api/html/seshu/session/location", "PUT", handlers.GeoThenPatchSeshuSession, Require},
		{"/api/html/seshu/session/events", "PUT", handlers.SubmitSeshuEvents, Require},

		// // Purchasables routes
		{"/api/purchasables/{event_id:[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreatePurchasableHandler, Require},   // Create a new purchasable
		{"/api/purchasables/{event_id:[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasableHandler, None},          // Get all purchasables
		{"/api/purchasables/{event_id:[0-9a-fA-F-]+}", "PUT", dynamodb_handlers.UpdatePurchasableHandler, Require},    // Update an existing purchasable
		{"/api/purchasables/{event_id:[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeletePurchasableHandler, Require}, // Delete a purchasable

		// RegistrationFields
		{"/api/registration-fields/{event_id:[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreateRegistrationFieldsHandler, Require},
		{"/api/registration-fields/{event_id:[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetRegistrationFieldsByEventIDHandler, None},
		{"/api/registration-fields/{event_id:[0-9a-fA-F-]+}", "PUT", dynamodb_handlers.UpdateRegistrationFieldsHandler, Require},
		{"/api/registration-fields/{event_id:[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeleteRegistrationFieldsHandler, Require},

		// Purchases
		{"/api/purchases/{event_id:[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}", "POST", dynamodb_handlers.CreatePurchaseHandler, Require},                     // Create a new event Purchase
		{"/api/purchases/{event_id:[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}/{created_at:[0-9]+}", "GET", dynamodb_handlers.GetPurchaseByPkHandler, Require}, // Get a specific event Purchase
		{"/api/purchases/event/{event_id:[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasesByEventIDHandler, Require},                                 // Get all event Purchases
		{"/api/purchases/user/{user_id:[0-9a-fA-F-]+}", "GET", dynamodb_handlers.GetPurchasesByUserIDHandler, Require},                                    // Get a specific event Purchase
		{"/api/purchases/{event_id:[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}/{created_at:[0-9]+}", "PUT", dynamodb_handlers.UpdatePurchaseHandler, None},     // Update an existing event Purchase
		{"/api/purchases/{event_id:[0-9a-fA-F-]+}/{user_id:[0-9a-fA-F-]+}", "DELETE", dynamodb_handlers.DeletePurchaseHandler, None},                      // Delete an event Purchase

		// Checkout Session
		{"/api/checkout/{event_id:[0-9a-fA-F-]+}", "POST", handlers.CreateCheckoutSessionHandler, Check},
		{"/api/webhook/checkout", "POST", handlers.HandleCheckoutWebhookHandler, None},
	}
}

type App struct {
	Router *mux.Router
	AuthZ  *authorization.Authorizer[*oauth.IntrospectionContext]
}

func NewApp() *App {
	app := &App{
		Router: mux.NewRouter(),
	}
	app.Router.Use(withContext)
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
				accessTokenCookie, err = r.Cookie("access_token")
				if err != nil {
					refreshTokenCookie, refreshTokenCookieErr = r.Cookie("refresh_token")
					if refreshTokenCookieErr != nil {
						http.Redirect(w, r, "/auth/login"+"?redirect="+redirectUrl, http.StatusFound)
						return
					}

					tokens, refreshAccessTokenErr := services.RefreshAccessToken(refreshTokenCookie.Value)
					if refreshAccessTokenErr != nil {
						log.Printf("Authentication Failed: %v", refreshAccessTokenErr)
						http.Error(w, "Authentication failed", http.StatusUnauthorized)
						return
					}

					// Store the access token and refresh token securely
					newAccessToken, ok := tokens["access_token"].(string)
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

					// Store tokens in cookies
					http.SetCookie(w, &http.Cookie{
						Name:     "access_token",
						Value:    newAccessToken,
						Path:     "/",
						HttpOnly: true,
					})

					http.SetCookie(w, &http.Cookie{
						Name:     "refresh_token",
						Value:    refreshToken,
						Path:     "/",
						HttpOnly: true,
					})

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
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
				} else {
					log.Printf("Redirecting to login, redirect is: %v", redirectUrl)
					http.Redirect(w, r, "/auth/login"+"?redirect="+redirectUrl, http.StatusFound)
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
			route.Handler(w, r).ServeHTTP(w, r)
		}
	case Check:
		handler = func(w http.ResponseWriter, r *http.Request) {
			// Get the access token from cookies
			accessTokenCookie, err = r.Cookie("access_token")
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
			accessTokenCookie, err = r.Cookie("access_token")
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
