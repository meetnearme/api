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


type EventHandler struct {
    EventService services.EventServiceInterface
}

func NewEventHandler(eventService services.EventServiceInterface) *EventHandler {
    return &EventHandler{EventService: eventService}
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
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

    db := transport.GetDB()
    res, err := h.EventService.InsertEvent(r.Context(), db, createEvent)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to add event: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new item: %+v", res)
    transport.SendServerRes(w, json, http.StatusCreated, nil)
}

func CreateEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    eventService := services.NewEventService()
    handler := NewEventHandler(eventService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.CreateEvent(w, r)
    }
}
