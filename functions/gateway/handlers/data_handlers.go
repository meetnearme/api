package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
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

func (h *EventHandler) PostEvents(w http.ResponseWriter, r *http.Request) {
    var createEvent services.Event
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

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.UpsertEventToMarqo(marqoClient, createEvent)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to upsert event to marqo: "+err.Error()), http.StatusInternalServerError, err)
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

func PostEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    eventService := services.NewEventService()
    handler := NewEventHandler(eventService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostEvents(w, r)
    }
}

func (h *EventHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Events []services.Event `json:"events"`
    }
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &payload)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    err = validate.Struct(&payload)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.BulkUpsertEventToMarqo(marqoClient, payload.Events)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to upsert event to marqo: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new items: %+v", res)
    transport.SendServerRes(w, json, http.StatusCreated, nil)
}

func PostBatchEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    eventService := services.NewEventService()
    handler := NewEventHandler(eventService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostBatchEvents(w, r)
    }
}

// func SearchMarqoEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64) ([]Event, error) {

func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }
    eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
    // Get the 'q' query parameter value
    query := r.URL.Query().Get("q")

    var res []services.Event
    if eventId != "" {
        var event services.Event
        event, err = services.GetMarqoEventByID(marqoClient, eventId)
        res = append(res, event)
    } else {
        // TODO: get user location from input payload, hardcoding for now
        res, err = services.SearchMarqoEvents(marqoClient, query, []float64{38.8951, -77.0364}, 300, []string{})
    }
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to search marqo events: "+err.Error()), http.StatusInternalServerError, err)
        return
    }
    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }
    transport.SendServerRes(w, json, http.StatusOK, nil)
}

func SearchEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    eventService := services.NewEventService()
    handler := NewEventHandler(eventService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.SearchEvents(w, r)
    }
}
