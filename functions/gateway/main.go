package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"

	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var db *dynamodb.Client


func init() {
	db = transport.CreateDbClient()
}

// Middleware to inject context into the request
func withContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Add context to request
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// Wrapper to convert a handler function to one that accepts http.Request
func makeHandler(fn func(http.ResponseWriter, *http.Request, *dynamodb.Client)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, db)
	}
}

func main() {
	r := mux.NewRouter()
	r.Use(withContext)
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Not found", r.RequestURI)
		http.Error(w, fmt.Sprintf("Not found: %s", r.RequestURI), http.StatusNotFound)
	})

	routes := []struct {
		path    string
		method  string
		handler func(http.ResponseWriter, *http.Request, *dynamodb.Client) http.HandlerFunc
	}{
		{"/", "GET", handlers.GetHomePage},
		{"/admin", "GET", handlers.GetAdminPage},
		{"/login", "GET", handlers.GetLoginPage},
		{"/login", "GET", handlers.GetEventDetailsPage},
		{"/events/{eventId}", "GET", handlers.GetEventDetailsPage},
		{"/api/event", "POST", handlers.CreateEvent},
		// TODO: delete this comment once user location is implemented in profile,
		// "/api/location/geo" is for use there
		{"/api/location/geo", "POST", handlers.GeoLookup},
		{"/api/seshu/session", "POST", handlers.CreateSeshuSession},
		{"/api/seshu/session/submit", "POST", handlers.SubmitSeshuSession},
		{"/api/seshu/session", "PATCH", handlers.UpdateSeshuSession},
		{"/api/seshu/session/location", "PATCH", handlers.UpdateSeshuSession},
		{"/api/seshu/session/location", "PATCH", handlers.GeoThenPatchSeshuSession},
		{"/api/seshu/session/events", "PATCH", handlers.SubmitSeshuEvents},
	}

	for _, route := range routes {
		currentRoute := route
		r.HandleFunc(currentRoute.path, makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
				currentRoute.handler(w, r, db)
			})).Methods(currentRoute.method)
	}

	adapter := gorillamux.NewV2(r)

	lambda.Start(func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		// store the original `events.APIGatewayV2HTTPRequest` in context for later access
		// NOTE: original requestContext is available via request.Context().GetValue(apiGwV2ReqKey).RequestContext
		ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, request)
		return adapter.ProxyWithContext(ctx, request)
	})
}
