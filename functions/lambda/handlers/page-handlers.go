package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/lambda/helpers"
	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/templates/pages"
)

func GetHomePage(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()
	apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)

	queryParameters := apiGwV2Req.QueryStringParameters
	startTimeStr := queryParameters["start_time"]
	endTimeStr := queryParameters["end_time"]
	latStr := queryParameters["lat"]
	lonStr := queryParameters["lon"]
	radiusStr := queryParameters["radius"]

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
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))

		return http.HandlerFunc(nil)
	}

	homePage := pages.HomePage(events)
	layoutTemplate := pages.Layout("Home", homePage)

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))

		return http.HandlerFunc(nil)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())

	return http.HandlerFunc(nil)
}

func GetLoginPage(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	ctx := r.Context()
	loginPage := pages.LoginPage()
	layoutTemplate := pages.Layout("Login", loginPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))

		return http.HandlerFunc(nil)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())

	return http.HandlerFunc(nil)
}

func GetEventDetailsPage(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	// TODO: Extract reading param values into a helper method.
	ctx := r.Context()
	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]

	if eventId == "" {
		// TODO: If no eventID is passed, return a 404 page or redirect to events list.
		fmt.Println("Event Id not found")
	}

	eventDetailsPage := pages.EventDetailsPage(eventId)
	layoutTemplate := pages.Layout("Event Details", eventDetailsPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))

		return http.HandlerFunc(nil)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())

	return http.HandlerFunc(nil)
}
