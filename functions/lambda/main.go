package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/meetnearme/api/functions/lambda/handlers"
	transport "github.com/meetnearme/api/internal/transport/lambda"
)

var router *transport.Router

func init() {
	router = transport.NewRouter()
	router.GET("/", handlers.GetHomePage)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req)
}

func main() {
	lambda.Start(Router)
}
