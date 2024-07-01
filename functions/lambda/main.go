package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"

	"github.com/meetnearme/api/functions/lambda/handlers"
	"github.com/meetnearme/api/functions/lambda/helpers"
	"github.com/meetnearme/api/functions/lambda/transport"
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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello")
	})
	r := mux.NewRouter()
	r.Use(withContext)
	// r.Use(apiGatewayMiddleware) // Add the API Gateway middleware
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Not found", r.RequestURI)
		http.Error(w, fmt.Sprintf("Not found: %s", r.RequestURI), http.StatusNotFound)
	})

	r.HandleFunc("/", makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
		handlers.GetHomePage(w, r, db)
	})).Methods("GET")

	r.HandleFunc("/login", makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
		handlers.GetLoginPage(w, r, db)
	})).Methods("GET")

	r.HandleFunc("/login", makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
		handlers.GetEventDetailsPage(w, r, db)
	})).Methods("GET")

	r.HandleFunc("/events/{eventId}", makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
		handlers.GetEventDetailsPage(w, r, db)
	})).Methods("GET")

	r.HandleFunc("/api/event", makeHandler(func(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) {
		handlers.CreateEvent(w, r, db)
	})).Methods("POST")

	adapter := gorillamux.NewV2(r)

	lambda.Start(func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		// store the original `events.APIGatewayV2HTTPRequest` in context for later access
		// NOTE: original requestContext is available via request.Context().GetValue(apiGwV2ReqKey).RequestContext
		ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, request)
		return adapter.ProxyWithContext(ctx, request)
	})
}
