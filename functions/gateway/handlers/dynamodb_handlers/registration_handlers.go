package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

// Validator instance for struct validation
var validate *validator.Validate = validator.New()

func init() {
	db = transport.CreateDbClient()
	log.Printf("db client: %v", db)
}

// UserHandler handles user-related requests
type RegistrationHandler struct {
	RegistrationService internal_types.RegistrationServiceInterface
}

// NewRegistrationHandler creates a new RegistrationHandler with the given RegistrationService
func NewRegistrationHandler(registrationService internal_types.RegistrationServiceInterface) *RegistrationHandler {
	return &RegistrationHandler{RegistrationService: registrationService}
}

func (h *RegistrationHandler) CreateRegistration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
        return
    }
	userId := vars["user_id"]
    if userId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    var createRegistration internal_types.RegistrationInsert
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }


    err = json.Unmarshal(body, &createRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

	createRegistration.CreatedAt = time.Now()
	createRegistration.UpdatedAt = time.Now()
	createRegistration.EventId = eventId
	createRegistration.UserId = userId

    err = validate.Struct(&createRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    res, err := h.RegistrationService.InsertRegistration(r.Context(), db, createRegistration, eventId, userId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create registration fields: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *RegistrationHandler) GetRegistrationByPk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
        return
    }

	userId := vars["user_id"]
    if userId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    registration, err := h.RegistrationService.GetRegistrationByPk(r.Context(), db, eventId, userId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get registrations: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(registration)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

// This needs to change for use cases of fetching multiple users based on org ID or other
func (h *RegistrationHandler) GetRegistrationsByEventID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
        return
    }

    registration, err := h.RegistrationService.GetRegistrationsByEventID(r.Context(), db, eventId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get registrations: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(registration)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationHandler) GetRegistrationsByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["user_id"]
    if userId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    registration, err := h.RegistrationService.GetRegistrationsByUserID(r.Context(), db, userId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get registrations: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(registration)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationHandler) UpdateRegistration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
        return
    }

	userId := vars["user_id"]
    if userId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    var updateRegistration internal_types.RegistrationUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

	updateRegistration.UpdatedAt = time.Now()

    err = json.Unmarshal(body, &updateRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

	updateRegistration.UserId = userId
	updateRegistration.EventId = eventId

    err = validate.Struct(&updateRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    updatedRegistration, err := h.RegistrationService.UpdateRegistration(r.Context(), db, eventId, userId, updateRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if updatedRegistration == nil {
        transport.SendServerRes(w, []byte("Registration not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(updatedRegistration)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationHandler) DeleteRegistration(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
    if eventId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

	userId := vars["user_id"]
    if userId == "" {
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    err := h.RegistrationService.DeleteRegistration(r.Context(), db, eventId, userId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("Registration successfully deleted"), http.StatusOK, nil)
}


func CreateRegistrationHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	log.Printf("in reg fields wrapper")
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateRegistration(w, r)
	}
}

// GetRegistrationsHandler is a wrapper that creates the RegistrationHandler and returns the handler function for getting all users
func GetRegistrationByPkHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetRegistrationByPk(w, r)
	}
}

// GetRegistrationsHandler is a wrapper that creates the RegistrationHandler and returns the handler function for getting all users
func GetRegistrationsByEventIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetRegistrationsByEventID(w, r)
	}
}

func GetRegistrationsByUserIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetRegistrationsByUserID(w, r)
	}
}


// UpdateRegistrationHandler is a wrapper that creates the RegistrationHandler and returns the handler function for updating a user
func UpdateRegistrationHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateRegistration(w, r)
	}
}

// DeleteRegistrationHandler is a wrapper that creates the RegistrationHandler and returns the handler function for deleting a user
func DeleteRegistrationHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationService := dynamodb_service.NewRegistrationService()
	handler := NewRegistrationHandler(registrationService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteRegistration(w, r)
	}
}
