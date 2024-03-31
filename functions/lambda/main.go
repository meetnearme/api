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

// https://8hpnqnaevi.execute-api.us-east-1.amazonaws.com/
// https://github.com/raphael-p/beango-messenger/blob/master/server/server.go
// https://raphael-p.medium.com/a-guide-to-making-a-go-web-server-without-a-framework-1439a965f2b1

func init() {
	db = transport.CreateDbClient()
	router = transport.NewRouter()
	router.GET("/", handlers.GetHomePage, transport.LogRequest)
	router.GET("/login", handlers.GetLoginPage)

	router.POST("/api/event", handlers.CreateEvent)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req, db)
}

func main() {
	lambda.Start(Router)
}
