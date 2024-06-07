package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/transport"
)

var validate *validator.Validate = validator.New()

func CreateEvent(ctx context.Context, r transport.Request, db *dynamodb.Client, clerkAuth *transport.ClerkAuth) transport.Response {
	var createEvent services.EventInsert
	err := json.Unmarshal([]byte(r.Body), &createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendHTTPError(&transport.HTTPError{
			Status:          http.StatusUnprocessableEntity,
			Message:         "Invalid JSON payload",
			ErrorComponent:  nil,
			ResponseHeaders: map[string]string{"Content-Type": "application/json"},
		})
	}

	err = validate.Struct(&createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusBadRequest,
			Message:        "Invalid body",
			ErrorComponent: nil,
		})
	}

	res, err := services.InsertEvent(ctx, db, createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendHTTPError(&transport.HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new item: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/user/%s", "hello res"),
		},
	}
}
