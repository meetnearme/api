package rds_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type PurchasableHandler struct {
	PurchasableService internal_types.PurchasableServiceInterface
}

func NewPurchasableHandler(purchasableService internal_types.PurchasableServiceInterface) *PurchasableHandler {
	return &PurchasableHandler{PurchasableService: purchasableService}
}


func (h *PurchasableHandler) CreatePurchasable(w http.ResponseWriter, r *http.Request) {
	var createPurchasable internal_types.PurchasableInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createPurchasable)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
	}

	err = validate.Struct(&createPurchasable)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
	}

	// TODO: check if these are redundant in all Insert functions because of NOW() from sql
    now := time.Now().UTC().Format(time.RFC3339)
    createPurchasable.CreatedAt = now
    createPurchasable.UpdatedAt = now

	// Parse timestamps
	createdAtTime, err := time.Parse(time.RFC3339, createPurchasable.CreatedAt)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid created_at timestamp: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedAtTime := createdAtTime // Default to the same value if not provided
	if createPurchasable.UpdatedAt != "" {
		updatedAtTime, err = time.Parse(time.RFC3339, createPurchasable.UpdatedAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid updated_at timestamp: "+err.Error()), http.StatusBadRequest, err)
			return
		}
	}

	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Format timestamps for RDS
	createPurchasable.CreatedAt = createdAtTime.Format(rdsTimeFormat)
	createPurchasable.UpdatedAt = updatedAtTime.Format(rdsTimeFormat)

    db := transport.GetRdsDB()
    res, err := h.PurchasableService.InsertPurchasable(r.Context(), db, createPurchasable)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create purchasable: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *PurchasableHandler) GetPurchasable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing purchasable ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.PurchasableService.GetPurchasableByID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("Purchasable not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchasableHandler) GetPurchasablesByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["user_id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user_id ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    purchasables, err := h.PurchasableService.GetPurchasablesByUserID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user's purchasables: "+err.Error()), http.StatusNotFound, err)
        return
    }

    response, err := json.Marshal(purchasables)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchasableHandler) UpdatePurchasable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing purchasable ID"), http.StatusBadRequest, nil)
        return
    }

    var updatePurchasable internal_types.PurchasableUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &updatePurchasable)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    err = validate.Struct(&updatePurchasable)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.PurchasableService.UpdatePurchasable(r.Context(), db, id, updatePurchasable)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update purchasable: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("Purchasable not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchasableHandler) DeletePurchasable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing purchasable ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    err := h.PurchasableService.DeletePurchasable(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete purchasable: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("Purchasable successfully deleted"), http.StatusOK, nil)
}

func CreatePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := rds_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreatePurchasable(w, r)
	}
}


// GetPurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for getting a purchasable by ID
func GetPurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := rds_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasable(w, r)
	}
}

// GetPurchasablesHandler is a wrapper that creates the UserHandler and returns the handler function for getting all purchasables
func GetPurchasablesHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := rds_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasablesByUserID(w, r)
	}
}

// UpdatePurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for updating a purchasable
func UpdatePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := rds_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdatePurchasable(w, r)
	}
}

// DeletePurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a purchasable
func DeletePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := rds_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeletePurchasable(w, r)
	}
}
