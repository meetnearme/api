package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"

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
var codeVerifier string
var codeChallenge string

func init() {
	Routes = []Route{
		{"/", "GET", handlers.GetHomePage, Check},
		{"/admin/add-event-source", "GET", handlers.GetAddEventSourcePage, Require},
		{"/admin/profile", "GET", handlers.GetProfilePage, Require},
		{"/map-embed", "GET", handlers.GetMapEmbedPage, None},
		// TODO: sometimes `Check` will fail to retrieve the user info, this is different
		// from `Require` which always creates a new session if the user isn't logged in...
		// the complexity is we might want "in the middle", which would be "auto-refresh
		// the session, but DO NOT redirect to /login if the user's session is expired'"
		// session duration might be a Zitadel configuration issue
		{"/events/{" + helpers.EVENT_ID_KEY + "}", "GET", handlers.GetEventDetailsPage, Check},
        {"/api/user/set-subdomain", "POST", handlers.SetUserSubdomain, Check},
		{"/api/event", "POST", handlers.CreateEventHandler, None},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo", "POST", handlers.GeoLookup, None},
		{"/api/html/seshu/session/submit", "POST", handlers.SubmitSeshuSession, None},
		{"/api/html/seshu/session/location", "PATCH", handlers.GeoThenPatchSeshuSession, None},
		{"/api/html/seshu/session/events", "PATCH", handlers.SubmitSeshuEvents, None},
	}

	var err error
	codeChallenge, codeVerifier, err = services.GenerateCodeChallengeAndVerifier()
	if err != nil {
		fmt.Println("Error generating code verifier and challenge:", err)
		return
	}
}

type App struct {
	Router *mux.Router
	Mw     *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
	AuthN  *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
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
	app.Mw, app.AuthN = services.GetAuthMw()
}

func (app *App) SetupRoutes(routes []Route) {
	for _, route := range routes {
		app.addRoute(route)
	}
}

func (app *App) addRoute(route Route) {
	var handler http.HandlerFunc
	switch route.Auth {
	case Require:
		handler = func(w http.ResponseWriter, r *http.Request) {
			if app.Mw == nil {
				log.Println("Warning: app.Mw is nil, skipping authentication")
				route.Handler(w, r).ServeHTTP(w, r)
				return
			}

			app.Mw.RequireAuthentication()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authState := app.Mw.Context(r.Context())
				if authState == nil {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}

				log.Printf("Auth State: %v", authState.UserInfo)
				route.Handler(w, r).ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		}
	case Check:
		handler = func(w http.ResponseWriter, r *http.Request) {
			app.Mw.CheckAuthentication()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				route.Handler(w, r).ServeHTTP(w, r)
			})).ServeHTTP(w, r)
		}
	default:
		handler = func(w http.ResponseWriter, r *http.Request) {
			route.Handler(w, r).ServeHTTP(w, r)
		}
	}

	app.Router.HandleFunc(route.Path, handler).Methods(route.Method).Name(route.Path)
}

func (app *App) SetupAuthRoutes() {
	app.Router.Handle("/login", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authURL, err := services.BuildAuthorizeRequest(codeChallenge)
		if err != nil {
			http.Error(w, "Failed to authorize request", http.StatusBadRequest)
			return
		}

		http.Redirect(w, req, authURL.String(), http.StatusFound)
	}))

	app.Router.Handle("/auth/callback", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
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

		// Redirect or respond to the user
		http.Redirect(w, req, "/", http.StatusSeeOther)
	}))

	app.Router.Handle("/auth/logout", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		app.AuthN.Logout(w, req)
	}))
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
	app.SetupAuthRoutes()
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
