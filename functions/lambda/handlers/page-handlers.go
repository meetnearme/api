package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/meetnearme/api/functions/lambda/helpers"
	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/templates/pages"
	"github.com/meetnearme/api/functions/lambda/transport"
)

func GetHomePage(ctx context.Context, r transport.Request, db *dynamodb.Client, clerkAuth *transport.ClerkAuth) transport.Response {
	var events []services.EventSelect
	var err error
	events, err = services.GetEvents(ctx, db)
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	homePage := pages.HomePage(events)
	layoutTemplate := pages.Layout("Home", homePage)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}
}

func GetLoginPage(ctx context.Context, r transport.Request, db *dynamodb.Client, clerkAuth *transport.ClerkAuth) transport.Response {
	loginPage := pages.LoginPage()
	layoutTemplate := pages.Layout("Login", loginPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}
}

func GetEventDetailsPage(ctx context.Context, r transport.Request, db *dynamodb.Client, clerkAuth *transport.ClerkAuth) transport.Response {
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
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}
}

func GetAccountPage(ctx context.Context, r transport.Request, db *dynamodb.Client, clerkAuth *transport.ClerkAuth) transport.Response {
	sessionClaims, ok := clerk.SessionClaimsFromContext(ctx)
	var user *clerk.User
	if ok {
		userID := sessionClaims.Subject
		userData, err := clerkAuth.UserClient.Get(ctx, userID)
		if err != nil {
			return transport.SendHTTPError(&transport.HTTPError{
				Status:  http.StatusInternalServerError,
				Message: err.Error(),
			})
		}
		user = userData
	}
	accountPage := pages.AccountPage(user)
	layoutTemplate := pages.Layout("Account", accountPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}
}
