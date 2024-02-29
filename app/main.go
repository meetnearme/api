package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/meetnearme/api/app/handlers"
)

var ginLambda *ginadapter.GinLambdaV2

func shouldWrapLayout() gin.HandlerFunc {
	return func(c *gin.Context) {
		hxRequestHeader := c.Request.Header["Hx-Request"]
		isHxRequest := hxRequestHeader != nil && hxRequestHeader[0] == "true"
		c.Set("shouldWrapLayout", !isHxRequest)
		c.Next()
	}
}

func init() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.Use(shouldWrapLayout())
	r.GET("/", handlers.GetEventsPageContent)
	r.GET("/events", handlers.GetEventsPageContent)
	r.GET("/account", handlers.GetAccountPageContent)
	r.GET("/login", handlers.GetLoginPageContent)

	componentRouterGroup := r.Group("/components")
	{
		componentRouterGroup.GET("/login-form", handlers.GetLoginFormComponent)
		componentRouterGroup.GET("/events-list", handlers.GetEventsList)
	}
	r.SetTrustedProxies(nil)

	ginLambda = ginadapter.NewV2(r)
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
