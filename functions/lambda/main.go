package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/lambda/handlers"
	"github.com/meetnearme/api/functions/lambda/transport"
)

var router *transport.Router
var db *dynamodb.Client

func init() {
	db = transport.CreateDbClient()
	router = transport.GetRouter()
	router.GET("/", handlers.GetHomePage)
	router.GET("/login", handlers.GetLoginPage)
	router.GET("/events/:eventId", handlers.GetEventDetailsPage)

	router.POST("/api/event", handlers.CreateEvent)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req, db)
}

func main() {
	lambda.Start(Router)
}
