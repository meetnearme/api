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
)

var mw   *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]
var authN *authentication.Authenticator[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]

// Middleware to inject context into the request
func withContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Add context to request
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()
	r.Use(withContext)

	// used by the auth service but must init in `main` https://go.dev/src/flag/example_test.go?s=933:2300#L33
	flag.Parse()
	services.InitAuth()

	mw, authN = services.GetAuthMw()

	r.Handle("/auth/login", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		query := ""
		if req.URL.RawQuery != "" {
			query = "?" + req.URL.RawQuery
		}
		authN.Authenticate(w, req, req.URL.Path + query)
	}))
	r.Handle("/auth/callback", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authN.Callback(w, req)
	}))
	r.Handle("/auth/logout", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authN.Logout(w, req)
	}))

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Not found", r.RequestURI)
		http.Error(w, fmt.Sprintf("Not found: %s", r.RequestURI), http.StatusNotFound)
	})

	type AuthType string

	const (
		None    AuthType = "none"
		Check   AuthType = "check"
		Require AuthType = "require"
	)

	routes := []struct {
		path    string
		method  string
		handler func(http.ResponseWriter, *http.Request) http.HandlerFunc
		auth    AuthType
	}{
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
		{"/events/{eventId}", "GET", handlers.GetEventDetailsPage, Check},
		{"/api/event", "POST", handlers.CreateEvent, None},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo", "POST", handlers.GeoLookup, None},
		{"/api/html/seshu/session/submit", "POST", handlers.SubmitSeshuSession, None},
		{"/api/html/seshu/session/location", "PATCH", handlers.GeoThenPatchSeshuSession, None},
		{"/api/html/seshu/session/events", "PATCH", handlers.SubmitSeshuEvents, None},
	}

	for _, route := range routes {
		currentRoute := route
		if currentRoute.auth == Require {
			r.HandleFunc(currentRoute.path, func(w http.ResponseWriter, r *http.Request) {
				mw.RequireAuthentication()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if authentication.IsAuthenticated(r.Context()) {
						currentRoute.handler(w, r)
					} else {
						http.Redirect(w, r, "/login", http.StatusFound)
					}
				})).ServeHTTP(w, r)
			}).Methods(currentRoute.method)
		} else if currentRoute.auth == Check {
			r.HandleFunc(currentRoute.path, func(w http.ResponseWriter, r *http.Request) {
				mw.CheckAuthentication()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					currentRoute.handler(w, r)
				})).ServeHTTP(w, r)
			}).Methods(currentRoute.method)
		} else {
			r.HandleFunc(currentRoute.path, func(w http.ResponseWriter, r *http.Request) {
				currentRoute.handler(w, r)
			}).Methods(currentRoute.method)
		}
	}

	adapter := gorillamux.NewV2(r)

	lambda.Start(func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		// store the original `events.APIGatewayV2HTTPRequest` in context for later access
		// NOTE: original requestContext is available via request.Context().GetValue(apiGwV2ReqKey).RequestContext
		ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, request)
		return adapter.ProxyWithContext(ctx, request)
	})
}
