package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func GetHomePage(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var events []services.EventSelect
	var err error
	events, err = services.GetEvents(ctx, db)
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
