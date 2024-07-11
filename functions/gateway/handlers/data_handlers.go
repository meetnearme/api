package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var validate *validator.Validate = validator.New()

func CreateEvent(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var createEvent services.EventInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal(body, &createEvent)

	if err != nil {
		return transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&createEvent)

	if err != nil {
		return transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusInternalServerError, err)
	}

	res, err := services.InsertEvent(ctx, Db, createEvent)

	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to add event: "+err.Error()), http.StatusInternalServerError, err)
	}

	json, err := json.Marshal(res)

	if err != nil {
		return transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new item: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	return transport.SendServerRes(w, []byte(string(json)), http.StatusCreated, nil)
}
