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

type RegistrationFieldsHandler struct {
	RegistrationFieldsService internal_types.RegistrationFieldsServiceInterface
}

func NewRegistrationFieldsHandler(registrationFieldsService internal_types.RegistrationFieldsServiceInterface) *RegistrationFieldsHandler {
	return &RegistrationFieldsHandler{RegistrationFieldsService: registrationFieldsService}
}


func (h *RegistrationFieldsHandler) CreateRegistrationFields(w http.ResponseWriter, r *http.Request) {
	var createRegistrationFields internal_types.RegistrationFieldsInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createRegistrationFields)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
	}
	log.Printf("body of registrationFields insert: %v", createRegistrationFields)

	err = validate.Struct(&createRegistrationFields)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
	}

	// TODO: check if these are redundant in all Insert functions because of NOW() from sql
    now := time.Now().UTC().Format(time.RFC3339)
    createRegistrationFields.CreatedAt = now
    createRegistrationFields.UpdatedAt = now

	// Parse timestamps
	createdAtTime, err := time.Parse(time.RFC3339, createRegistrationFields.CreatedAt)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid created_at timestamp: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedAtTime := createdAtTime // Default to the same value if not provided
	if createRegistrationFields.UpdatedAt != "" {
		updatedAtTime, err = time.Parse(time.RFC3339, createRegistrationFields.UpdatedAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid updated_at timestamp: "+err.Error()), http.StatusBadRequest, err)
			return
		}
	}

	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Format timestamps for RDS
	createRegistrationFields.CreatedAt = createdAtTime.Format(rdsTimeFormat)
	createRegistrationFields.UpdatedAt = updatedAtTime.Format(rdsTimeFormat)

    db := transport.GetRdsDB()
    res, err := h.RegistrationFieldsService.InsertRegistrationFields(r.Context(), db, createRegistrationFields)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create registrationFields: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new registrationFields: %+v", res)
    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *RegistrationFieldsHandler) GetRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing registrationFields ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.RegistrationFieldsService.GetRegistrationFieldsByID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("RegistrationFields not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationFieldsHandler) UpdateRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing registrationFields ID"), http.StatusBadRequest, nil)
        return
    }

    var updateRegistrationFields internal_types.RegistrationFieldsUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

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

    db := transport.GetRdsDB()
    user, err := h.RegistrationFieldsService.UpdateRegistrationFields(r.Context(), db, id, updateRegistrationFields)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update registrationFields: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("RegistrationFields not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *RegistrationFieldsHandler) DeleteRegistrationFields(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Vars in delete: %v", vars)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing registrationFields ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    err := h.RegistrationFieldsService.DeleteRegistrationFields(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete registrationFields: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("RegistrationFields successfully deleted"), http.StatusOK, nil)
}

func CreateRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := rds_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateRegistrationFields(w, r)
	}
}


// GetRegistrationFieldsHandler is a wrapper that creates the UserHandler and returns the handler function for getting a registrationFields by ID
func GetRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := rds_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetRegistrationFields(w, r)
	}
}

// UpdateRegistrationFieldsHandler is a wrapper that creates the UserHandler and returns the handler function for updating a registrationFields
func UpdateRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := rds_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateRegistrationFields(w, r)
	}
}

// DeleteRegistrationFieldsHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a registrationFields
func DeleteRegistrationFieldsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	registrationFieldsService := rds_service.NewRegistrationFieldsService()
	handler := NewRegistrationFieldsHandler(registrationFieldsService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteRegistrationFields(w, r)
	}
}
