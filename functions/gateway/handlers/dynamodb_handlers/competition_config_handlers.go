package dynamodb_handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
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

	if updateCompetitionConfigPayload.Id == "" {
		updateCompetitionConfigPayload.Id = uuid.NewString()
	}

	now := time.Now().Unix()
	updateCompetitionConfigPayload.UpdatedAt = now
	updateCompetitionConfigPayload.PrimaryOwner = userInfo.Sub

	err = validate.Struct(&updateCompetitionConfigPayload)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	// Store teams data before removing it from the struct
	teamsData := updateCompetitionConfigPayload.Teams
	roundsData := updateCompetitionConfigPayload.Rounds

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

	competitionConfigRes, err := h.CompetitionConfigService.UpdateCompetitionConfig(r.Context(), db, configUpdate.Id, configUpdate)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	updatedRounds := make([]internal_types.CompetitionRoundUpdate, len(roundsData))
	for i := range roundsData {
		updatedRounds[i] = roundsData[i]
		updatedRounds[i].CompetitionId = competitionConfigRes.Id
	}
	roundsData = updatedRounds

	if len(teamsData) > 0 {

		findTeamMembers := func(teamId string) []string {
			for _, team := range teamsData {
				if team.Id == teamId {
					competitors := make([]string, len(team.Competitors))
					for i, comp := range team.Competitors {
						competitors[i] = comp.UserId
					}
					return competitors
				}
			}
			return []string{}
		}

		candidateUsers := []string{}
		for _, team := range teamsData {
			candidateUsers = append(candidateUsers, team.Id)
		}
		existingUsers, err := helpers.SearchUsersByIDs(candidateUsers, false)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to search existing users: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		existingUserIds := make(map[string]bool)
		for _, user := range existingUsers {
			existingUserIds[user.UserID] = true
		}

		filteredUsersToCreate := make([]map[string]interface{}, 0)
		for _, team := range teamsData {
			if !existingUserIds[team.Id] {
				filteredUsersToCreate = append(filteredUsersToCreate, map[string]interface{}{
					"id":          team.Id,
					"displayName": team.DisplayName,
					"members":     strings.Join(findTeamMembers(team.Id), ","),
				})
			}
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(filteredUsersToCreate))
		var users []types.UserSearchResultDangerous
		for _, user := range filteredUsersToCreate {
			wg.Add(1)
			go func(userData map[string]interface{}) {
				defer wg.Done()
				members := strings.Split(userData["members"].(string), ",")
				user, err := helpers.CreateTeamUserWithMembers(
					userData["displayName"].(string),
					userData["id"].(string),
					members,
				)
				if err != nil {
					errChan <- fmt.Errorf("failed to create team user %s: %w", userData["id"].(string), err)
				}
				users = append(users, user)
			}(user)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(errChan)

		// Check for any errors
		for err := range errChan {
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to create team users: "+err.Error()), http.StatusInternalServerError, err)
				return
			}
		}

	}

	roundsData, err = helpers.NormalizeCompetitionRounds(roundsData)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to normalize competition rounds: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	service := dynamodb_service.NewCompetitionRoundService()
	_, err = service.PutCompetitionRounds(ctx, db, &roundsData)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to save competition rounds: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	rounds := make([]types.CompetitionRound, len(roundsData))
	for i, r := range roundsData {
		rounds[i] = types.CompetitionRound(r)
	}
	competitionConfigRes.Rounds = rounds

	response, err := json.Marshal(competitionConfigRes)
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
