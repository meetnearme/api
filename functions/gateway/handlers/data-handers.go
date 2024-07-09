package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var validate *validator.Validate = validator.New()

func CreateEvent(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
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

	res, err := services.InsertEvent(ctx, db, createEvent)

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
	return transport.SendHtmlRes(w, []byte(string(json)), http.StatusCreated, nil)
}

func CreateSeshuSession(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	ctx := r.Context()
	var createSehsuSession services.SeshuSessionInput
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Failed to read request body")), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal(body, &createSehsuSession)

	if err != nil {
		return transport.SendHtmlRes(w,  []byte(string("Invalid JSON payload")), http.StatusUnprocessableEntity, err)
	}

	err = validate.Struct(&createSehsuSession)

	if err != nil {
		return transport.SendHtmlRes(w,  []byte(string("Invalid Body")), http.StatusBadRequest, err)
	}

	res, err := services.InsertSeshuSession(ctx, db, createSehsuSession)

	if err != nil {
		return transport.SendServerRes(w, []byte(string(err.Error())), http.StatusInternalServerError, err)
	}

	json, err := json.Marshal(res)

	if err != nil {
		return transport.SendServerRes(w, []byte(string(err.Error())), http.StatusInternalServerError, err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new seshu session: %+v", res)

	return transport.SendServerRes(w, []byte(string(json)), http.StatusCreated, nil)
}

func UpdateSeshuSession(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	ctx := r.Context()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error reading request body")), http.StatusUnprocessableEntity, err)
	}
	var updateSehsuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSehsuSession)
	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendHtmlRes(w, []byte(string("Invalid JSON payload")), http.StatusUnprocessableEntity, err)
	}

	if (updateSehsuSession.Url == "") {
		var msg = "ERR: Invalid body: url is required"
		log.Println(msg)
		return transport.SendHtmlRes(w, []byte(string(msg)), http.StatusBadRequest, err)
	}

	res, err := services.UpdateSeshuSession(ctx, db, updateSehsuSession)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte(string(err.Error())), http.StatusInternalServerError, err)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		return transport.SendServerRes(w, []byte(string(err.Error())), http.StatusInternalServerError, err)
	}

	// TODO: consider log levels / log volume
	log.Printf("Updated seshu session: %+v", res.Url)

	// TODO: Replace JSON response with htmx template with event data
	return transport.SendServerRes(w, []byte(string(json)), http.StatusCreated, nil)
}
