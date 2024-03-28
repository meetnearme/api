package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/meetnearme/api/functions/lambda/handlers"
	"github.com/meetnearme/api/functions/lambda/transport"
)

var router *transport.Router

// https://8hpnqnaevi.execute-api.us-east-1.amazonaws.com/
// https://github.com/raphael-p/beango-messenger/blob/master/server/server.go
// https://raphael-p.medium.com/a-guide-to-making-a-go-web-server-without-a-framework-1439a965f2b1

func init() {
	router = transport.NewRouter()
	router.GET("/", handlers.GetHomePage)
	router.GET("/login", handlers.GetLoginPage)
	router.OPTIONS("/login", handlers.GetLoginPage)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req)
}

func main() {
	lambda.Start(Router)
}
