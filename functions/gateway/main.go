package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization"
	"github.com/zitadel/zitadel-go/v3/pkg/authorization/oauth"

	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

type AuthType string

const (
	None    AuthType = "none"
	Check   AuthType = "check"
	Require AuthType = "require"
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
		{"/", "GET", handlers.GetHomePage, Check},
		{"/auth/login", "GET", handlers.HandleLogin, None},
		{"/auth/callback", "GET", handlers.HandleCallback, None},
		{"/auth/logout", "GET", handlers.HandleLogout, None},
		{"/admin/add-event-source", "GET", handlers.GetAddEventSourcePage, Require},
		{"/admin/profile", "GET", handlers.GetProfilePage, Require},
		{"/admin/profile/settings", "GET", handlers.GetProfileSettingsPage, Require},
		{"/map-embed", "GET", handlers.GetMapEmbedPage, None},
		// TODO: sometimes `Check` will fail to retrieve the user info, this is different
		// from `Require` which always creates a new session if the user isn't logged in...
		// the complexity is we might want "in the middle", which would be "auto-refresh
		// the session, but DO NOT redirect to /login if the user's session is expired'"
		// session duration might be a Zitadel configuration issue
		{"/events/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetEventDetailsPage, Check},

		// API routes

		// == START == need to expose these via permanent key for headless clients
		{"/api/event", "POST", handlers.PostEventHandler, None},
		{"/api/events", "POST", handlers.PostBatchEventsHandler, None},
		{"/api/events", "GET", handlers.SearchEventsHandler, None},
		{"/api/events/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetOneEventHandler, None},
		//  == END == need to expose these via permanent key for headless clients

		// {"/api/event", "POST", handlers.CreateEventHandler, None},
		{"/api/user/set-subdomain", "POST", handlers.SetUserSubdomain, Check},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo", "POST", handlers.GeoLookup, None},
		{"/api/html/seshu/session/submit", "POST", handlers.SubmitSeshuSession, None},
		{"/api/html/seshu/session/location", "PATCH", handlers.GeoThenPatchSeshuSession, None},
		{"/api/html/seshu/session/events", "PATCH", handlers.SubmitSeshuEvents, None},
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

func (app *App) addRoute(route Route) {
	var handler http.HandlerFunc
	var accessTokenCookie *http.Cookie
	var refreshTokenCookie *http.Cookie
	var err error
	var refreshTokenCookieErr error
	switch route.Auth {
	case Require:
		handler = func(w http.ResponseWriter, r *http.Request) {
			accessTokenCookie, err = r.Cookie("access_token")
			if err != nil {
				refreshTokenCookie, refreshTokenCookieErr = r.Cookie("refresh_token")
				if refreshTokenCookieErr != nil {
					http.Redirect(w, r, "/auth/login"+"?redirect="+route.Path, http.StatusFound)
					return
				}

				tokens, refreshAccessTokenErr := services.RefreshAccessToken(refreshTokenCookie.Value)
				if refreshAccessTokenErr != nil {
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

				var userRedirectURL string = route.Path
				http.Redirect(w, r, userRedirectURL, http.StatusFound)
				return
			}

			accessToken := "Bearer " + accessTokenCookie.Value

			// Use the Authorizer to introspect the access token
			authCtx, err := app.AuthZ.CheckAuthorization(r.Context(), accessToken)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid access token", http.StatusUnauthorized)
				return
			}

			claims := authCtx.Claims
			roleClaims := services.ExtractRoleClaims(claims)

			userInfo := helpers.UserInfo{}
			data, err := json.MarshalIndent(authCtx, "", "	")
			if err != nil {
				http.Error(w, "Unauthorized: Unable to fetch user information", http.StatusUnauthorized)
				return
			}
			err = json.Unmarshal(data, &userInfo)
			if err != nil {
				http.Error(w, "Unauthorized: Unable to fetch user information", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), "userInfo", userInfo)

			if roleClaims != nil {
				ctx = context.WithValue(ctx, "roleClaims", roleClaims)
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
			roleClaims := services.ExtractRoleClaims(claims)

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
			r = r.WithContext(ctx)
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
