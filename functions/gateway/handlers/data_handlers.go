package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
    "github.com/meetnearme/api/functions/gateway/types"
)

var validate *validator.Validate = validator.New()

// EventServiceInterface defines the methods we need from the services package
type EventServiceInterface interface {
    InsertEvent(ctx context.Context, db types.DynamoDBAPI, createEvent services.EventInsert) (*services.EventSelect, error)
}

// RealEventService is a wrapper for the actual services package
type RealEventService struct{}

func (s *RealEventService) InsertEvent(ctx context.Context, db types.DynamoDBAPI, createEvent services.EventInsert) (*services.EventSelect, error) {
    return services.InsertEvent(ctx, db, createEvent)
}

// Wrapper function to satisfy the lambdaHandlerFunc type 
func CreateEvent(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        db := Db
        if db == nil {
            db = transport.GetDB()
        }
        createEventHandler(r.Context(), w, r, &RealEventService{}, db)
    }
}

func createEventHandler(ctx context.Context, w  http.ResponseWriter, r *http.Request, eventService EventServiceInterface, db types.DynamoDBAPI) {
	var createEvent services.EventInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
	}

	err = json.Unmarshal(body, &createEvent)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
	}

	err = validate.Struct(&createEvent)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return 
	}

	res, err := eventService.InsertEvent(ctx, db, createEvent)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to add event: "+err.Error()), http.StatusInternalServerError, err)
        return
	}

	json, err := json.Marshal(res)

	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
	}

	// TODO: consider log levels / log volume
	log.Printf("Inserted new item: %+v", res)

	// TODO: Replace JSON response with htmx template with event data
	transport.SendServerRes(w, []byte(string(json)), http.StatusCreated, nil)
}
