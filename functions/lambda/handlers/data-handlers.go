package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/lambda/services"
)

var validate *validator.Validate = validator.New()

func CreateEvent(w http.ResponseWriter, r *http.Request, db *dynamodb.Client) http.HandlerFunc {
	ctx := r.Context()
	var createEvent services.EventInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		msg := "Failed to read request body: " + err.Error()
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))

		return http.HandlerFunc(nil)
	}

	err = json.Unmarshal(body, &createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		msg := "Invalid JSON payload: " + err.Error()
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))

		return http.HandlerFunc(nil)
	}

	err = validate.Struct(&createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		msg := "Invalid body: " + err.Error()
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))

		return http.HandlerFunc(nil)
	}

	res, err := services.InsertEvent(ctx, db, createEvent)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))

		return http.HandlerFunc(nil)
	}

	json, err := json.Marshal(res)

	// TODO: Update errors to send htmx template with error message
	if err != nil {
		msg := "ERR: marshaling JSON: " + err.Error()
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))

		return http.HandlerFunc(nil)
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new item: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(string(json)))

	return http.HandlerFunc(nil)
}
