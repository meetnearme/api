package dynamodb_handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
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
	ctx := r.Context()
	id := vars["user_id"]

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	userId := userInfo.Sub
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusUnauthorized, nil)
		return
	}

	if id != userId {
		transport.SendServerRes(w, []byte("You are not authorized to view this user's purchases"), http.StatusForbidden, nil)
		return
	}

	limit := r.URL.Query().Get("limit")
	limitInt, err := strconv.ParseInt(limit, 10, 32)
	if err != nil || limit == "" {
		limitInt = helpers.DEFAULT_PAGINATION_LIMIT
	}
	startKey := r.URL.Query().Get("start_key")
	if id == "" {
		transport.SendServerRes(w, []byte("Missing event_id ID"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	purchases, lastEvaluatedKey, err := h.PurchaseService.GetPurchasesByUserID(r.Context(), db, id, int32(limitInt), startKey)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get user's eventPurchases: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	responseData := struct {
		Count     int                                      `json:"count"`
		NextKey   map[string]dynamodb_types.AttributeValue `json:"nextKey"`
		Purchases []internal_types.Purchase                `json:"purchases"`
	}{
		Count:     len(purchases),
		NextKey:   lastEvaluatedKey,
		Purchases: purchases,
	}

	response, err := json.Marshal(responseData)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *PurchaseHandler) GetPurchasesByEventID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := r.Context()
	eventId := vars["event_id"]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	// Get user info from context
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	userId := userInfo.Sub
	if userId == "" {
		transport.SendServerRes(w, []byte("You must be logged in to view this event's purchases"), http.StatusUnauthorized, nil)
		return
	}

	roleClaims := []helpers.RoleClaim{}
	if _, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = ctx.Value("roleClaims").([]helpers.RoleClaim)
	}
	// Validate event ownership
	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get Marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}
	event, err := services.GetMarqoEventByID(marqoClient, eventId, "")
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	canEdit := helpers.CanEditEvent(event, &userInfo, roleClaims)

	if !canEdit {
		transport.SendServerRes(w, []byte("You are not authorized to view this event's purchases"), http.StatusForbidden, nil)
		return
	}

	// Handle pagination
	limit := r.URL.Query().Get("limit")
	limitInt, err := strconv.ParseInt(limit, 10, 32)
	if err != nil || limit == "" {
		limitInt = helpers.DEFAULT_PAGINATION_LIMIT
	}
	startKey := r.URL.Query().Get("start_key")

	db := transport.GetDB()
	purchases, lastEvaluatedKey, err := h.PurchaseService.GetPurchasesByEventID(r.Context(), db, eventId, int32(limitInt), startKey)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get event's purchases: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	responseData := struct {
		Count     int                                      `json:"count"`
		NextKey   map[string]dynamodb_types.AttributeValue `json:"nextKey"`
		Purchases []internal_types.Purchase                `json:"purchases"`
	}{
		Count:     len(purchases),
		NextKey:   lastEvaluatedKey,
		Purchases: purchases,
	}

	response, err := json.Marshal(responseData)
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

	createdAt := vars["created_at"]
	if createdAt == "" {
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
	user, err := h.PurchaseService.UpdatePurchase(r.Context(), db, eventId, userId, createdAt, updatePurchase)
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
