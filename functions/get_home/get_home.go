package main

import (
	"bytes"
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/meetnearme/api/functions/lambda/views"
)

// aliasing the types to keep lines short
type Request = events.APIGatewayV2HTTPRequest
type Response = events.APIGatewayV2HTTPResponse

func handler(ctx context.Context, r Request) (Response, error) {
	layoutTemplate := views.Layout("Meet Near Me - Home")
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return serverError(err)
	}
	return Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func main() {
	lambda.Start(handler)
}

func serverError(err error) (Response, error) {
	log.Println(err.Error())

	return Response{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}
