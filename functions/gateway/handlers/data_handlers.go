package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var validate *validator.Validate = validator.New()


type MarqoHandler struct {
    MarqoService services.MarqoServiceInterface
}

func NewMarqoHandler(marqoService services.MarqoServiceInterface) *MarqoHandler {
    return &MarqoHandler{MarqoService: marqoService}
}

// Create a new struct for raw JSON operations
type rawEventData struct {
    Id          string        `json:"id"`
    EventOwners []string      `json:"eventOwners" validate:"required,min=1"`
    Name        string        `json:"name" validate:"required"`
    Description string        `json:"description"`
    Address     string        `json:"address"`
    Lat         float64       `json:"lat"`
    Long        float64       `json:"long"`
}

type rawEvent struct {
    rawEventData
    StartTime interface{} `json:"startTime" validate:"required"`
    EndTime   interface{} `json:"endTime,omitempty"`
}

func ConvertRawEventToEvent(raw rawEvent) (services.Event, error) {
    event := services.Event{
        Id:          raw.Id,
        EventOwners: raw.EventOwners,
        Name:        raw.Name,
        Description: raw.Description,
        Address:     raw.Address,
        Lat:         raw.Lat,
        Long:        raw.Long,
    }
    if raw.StartTime == nil {
        return services.Event{}, fmt.Errorf("startTime is required")
    }
    startTime, err := helpers.UtcOrUnixToUnix64(raw.StartTime)
    if err != nil || startTime == 0 {
        return services.Event{}, fmt.Errorf("invalid StartTime: %w", err)
    }
    event.StartTime = startTime

    if raw.EndTime != nil {
        endTime, err := helpers.UtcOrUnixToUnix64(raw.EndTime)
        if err != nil || endTime == 0 {
            return services.Event{}, fmt.Errorf("invalid EndTime: %w", err)
        } else {
            event.EndTime = nil
        }
    }
    return event, nil
}

func (h *MarqoHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
    var raw rawEvent

    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &raw)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    createEvent, err := ConvertRawEventToEvent(raw)
    if err != nil {
        transport.SendServerRes(w, []byte(err.Error()), http.StatusBadRequest, err)
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
        transport.SendServerRes(w, []byte("Failed to upsert event: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostEvent(w, r)
    }
}

func HandleBatchEventValidation(w http.ResponseWriter, r *http.Request) ([]services.Event, int, error) {
    var payload struct {
        Events []rawEvent `json:"events"`
    }
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return nil, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err)
    }

    err = json.Unmarshal(body, &payload)
    if err != nil {
        return nil, http.StatusUnprocessableEntity, fmt.Errorf("invalid JSON payload: %w", err)
    }

    err = validate.Struct(&payload)
    if err != nil {
        return nil, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err)
    }

    events := make([]services.Event, len(payload.Events))
    for i, rawEvent := range payload.Events {
        if len(rawEvent.EventOwners) == 0 {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d is missing eventOwners", i)
        }
        event, err := ConvertRawEventToEvent(rawEvent)
        if err != nil {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid event at index %d: %s", i, err.Error())
        }
        events[i] = event
    }

    return events, http.StatusOK, nil
}

func (h *MarqoHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
    // TODO: use this in BatchUpdateEvent / UpdateOneEvent  handler but add conditional logic to check for ID which is required in Update and Forbidden in PostBatchEvents
    events, status, err := HandleBatchEventValidation(w, r)

    if err != nil {
        transport.SendServerRes(w, []byte(err.Error()), status, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.BulkUpsertEventToMarqo(marqoClient, events)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to upsert events: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostBatchEvents(w, r)
    }
}

func (h *MarqoHandler) GetOneEvent(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }
    eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
    var event *services.Event
    event, err = services.GetMarqoEventByID(marqoClient, eventId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(event)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }
    transport.SendServerRes(w, json, http.StatusOK, nil)
}

func GetOneEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.GetOneEvent(w, r)
    }
}

func (h *MarqoHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    // NOTE: these defaults are random, we should fix
    latFloat := float64(38.8951)
    longFloat := float64(-77.0364)
    maxDistance := float64(300)

    query := r.URL.Query().Get("q")
    lat := r.URL.Query().Get("lat")
    if lat != "" {
        latFloat, _ = strconv.ParseFloat(lat, 64)
    }
    long := r.URL.Query().Get("lon")
    if long != "" {
        longFloat, _ = strconv.ParseFloat(long, 64)
    }

    // TODO: add start time here,
    // need to convert marqo DB index to unix for `startTime` / `endTime`

    var res services.EventSearchResponse
    res, err = services.SearchMarqoEvents(marqoClient, query, []float64{latFloat, longFloat}, maxDistance, []string{})
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.SearchEvents(w, r)
    }
}
