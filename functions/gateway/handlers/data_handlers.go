package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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
    Id              string   `json:"id"`
    EventOwners     []string `json:"eventOwners" validate:"required,min=1"`
    Name            string   `json:"name" validate:"required"`
    Description     string   `json:"description"`
    Address         string   `json:"address"`
    Lat             float64   `json:"lat"`
    Long            float64   `json:"long"`
}

type rawEvent struct {
    rawEventData
    StartTime interface{} `json:"startTime" validate:"required"`
    EndTime   interface{} `json:"endTime,omitempty"`
    StartingPrice   *int32    `json:"startingPrice,omitempty"`
    Currency        *string     `json:"currency,omitempty"`
    PayeeId         *string     `json:"payeeId,omitempty"`
	HasRegistrationFields *bool `json:"hasRegistrationFields,omitempty"`
	HasPurchasable *bool  `json:"hasPurchasable,omitempty"`
	ImageUrl      *string `json:"imageUrl,omitempty"`
	Timezone      *string `json:"timezone,omitempty"`
	CreatedAt     *int64 `json:"createdAt,omitempty"`
	UpdatedAt     *int64 `json:"updatedAt,omitempty"`
	UpdatedBy     *string `json:"updatedBy,omitempty"`
}

func ConvertRawEventToEvent(raw rawEvent, requireId bool) (services.Event, error) {
    event := services.Event{
        Id:          raw.Id,
        EventOwners: raw.EventOwners,
        Name:        raw.Name,
        Description: raw.Description,
        Address:     raw.Address,
        Lat:         raw.Lat,
        Long:        raw.Long,
    }

    // Safely assign pointer values
    if raw.StartingPrice != nil {
        event.StartingPrice = *raw.StartingPrice
    }
    if raw.Currency != nil {
        event.Currency = *raw.Currency
    }
    if raw.PayeeId != nil {
        event.PayeeId = *raw.PayeeId
    }
    if raw.HasRegistrationFields != nil {
        event.HasRegistrationFields = *raw.HasRegistrationFields
    }
    if raw.HasPurchasable != nil {
        event.HasPurchasable = *raw.HasPurchasable
    }
    if raw.ImageUrl != nil {
        event.ImageUrl = *raw.ImageUrl
    }
    if raw.Timezone != nil {
        event.Timezone = *raw.Timezone
    }
    if raw.CreatedAt != nil {
        event.CreatedAt = *raw.CreatedAt
    }
    if raw.UpdatedAt != nil {
        event.UpdatedAt = *raw.UpdatedAt
    }
    if raw.UpdatedBy != nil {
        event.UpdatedBy = *raw.UpdatedBy
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
        }
    }
    if raw.PayeeId != nil || raw.StartingPrice != nil || raw.Currency != nil {

        if (raw.PayeeId == nil || raw.StartingPrice == nil || raw.Currency == nil) {
            return services.Event{}, fmt.Errorf("all of 'PayeeId', 'StartingPrice', and 'Currency' are required if any are present")
        }

        if raw.PayeeId != nil {
            event.PayeeId = *raw.PayeeId
        }
        if raw.Currency != nil {
            event.Currency = *raw.Currency
        }
        if raw.StartingPrice != nil {
            event.StartingPrice = *raw.StartingPrice
        }
    }
    return event, nil
}

func ValidateSingleEventPaylod(w http.ResponseWriter, r *http.Request, requireId bool) (event services.Event, status int, err error) {
    var raw rawEvent

    body, err := io.ReadAll(r.Body)
    if err != nil {
        return services.Event{}, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err)
    }

    err = json.Unmarshal(body, &raw)
    if err != nil {
        return services.Event{}, http.StatusUnprocessableEntity, fmt.Errorf("invalid JSON payload: %w", err)
    }

    event, err = ConvertRawEventToEvent(raw, requireId)
    if err != nil {
        return services.Event{}, http.StatusBadRequest, fmt.Errorf("failed to convert raw event: %w", err)
    }

    err = validate.Struct(&event)
    if err != nil {
        return services.Event{}, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err)
    }

    return event, status, nil
}


func (h *MarqoHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
    createEvent, status, err := ValidateSingleEventPaylod(w, r, false)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    createEvents := []services.Event{createEvent}

    res, err := services.BulkUpsertEventToMarqo(marqoClient, createEvents, false)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to upsert event: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, json, http.StatusCreated, nil)
}


func PostEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostEvent(w, r)
    }
}

func HandleBatchEventValidation(w http.ResponseWriter, r *http.Request, requireIds bool) ([]services.Event, int, error) {
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
        if requireIds && rawEvent.Id == "" {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d has no id", i)
        }
        if len(rawEvent.EventOwners) == 0 {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d is missing eventOwners", i)
        }

        event, err := ConvertRawEventToEvent(rawEvent, requireIds)
        if err != nil {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid event at index %d: %s", i, err.Error())
        }
        events[i] = event
    }

    return events, http.StatusOK, nil
}

func (h *MarqoHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
    events, status, err := HandleBatchEventValidation(w, r, false)

    if err != nil {
        transport.SendServerRes(w, []byte(err.Error()), status, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.BulkUpsertEventToMarqo(marqoClient, events, false)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to upsert events: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }
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

func (h *MarqoHandler) BulkUpdateEvents(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    events, status, err := HandleBatchEventValidation(w, r, true)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
        return
    }


    res, err := services.BulkUpdateMarqoEventByID(marqoClient, events)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }
    transport.SendServerRes(w, json, http.StatusOK, nil)
}

func BulkUpdateEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.BulkUpdateEvents(w, r)
    }
}

func SearchLocationsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        query := r.URL.Query().Get("q")

        // URL decode the query
        decodedQuery, err := url.QueryUnescape(query)
        if err != nil {
            transport.SendServerRes(w, []byte("Failed to decode query"), http.StatusBadRequest, err)
            return
        }

        // Search for matching cities
        query = strings.ToLower(decodedQuery)
        matches := helpers.SearchCitiesIndexed(query)

        // Prepare the response
        var jsonResponse []byte

        if len(matches) < 1 {
            jsonResponse = []byte("[]")
        } else {
            jsonResponse, err = json.Marshal(matches)
            if err != nil {
                transport.SendServerRes(w, []byte("Failed to create JSON response"), http.StatusInternalServerError, err)
                return
            }
        }

        w.Header().Set("Content-Type", "application/json")
        transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
    }
}

func (h *MarqoHandler) UpdateOneEvent(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Event must have an id "), http.StatusInternalServerError, err)
        return
    }

    updateEvent, status, err := ValidateSingleEventPaylod(w, r, false)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
        return
    }

    updateEvent.Id = eventId
    updateEvents := []services.Event{updateEvent}

    res, err := services.BulkUpdateMarqoEventByID(marqoClient, updateEvents)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    json, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }
    transport.SendServerRes(w, json, http.StatusOK, nil)
}


func UpdateOneEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.UpdateOneEvent(w, r)
    }
}

func (h *MarqoHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
    // Extract parameter values from the request query parameters
    q, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories := GetSearchParamsFromReq(r)

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    var res services.EventSearchResponse
    res, err = services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories)
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
