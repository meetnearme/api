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

func (h *MarqoHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
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

func (h *MarqoHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Events []services.Event
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

    // Additional validation for EventOwners
    for i, event := range payload.Events {
        if len(event.EventOwners) == 0 {
            transport.SendServerRes(w, []byte(fmt.Sprintf("Invalid body: Event at index %d is missing EventOwners", i)), http.StatusBadRequest, nil)
            return
        }
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.BulkUpsertEventToMarqo(marqoClient, payload.Events)
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
