package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var db internal_types.DynamoDBAPI

func init() {
	db = transport.CreateDbClient()
}

// UserHandler handles user-related requests
type RegistrationFieldsHandler struct {
	RegistrationFieldsService internal_types.RegistrationFieldsServiceInterface
}

// NewRegistrationHandler creates a new RegistrationHandler with the given RegistrationService
func NewRegistrationFieldsHandler(registrationFieldsService internal_types.RegistrationFieldsServiceInterface) *RegistrationFieldsHandler {
	return &RegistrationFieldsHandler{RegistrationFieldsService: registrationFieldsService}
}

func (h *RegistrationFieldsHandler) CreateRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	var createRegistrationFields internal_types.RegistrationFieldsInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	createRegistrationFields.CreatedAt = time.Now()
	createRegistrationFields.UpdatedAt = time.Now()
	createRegistrationFields.EventId = eventId

	err = json.Unmarshal(body, &createRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&createRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	res, err := h.RegistrationFieldsService.InsertRegistrationFields(r.Context(), db, createRegistrationFields, eventId)
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

// This needs to change for use cases of fetching multiple users based on org ID or other
func (h *RegistrationFieldsHandler) GetRegistrationFieldsByEventID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	registrationFields, err := h.RegistrationFieldsService.GetRegistrationFieldsByEventID(r.Context(), db, eventId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get users: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(registrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationFieldsHandler) UpdateRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	var updateRegistrationFields internal_types.RegistrationFieldsUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updateRegistrationFields.UpdatedAt = time.Now()

	err = json.Unmarshal(body, &updateRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&updateRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedRegistrationFields, err := h.RegistrationFieldsService.UpdateRegistrationFields(r.Context(), db, eventId, updateRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update user: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if updatedRegistrationFields == nil {
		transport.SendServerRes(w, []byte("Registration not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(updatedRegistrationFields)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationFieldsHandler) DeleteRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
		return
	}

	err := h.RegistrationFieldsService.DeleteRegistrationFields(r.Context(), db, eventId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete user: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Registration successfully deleted"), http.StatusOK, nil)
}

func CreateRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := dynamodb_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateRegistrationFields(w, r)
	}
}

// GetRegistrationsHandler is a wrapper that creates the RegistrationHandler and returns the handler function for getting all users
func GetRegistrationFieldsByEventIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := dynamodb_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetRegistrationFieldsByEventID(w, r)
	}
}

// UpdateRegistrationHandler is a wrapper that creates the RegistrationHandler and returns the handler function for updating a user
func UpdateRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := dynamodb_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateRegistrationFields(w, r)
	}
}

// DeleteRegistrationHandler is a wrapper that creates the RegistrationHandler and returns the handler function for deleting a user
func DeleteRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := dynamodb_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteRegistrationFields(w, r)
	}
}
