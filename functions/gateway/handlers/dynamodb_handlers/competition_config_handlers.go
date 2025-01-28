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

	// Store rounds data before removing it from the struct
	roundsData := updateCompetitionConfigPayload.Rounds
	teamsData := updateCompetitionConfigPayload.Teams
	log.Printf("70 >>> roundsData: %+v", roundsData)
	log.Printf("71 >>> teamsData: %+v", teamsData)
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

	if len(roundsData) > 0 {
		var competitionRoundsUpdate []internal_types.CompetitionRoundUpdate
		var usersToCreate []map[string]string

		// Helper function to find team display name
		findTeamDisplayName := func(teamId string) string {
			for _, team := range teamsData {
				if team.Id == teamId {
					return team.DisplayName
				}
			}
			return "" // Return empty string if team not found
		}

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

		for _, round := range roundsData {
			round.CompetitionId = competitionConfigRes.Id
			log.Printf("round: %+v", round)
			competitionRoundsUpdate = append(competitionRoundsUpdate, internal_types.CompetitionRoundUpdate(round))

			if strings.Contains(round.CompetitorA, helpers.COMP_TEAM_ID_PREFIX) {
				displayName := findTeamDisplayName(round.CompetitorA)
				usersToCreate = append(usersToCreate, map[string]string{
					"id":          round.CompetitorA,
					"displayName": displayName,
					"members":     strings.Join(findTeamMembers(round.CompetitorA), ","),
				})
			}
			if strings.Contains(round.CompetitorB, helpers.COMP_TEAM_ID_PREFIX) {
				displayName := findTeamDisplayName(round.CompetitorB)
				usersToCreate = append(usersToCreate, map[string]string{
					"id":          round.CompetitorB,
					"displayName": displayName,
					"members":     strings.Join(findTeamMembers(round.CompetitorB), ","),
				})
			}
		}

		log.Printf("usersToCreate: %+v", usersToCreate)

		if len(usersToCreate) > 0 {
			var wg sync.WaitGroup
			errChan := make(chan error, len(usersToCreate))
			var users []types.UserSearchResultDangerous
			for _, user := range usersToCreate {
				wg.Add(1)
				go func(userData map[string]string) {
					defer wg.Done()
					members := strings.Split(userData["members"], ",")
					user, err := helpers.CreateTeamUserWithMembers(
						userData["displayName"],
						userData["id"],
						members,
					)
					if err != nil {
						errChan <- fmt.Errorf("failed to create team user %s: %w", userData["id"], err)
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

		// TODO: this is only commented out to avoid creating MANY competition rounds
		competitionRoundsUpdate, err = helpers.NormalizeCompetitionRounds(competitionRoundsUpdate)
		if err != nil {
			// return &competitionConfigResponse, err
			transport.SendServerRes(w, []byte("Failed to normalize competition rounds: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		log.Printf("competitionRoundsUpdate: %+v", competitionRoundsUpdate)
		service := dynamodb_service.NewCompetitionRoundService()
		competitionRounds, err := service.PutCompetitionRounds(ctx, db, &competitionRoundsUpdate)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to save competition rounds: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		log.Printf("competitionRounds: %+v", competitionRounds)
	} else {
		// TODO: delete this log
		log.Printf("no rounds data")
	}
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
