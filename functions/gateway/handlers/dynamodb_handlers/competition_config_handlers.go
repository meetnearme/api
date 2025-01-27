package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
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

func (h *CompetitionConfigHandler) UpdateCompetitionConfig(w http.ResponseWriter, r *http.Request) {
	var updateCompetitionConfigPayload internal_types.CompetitionConfigUpdatePayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	err = json.Unmarshal(body, &updateCompetitionConfigPayload)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	var getCompetitionConfigResponse internal_types.CompetitionConfigResponse
	if updateCompetitionConfigPayload.Id == "" {
		updateCompetitionConfigPayload.Id = uuid.NewString()
	} else {
		db := transport.GetDB()

		service := dynamodb_service.NewCompetitionConfigService()
		getCompetitionConfigResponse, err = service.GetCompetitionConfigById(ctx, db, updateCompetitionConfigPayload.Id)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

	}

	now := time.Now().Unix()
	updateCompetitionConfigPayload.UpdatedAt = now
	updateCompetitionConfigPayload.PrimaryOwner = userInfo.Sub

	err = validate.Struct(&updateCompetitionConfigPayload)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	// Store rounds data before removing it from the struct
	roundsData := updateCompetitionConfigPayload.Rounds

	log.Printf("roundsData: %+v", roundsData)
	// Create target struct
	var configUpdate internal_types.CompetitionConfigUpdate

	// Use reflection to copy fields
	sourceVal := reflect.ValueOf(updateCompetitionConfigPayload)
	targetVal := reflect.ValueOf(&configUpdate).Elem()

	// copy the valid properties from the source, but omit invalid ones in the new struct
	for i := 0; i < targetVal.NumField(); i++ {
		fieldName := targetVal.Type().Field(i).Name
		if sourceField := sourceVal.FieldByName(fieldName); sourceField.IsValid() {
			targetVal.Field(i).Set(sourceField)
		}
	}

	res, err := h.CompetitionConfigService.UpdateCompetitionConfig(r.Context(), db, configUpdate.Id, configUpdate)
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

// Get config by ID
func (h *CompetitionConfigHandler) GetCompetitionConfigsById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["competitionId"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing competitionId"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	eventCompetitionConfig, err := h.CompetitionConfigService.GetCompetitionConfigById(r.Context(), db, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionConfig.Id == "" {
		transport.SendServerRes(w, []byte("CompetitionConfig not found"), http.StatusNotFound, nil)
		return
	}

	var CompetitionConfigResponse internal_types.CompetitionConfigResponse
	CompetitionConfigResponse.CompetitionConfig = eventCompetitionConfig.CompetitionConfig
	CompetitionConfigResponse.Owners = eventCompetitionConfig.Owners

	response, err := json.Marshal(CompetitionConfigResponse)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

// Get all configs that a primaryOwner has
func (h *CompetitionConfigHandler) GetCompetitionConfigsByPrimaryOwner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	log.Printf("Handler: Getting configs for user: %s", userInfo.Sub)
	if userInfo.Sub == "" {
		transport.SendServerRes(w, []byte("User not authenticated"), http.StatusUnauthorized, nil)
		return
	}

	db := transport.GetDB()
	configs, err := h.CompetitionConfigService.GetCompetitionConfigsByPrimaryOwner(ctx, db, userInfo.Sub)
	if err != nil {
		log.Printf("Handler ERROR: Failed to get configs: %v", err)
		transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if configs == nil {
		configs = &[]internal_types.CompetitionConfig{}
	}

	response, err := json.Marshal(configs)
	if err != nil {
		log.Printf("Handler ERROR: Failed to marshal response: %v", err)
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	log.Printf("Handler: Successfully retrieved %d configs", len(*configs))
	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionConfigHandler) DeleteCompetitionConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing id"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	err := h.CompetitionConfigService.DeleteCompetitionConfig(r.Context(), db, competitionId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionConfig successfully deleted"), http.StatusOK, nil)
}

func UpdateCompetitionConfigHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateCompetitionConfig(w, r)
	}
}

// GetCompetitionConfigHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventCompetitionConfig by ID
func GetCompetitionConfigByIdHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionConfigsById(w, r)
	}
}

func GetCompetitionConfigsByPrimaryOwnerHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionConfigsByPrimaryOwner(w, r)
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
