package main

import (
	"bytes"
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/meetnearme/api/functions/lambda/templates/pages"
	transport "github.com/meetnearme/api/internal/transport/lambda"
)

func handler(ctx context.Context, r transport.Request) (transport.Response, error) {
	loginPage := pages.LoginPage()
	layoutTemplate := pages.App("Meet Near Me - Login", loginPage)
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

func main() {
	lambda.Start(handler)
}
