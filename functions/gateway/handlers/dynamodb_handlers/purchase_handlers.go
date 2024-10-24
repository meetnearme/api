package dynamodb_handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type PurchaseHandler struct {
	PurchaseService internal_types.PurchaseServiceInterface
}

func NewPurchaseHandler(eventPurchaseService internal_types.PurchaseServiceInterface) *PurchaseHandler {
	return &PurchaseHandler{PurchaseService: eventPurchaseService}
}

func (h *PurchaseHandler) CreatePurchase(w http.ResponseWriter, r *http.Request) {
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

	var createPurchase internal_types.PurchaseInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createPurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	now := time.Now().Unix()
	createPurchase.CreatedAt = now
	createPurchase.UpdatedAt = now
	createPurchase.CompositeKey = fmt.Sprintf("%s-%s", eventId, userId)
	createPurchase.EventID = eventId
	createPurchase.UserID = userId

	err = validate.Struct(&createPurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	res, err := h.PurchaseService.InsertPurchase(r.Context(), db, createPurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to create eventPurchase: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *PurchaseHandler) GetPurchaseByPk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing eventPurchase ID"), http.StatusBadRequest, nil)
		return
	}
	userId := vars["user_id"]
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing eventPurchase ID"), http.StatusBadRequest, nil)
		return
	}
	createdAt := vars["created_at"]
	if createdAt == "" {
		transport.SendServerRes(w, []byte("Missing eventPurchase createdAt timestamp"), http.StatusBadRequest, nil)
		return
	}

	createdAtInt, err := strconv.ParseInt(createdAt, 10, 64)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid createdAt timestamp"), http.StatusBadRequest, err)
		return
	}

	createdAtString := fmt.Sprintf("%020d", createdAtInt)

	db := transport.GetDB()
	eventPurchase, err := h.PurchaseService.GetPurchaseByPk(r.Context(), db, eventId, userId, createdAtString)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventPurchase == nil {
		transport.SendServerRes(w, []byte("Purchase not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventPurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchaseHandler) GetPurchasesByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["user_id"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing user_id ID"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	users, err := h.PurchaseService.GetPurchasesByUserID(r.Context(), db, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get user's eventPurchases: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(users)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchaseHandler) GetPurchasesByEventID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["event_id"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing event_id ID"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	events, err := h.PurchaseService.GetPurchasesByEventID(r.Context(), db, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get event's purchases: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(events)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchaseHandler) UpdatePurchase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["event_id"]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing eventPurchase ID"), http.StatusBadRequest, nil)
		return
	}
	userId := vars["user_id"]
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing eventPurchase ID"), http.StatusBadRequest, nil)
		return
	}

	var updatePurchase internal_types.PurchaseUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &updatePurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&updatePurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	user, err := h.PurchaseService.UpdatePurchase(r.Context(), db, eventId, userId, updatePurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update eventPurchase: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		transport.SendServerRes(w, []byte("Purchase not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(user)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchaseHandler) DeletePurchase(w http.ResponseWriter, r *http.Request) {
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

	db := transport.GetDB()
	err := h.PurchaseService.DeletePurchase(r.Context(), db, eventId, userId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventPurchase: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Purchase successfully deleted"), http.StatusOK, nil)
}

func CreatePurchaseHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreatePurchase(w, r)
	}
}

// GetPurchaseHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventPurchase by ID
func GetPurchaseByPkHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchaseByPk(w, r)
	}
}

// GetPurchasesHandler is a wrapper that creates the UserHandler and returns the handler function for getting all eventPurchases
func GetPurchasesByEventIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasesByEventID(w, r)
	}
}

func GetPurchasesByUserIDHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetPurchasesByUserID(w, r)
	}
}

// UpdatePurchaseHandler is a wrapper that creates the UserHandler and returns the handler function for updating a eventPurchase
func UpdatePurchaseHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdatePurchase(w, r)
	}
}

// DeletePurchaseHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a eventPurchase
func DeletePurchaseHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventPurchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchaseHandler(eventPurchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeletePurchase(w, r)
	}
}
