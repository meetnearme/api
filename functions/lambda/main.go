package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/meetnearme/api/functions/lambda/handlers"
	"github.com/meetnearme/api/functions/lambda/transport"
)

var router *transport.Router
var db *dynamodb.Client
var clerkAuth *transport.ClerkAuth

func init() {
	// Setup DB Client
	db = transport.CreateDbClient()

	// Setup Clerk Auth
	apiKey := os.Getenv("CLERK_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: CLERK_API_KEY environment variable not set")
		return
	}

	config := &clerk.ClientConfig{}
	config.Key = &apiKey

	clerk.SetKey(apiKey)

	clerkAuth = transport.InitClerkAuth(config)

	// Setup Routing
	router = transport.GetRouter()
	router.GET("/", handlers.GetHomePage)
	router.GET("/login", handlers.GetLoginPage)
	router.GET("/signup", handlers.GetSignUpPage)
	router.GET("/events/:eventId", handlers.GetEventDetailsPage)

	router.GET("/account", handlers.GetAccountPage, transport.ParseCookies, transport.RequireHeaderAuthorization)

	router.POST("/api/event", handlers.CreateEvent)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req, db, clerkAuth)
}

func main() {
	lambda.Start(Router)
}
