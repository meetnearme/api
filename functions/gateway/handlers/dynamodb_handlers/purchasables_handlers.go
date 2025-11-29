package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var validate *validator.Validate = validator.New()

type PurchasableHandler struct {
	PurchasableService internal_types.PurchasableServiceInterface
}

func NewPurchasableHandler(purchasableService internal_types.PurchasableServiceInterface) *PurchasableHandler {
	return &PurchasableHandler{PurchasableService: purchasableService}
}

func (h *PurchasableHandler) CreatePurchasable(w http.ResponseWriter, r *http.Request) {
	// TODO: validate that all purchasables in the payload array
	// have the same currency
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
	now := time.Now()
	createPurchasable.CreatedAt = now
	createPurchasable.UpdatedAt = now

	db := transport.GetDB()
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
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing purchasable eventId"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	purchasables, err := h.PurchasableService.GetPurchasablesByEventID(r.Context(), db, eventId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get purchasable: "+err.Error()), http.StatusNotFound, err)
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
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing purchasable event_id"), http.StatusBadRequest, nil)
		return
	}

	var updatePurchasable internal_types.PurchasableUpdate
	updatePurchasable.EventId = eventId
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

	db := transport.GetDB()
	purchasables, err := h.PurchasableService.UpdatePurchasable(r.Context(), db, updatePurchasable)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update purchasable: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(purchasables)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchasableHandler) DeletePurchasable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing purchasable event_id"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	err := h.PurchasableService.DeletePurchasable(r.Context(), db, eventId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete purchasable: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Purchasable successfully deleted"), http.StatusOK, nil)
}

func CreatePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreatePurchasable(w, r)
	}
}

// GetPurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for getting a purchasable by ID
func GetPurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasable(w, r)
	}
}

// GetPurchasablesHandler is a wrapper that creates the UserHandler and returns the handler function for getting all purchasables
func GetPurchasablesHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasable(w, r)
	}
}

// UpdatePurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for updating a purchasable
func UpdatePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdatePurchasable(w, r)
	}
}

// DeletePurchasableHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a purchasable
func DeletePurchasableHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	handler := NewPurchasableHandler(purchasableService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeletePurchasable(w, r)
	}
}
