package handlers

import (
	"bytes"
	"context"
	"net/http"

	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/templates/pages"
	"github.com/meetnearme/api/functions/lambda/transport"
)

func GetHomePage(ctx context.Context, r transport.Request) (transport.Response, error) {
	var events []services.Event = services.GetEvents()
	homePage := pages.HomePage(events)
	layoutTemplate := pages.App("Home", homePage)
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

func GetLoginPage(ctx context.Context, r transport.Request) (transport.Response, error) {
	loginPage := pages.LoginPage()
	layoutTemplate := pages.App("Login", loginPage)
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
