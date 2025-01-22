package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type CompetitionConfigHandler struct {
	CompetitionConfigService internal_types.CompetitionConfigServiceInterface
}

func NewCompetitionConfigHandler(eventCompetitionConfigService internal_types.CompetitionConfigServiceInterface) *CompetitionConfigHandler {
	return &CompetitionConfigHandler{CompetitionConfigService: eventCompetitionConfigService}
}

func (h *CompetitionConfigHandler) CreateCompetitionConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primaryOwner ID"), http.StatusBadRequest, nil)
		return
	}

	var createCompetitionConfig internal_types.CompetitionConfigInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	id := uuid.NewString()

	now := time.Now().Unix()
	createCompetitionConfig.CreatedAt = now
	createCompetitionConfig.UpdatedAt = now
	createCompetitionConfig.Id = id
	createCompetitionConfig.PrimaryOwnerId = primaryOwner

	err = validate.Struct(&createCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	res, err := h.CompetitionConfigService.InsertCompetitionConfig(r.Context(), db, createCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusCreated, nil)
}

// Get all configs that a primaryOwner has
func (h *CompetitionConfigHandler) GetCompetitionConfigsByPk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primaryOwner ID"), http.StatusBadRequest, nil)
		return
	}
	id := vars["id"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing id"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	eventCompetitionConfig, err := h.CompetitionConfigService.GetCompetitionConfigByPk(r.Context(), db, primaryOwner, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionConfig == nil {
		transport.SendServerRes(w, []byte("CompetitionConfig not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionConfigHandler) UpdateCompetitionConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primaryOwner ID"), http.StatusBadRequest, nil)
		return
	}
	id := vars["id"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing id"), http.StatusBadRequest, nil)
		return
	}

	var updateCompetitionConfig internal_types.CompetitionConfigUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &updateCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&updateCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	user, err := h.CompetitionConfigService.UpdateCompetitionConfig(r.Context(), db, primaryOwner, id, updateCompetitionConfig)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		transport.SendServerRes(w, []byte("CompetitionConfig not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(user)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionConfigHandler) DeleteCompetitionConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primaryOwner ID"), http.StatusBadRequest, nil)
		return
	}
	id := vars["id"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing id"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	err := h.CompetitionConfigService.DeleteCompetitionConfig(r.Context(), db, primaryOwner, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionConfig successfully deleted"), http.StatusOK, nil)
}

func CreateCompetitionConfigHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateCompetitionConfig(w, r)
	}
}

// GetCompetitionConfigHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventCompetitionConfig by ID
func GetCompetitionConfigByPkHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionConfigsByPk(w, r)
	}
}

// UpdateCompetitionConfigHandler is a wrapper that creates the UserHandler and returns the handler function for updating a eventCompetitionConfig
func UpdateCompetitionConfigHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateCompetitionConfig(w, r)
	}
}

// DeleteCompetitionConfigHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a eventCompetitionConfig
func DeleteCompetitionConfigHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteCompetitionConfig(w, r)
	}
}
