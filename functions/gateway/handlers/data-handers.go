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
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var validate *validator.Validate = validator.New()

func CreateEvent(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var createEvent services.EventInsert
	err := json.Unmarshal([]byte(r.Body), &createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	err = validate.Struct(&createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return transport.SendClientError(http.StatusBadRequest, "Invalid Body")
	}

	res, err := services.InsertEvent(ctx, db, createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
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
	}, nil
}

func CreateSeshuSession(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var createSehsuSession services.SeshuSessionInput
	err := json.Unmarshal([]byte(r.Body), &createSehsuSession)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	err = validate.Struct(&createSehsuSession)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return transport.SendClientError(http.StatusBadRequest, "Invalid Body")
	}

	res, err := services.InsertSeshuSession(ctx, db, createSehsuSession)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new seshu session: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/user/%s", "hello res"),
		},
	}, nil
}

func UpdateSeshuSession(ctx context.Context, r transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var updateSehsuSession services.SeshuSessionUpdate
	err := json.Unmarshal([]byte(r.Body), &updateSehsuSession)
	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	if (updateSehsuSession.Url == "") {
		var msg = "ERR: Invalid body: url is required"
		log.Println(msg)
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	res, err := services.UpdateSeshuSession(ctx, db, updateSehsuSession)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerError(err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Updated seshu session: %+v", res.Url)

	// TODO: Replace JSON response with htmx template with event data
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Location": fmt.Sprintf("/user/%s", "hello res"),
		},
	}, nil
}
