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
    None AuthType = "none"
    Check AuthType = "check"
    Require AuthType = "require"
)

type Route struct {
    Path string
    Method  string
    Handler func(http.ResponseWriter, *http.Request) http.HandlerFunc
    Auth    AuthType
}

var Routes []Route

func init() {
    Routes = []Route{
        {"/", "GET", handlers.GetHomePage, Check},
		{"/login", "GET", handlers.GetLoginPage, Check},
		{"/admin/add-event-source", "GET", handlers.GetAddEventSourcePage, Require},
		{"/admin/profile", "GET", handlers.GetProfilePage, Require},
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
		{"/api/db/tables", "GET", handlers.GetDatabaseTablesHandler, None},
    }
}


type App struct {
    Router *mux.Router
     Mw   *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
    AuthN *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
}


func NewApp() *App {
    app := &App{
        Router: mux.NewRouter(),
    }
    log.Printf("App created: %+v", app)
    app.Router.Use(withContext)
    app.InitializeAuth()
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
                if authentication.IsAuthenticated(r.Context()) {
                    route.Handler(w, r).ServeHTTP(w, r)
                } else {
                    http.Redirect(w, r, "/login", http.StatusFound)
                }

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
    app.Router.Handle("/auth/login", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        query := ""
        if req.URL.RawQuery != "" {
            query = "?" + req.URL.RawQuery
        }
        app.AuthN.Authenticate(w, req, req.URL.Path+query)
    }))
    app.Router.Handle("/auth/callback", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		app.AuthN.Callback(w, req)
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
                        Path: r.URL.Path,
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
