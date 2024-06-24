package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var router *transport.Router
var db *dynamodb.Client

func init() {
	db = transport.CreateDbClient()
	router = transport.GetRouter()
	router.GET("/", handlers.GetHomePage)
	router.GET("/login", handlers.GetLoginPage)
	router.GET("/admin", handlers.GetAdminPage)
	router.GET("/map", handlers.GetMapEmbedPage)
	router.GET("/events/:eventId", handlers.GetEventDetailsPage)

	router.POST("/api/event", handlers.CreateEvent)
	// TODO: delete this comment once user location is implemented in profile,
	// "/api/location/geo" is for use there
	router.POST("/api/location/geo", handlers.GeoLookup)
	router.POST("/api/seshu/session", handlers.CreateSeshuSession)
	router.POST("/api/seshu/session/submit", handlers.SubmitSeshuSession)
	router.PATCH("/api/seshu/session", handlers.UpdateSeshuSession)
	router.PATCH("/api/seshu/session/location", handlers.GeoThenPatchSeshuSession)
	router.PATCH("/api/seshu/session/events", handlers.SubmitSeshuEvents)
}

func Router(ctx context.Context, req transport.Request) (transport.Response, error) {
	return router.ServeHTTP(ctx, req, db)
}

func main() {
	lambda.Start(Router)
}