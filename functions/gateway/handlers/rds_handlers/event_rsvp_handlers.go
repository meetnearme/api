package rds_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type EventRsvpHandler struct {
	EventRsvpService internal_types.EventRsvpServiceInterface
}

func NewEventRsvpHandler(eventRsvpService internal_types.EventRsvpServiceInterface) *EventRsvpHandler {
	return &EventRsvpHandler{EventRsvpService: eventRsvpService}
}


func (h *EventRsvpHandler) CreateEventRsvp(w http.ResponseWriter, r *http.Request) {
	var createEventRsvp internal_types.EventRsvpInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createEventRsvp)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
	}
	log.Printf("body of eventRsvp insert: %v", createEventRsvp)

	err = validate.Struct(&createEventRsvp)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
	}

	// TODO: check if these are redundant in all Insert functions because of NOW() from sql
    now := time.Now().UTC().Format(time.RFC3339)
    createEventRsvp.CreatedAt = now
    createEventRsvp.UpdatedAt = now

	// Parse timestamps
	createdAtTime, err := time.Parse(time.RFC3339, createEventRsvp.CreatedAt)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid created_at timestamp: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedAtTime := createdAtTime // Default to the same value if not provided
	if createEventRsvp.UpdatedAt != "" {
		updatedAtTime, err = time.Parse(time.RFC3339, createEventRsvp.UpdatedAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid updated_at timestamp: "+err.Error()), http.StatusBadRequest, err)
			return
		}
	}

	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Format timestamps for RDS
	createEventRsvp.CreatedAt = createdAtTime.Format(rdsTimeFormat)
	createEventRsvp.UpdatedAt = updatedAtTime.Format(rdsTimeFormat)

    db := transport.GetRdsDB()
    res, err := h.EventRsvpService.InsertEventRsvp(r.Context(), db, createEventRsvp)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create eventRsvp: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new eventRsvp: %+v", res)
    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *EventRsvpHandler) GetEventRsvp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing eventRsvp ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.EventRsvpService.GetEventRsvpByID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("EventRsvp not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *EventRsvpHandler) GetEventRsvpsByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("vars get by user id: %v", vars)
	id := vars["user_id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user_id ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    users, err := h.EventRsvpService.GetEventRsvpsByUserID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user's eventRsvps: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(users)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *EventRsvpHandler) GetEventRsvpsByEventID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("vars get by event id: %v", vars)
	id := vars["event_id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing event_id ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    events, err := h.EventRsvpService.GetEventRsvpsByEventID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user's eventRsvps: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(events)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *EventRsvpHandler) UpdateEventRsvp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing eventRsvp ID"), http.StatusBadRequest, nil)
        return
    }

    var updateEventRsvp internal_types.EventRsvpUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &updateEventRsvp)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    err = validate.Struct(&updateEventRsvp)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.EventRsvpService.UpdateEventRsvp(r.Context(), db, id, updateEventRsvp)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update eventRsvp: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("EventRsvp not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *EventRsvpHandler) DeleteEventRsvp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Vars in delete: %v", vars)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing eventRsvp ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    err := h.EventRsvpService.DeleteEventRsvp(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete eventRsvp: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("EventRsvp successfully deleted"), http.StatusOK, nil)
}

func CreateEventRsvpHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateEventRsvp(w, r)
	}
}


// GetEventRsvpHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventRsvp by ID
func GetEventRsvpHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetEventRsvp(w, r)
	}
}

// GetEventRsvpsHandler is a wrapper that creates the UserHandler and returns the handler function for getting all eventRsvps
func GetEventRsvpsByEventIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetEventRsvpsByEventID(w, r)
	}
}

func GetEventRsvpsByUserIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetEventRsvpsByUserID(w, r)
	}
}

// UpdateEventRsvpHandler is a wrapper that creates the UserHandler and returns the handler function for updating a eventRsvp
func UpdateEventRsvpHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateEventRsvp(w, r)
	}
}

// DeleteEventRsvpHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a eventRsvp
func DeleteEventRsvpHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventRsvpService := rds_service.NewEventRsvpService()
	handler := NewEventRsvpHandler(eventRsvpService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteEventRsvp(w, r)
	}
}
