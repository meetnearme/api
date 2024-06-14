package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func GetHomePage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
    // Extract parameter values from the request query parameters
    startTimeStr := r.QueryStringParameters["start_time"]
    endTimeStr := r.QueryStringParameters["end_time"]
    latStr := r.QueryStringParameters["lat"]
    lonStr := r.QueryStringParameters["lon"]
    radiusStr := r.QueryStringParameters["radius"]

    // Set default values if query parameters are not provided
    startTime := time.Now()
    endTime := startTime.AddDate(100, 0, 0)
    lat := float32(39.8283)
    lon := float32(-98.5795)
    radius := float32(2500.0)

    // Parse parameter values if provided
    if startTimeStr != "" {
        startTime, _ = time.Parse(time.RFC3339, startTimeStr)
    }
    if endTimeStr != "" {
        endTime, _ = time.Parse(time.RFC3339, endTimeStr)
    }
    if latStr != "" {
        lat64, _ := strconv.ParseFloat(latStr, 32)
        lat = float32(lat64)
    }
    if lonStr != "" {
        lon64, _ := strconv.ParseFloat(lonStr, 32)
        lon = float32(lon64)
    }
    if radiusStr != "" {
        radius64, _ := strconv.ParseFloat(radiusStr, 32)
        radius = float32(radius64)
    }

    // Call the GetEventsZOrder service to retrieve events
    events, err := services.GetEventsZOrder(ctx, db, startTime, endTime, lat, lon, radius)
    if err != nil {
        return transport.SendServerError(err)
    }

	homePage := pages.HomePage(events)
	layoutTemplate := pages.Layout("Home", homePage)

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}

	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func GetLoginPage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	loginPage := pages.LoginPage()
	layoutTemplate := pages.Layout("Login", loginPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func GetMapEmbedPage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	mapEmbedPage := pages.MapEmbedPage(r.QueryStringParameters["address"])
	layoutTemplate := pages.Layout("Embed", mapEmbedPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func GetEventDetailsPage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	// TODO: Extract reading param values into a helper method.
	eventId, error := ctx.Value(helpers.EVENT_ID_KEY).(string)
	if error {
		// If no eventID is passed, return a 404 page or redirect to events list.
		fmt.Println("Event Id not found")
	}
	eventDetailsPage := pages.EventDetailsPage(eventId)
	layoutTemplate := pages.Layout("Event Details", eventDetailsPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func GetAdminPage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	adminPage := pages.AdminPage()
	layoutTemplate := pages.Layout("Admin", adminPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}
