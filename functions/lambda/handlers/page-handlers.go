package handlers

import (
	"bytes"
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/templates/pages"
	"github.com/meetnearme/api/functions/lambda/transport"
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
