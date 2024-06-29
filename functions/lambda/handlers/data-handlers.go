package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/lambda/services"
	"github.com/meetnearme/api/functions/lambda/transport"
)

var validate *validator.Validate = validator.New()

func CreateEvent(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	ctx := r.Context()
	var createEvent services.EventInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("ERR: Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal(body, &createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte("ERR: Invalid JSON payload: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte("ERR: Invalid body: "+err.Error()), http.StatusInternalServerError, err)
	}

	res, err := services.InsertEvent(ctx, db, createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte("ERR: Failed to add event: "+err.Error()), http.StatusInternalServerError, err)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte("ERR: marshaling JSON: "+err.Error()), http.StatusInternalServerError, err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new item: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	return transport.SendHtmlSuccess(w, []byte(string(json)), http.StatusCreated)
}
