package dynamodb_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
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

	err = json.Unmarshal(body, &updateCompetitionConfigPayload)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	isNew := false
	if updateCompetitionConfigPayload.Id == "" {
		updateCompetitionConfigPayload.Id = uuid.NewString()
		isNew = true
	}

	now := time.Now().Unix()
	updateCompetitionConfigPayload.UpdatedAt = now

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

	competitionConfigRes, err := h.CompetitionConfigService.UpdateCompetitionConfig(r.Context(), db, configUpdate.Id, configUpdate, isNew)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	authorizedOwners := []string{userInfo.Sub}
	authorizedOwners = append(authorizedOwners, competitionConfigRes.AuxilaryOwners...)
	isAuthorized := false
	for _, owner := range authorizedOwners {
		if owner == userInfo.Sub {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized {
		transport.SendServerRes(w, []byte("You are not authorized to update this competition"), http.StatusUnauthorized, nil)
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
		filteredUsersToUpdate := make([]map[string]interface{}, 0)
		for _, team := range teamsData {
			if !existingUserIds[team.Id] && team.ShouldCreate {
				filteredUsersToCreate = append(filteredUsersToCreate, map[string]interface{}{
					"id":          team.Id,
					"displayName": team.DisplayName,
					"members":     strings.Join(findTeamMembers(team.Id), ","),
				})
			} else if existingUserIds[team.Id] && team.ShouldUpdate {
				filteredUsersToUpdate = append(filteredUsersToUpdate, map[string]interface{}{
					"id":          team.Id,
					"displayName": team.DisplayName,
					"members":     strings.Join(findTeamMembers(team.Id), ","),
				})
			}
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(filteredUsersToCreate)+len(filteredUsersToUpdate))
		var users []types.UserSearchResultDangerous
		for _, user := range filteredUsersToCreate {
			wg.Add(1)
			go func(userData map[string]interface{}) {
				defer wg.Done()
				var members []string
				if membersStr, ok := userData["members"].(string); ok && membersStr != "" {
					members = strings.Split(membersStr, ",")
				}
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

		for _, user := range filteredUsersToUpdate {
			wg.Add(1)
			go func(userData map[string]interface{}) {
				defer wg.Done()
				err := helpers.UpdateUserMetadataKey(userData["id"].(string),
					"members",
					userData["members"].(string),
				)
				if err != nil {
					errChan <- fmt.Errorf("failed to update team user %s: %w", userData["id"].(string), err)
				}
				users = append(users, types.UserSearchResultDangerous{
					UserID:      userData["id"].(string),
					DisplayName: userData["displayName"].(string),
				})
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

	// Initialize empty rounds slice for the response
	competitionConfigRes.Rounds = []types.CompetitionRound{}

	// Only process rounds if there are any
	// Only process rounds if there are any
	if len(roundsData) > 0 {
		normalizedRounds, err := helpers.NormalizeCompetitionRounds(roundsData)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to normalize competition rounds: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		service := dynamodb_service.NewCompetitionRoundService()

		// Define base update keys (excluding eventId)
		baseKeysToUpdate := []string{
			"roundName",
			"competitorA",
			"competitorAScore",
			"competitorB",
			"competitorBScore",
			"matchup",
			"status",
			"isPending",
			"isVotingOpen",
			"description",
		}

		// Group rounds by whether they need eventId update
		var unassignedEventRounds, assignedEventRounds []internal_types.CompetitionRoundUpdate
		for _, round := range normalizedRounds {
			if round.EventId == helpers.COMP_UNASSIGNED_ROUND_EVENT_ID {
				unassignedEventRounds = append(unassignedEventRounds, round)
			} else {
				assignedEventRounds = append(assignedEventRounds, round)
			}
		}

		// Handle rounds with unassigned events (include eventId in updates)
		if len(unassignedEventRounds) > 0 {
			keysWithEventId := append(baseKeysToUpdate, "eventId")
			err = service.BatchPatchCompetitionRounds(ctx, db, unassignedEventRounds, keysWithEventId)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to update unassigned rounds: "+err.Error()), http.StatusInternalServerError, err)
				return
			}
		}

		// Handle rounds with assigned events (exclude eventId from updates)
		if len(assignedEventRounds) > 0 {
			err = service.BatchPatchCompetitionRounds(ctx, db, assignedEventRounds, baseKeysToUpdate)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to update assigned rounds: "+err.Error()), http.StatusInternalServerError, err)
				return
			}
		}

		// Combine all rounds for response
		allRounds := append(unassignedEventRounds, assignedEventRounds...)
		rounds := make([]types.CompetitionRound, len(allRounds))
		for i, r := range allRounds {
			rounds[i] = types.CompetitionRound(r)
		}
		competitionConfigRes.Rounds = rounds
	}

	// because we store rounds data in dynamo, there is no relational tables / foreign keys
	// this means that we don't know if there are existing rounds for this competition that
	// need to be deleted. Here, we make an API call to get all existing rounds and delete
	// any that are beyond the last index of `roundsData`
	defer func() {
		// delete all rounds of index position greater than or equal to the lenght of `roundsData`
		service := dynamodb_service.NewCompetitionRoundService()
		rounds, err := service.GetCompetitionRounds(r.Context(), db, competitionConfigRes.Id)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get competition rounds: "+err.Error()), http.StatusInternalServerError, err)
			return
		}
		for i, round := range *rounds {
			if i >= len(roundsData) {
				log.Printf("Deleting round, competitionId: %s, roundNumber: %s", round.CompetitionId, strconv.Itoa(int(round.RoundNumber)))
				service.DeleteCompetitionRound(r.Context(), db, round.CompetitionId, strconv.Itoa(int(round.RoundNumber)))
			}
		}
	}()

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
func (h *CompetitionConfigHandler) GetCompetitionConfigsByPrimaryOwner(w http.ResponseWriter, r *http.Request, isHtml bool) {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	// implicitly we fetch from the `primaryOwner` index if `ownerId` is not provided, if
	// provided, such as for community pages showing a DIFFERENT owner's configs,
	// we fetch from the `endTime` index
	ownerId := mux.Vars(r)[helpers.USER_ID_KEY]
	isSelf := true

	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
		if ownerId == "" {
			ownerId = userInfo.Sub
			isSelf = false
		}
	}

	if ownerId == "" && userInfo.Sub == "" {
		transport.SendServerRes(w, []byte("Must either be authenticated or provide an ownerId"), http.StatusUnauthorized, nil)
		return
	}

	db := transport.GetDB()
	configs, err := h.CompetitionConfigService.GetCompetitionConfigsByPrimaryOwner(ctx, db, ownerId, isSelf)
	if err != nil {
		log.Printf("Handler ERROR: Failed to get configs: %v", err)
		transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if configs == nil {
		configs = &[]internal_types.CompetitionConfig{}
	}

	if isHtml {
		var buf bytes.Buffer
		competitionConfigListPartial := partials.CompetitionConfigAdminList(configs)

		err = competitionConfigListPartial.Render(r.Context(), &buf)
		if err != nil {
			transport.SendHtmlErrorPartial([]byte("Failed to render competition config admin list: "+err.Error()), http.StatusInternalServerError)
			return
		}

		// TODO: this is painfully inconsistent, our `page_handlers.go` is returning
		// this as a `http.HandlerFunc` but here we are calling the function directly
		// rather than calling `return transport.SendHtmlRes` because the data handler
		// needs to simply write to the response buffer rather than returning an
		// `http.HandlerFunc`
		transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)(w, r)
	} else {
		response, err := json.Marshal(configs)
		if err != nil {
			transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
			return
		}
		transport.SendServerRes(w, response, http.StatusOK, nil)
	}

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
		handler.GetCompetitionConfigsByPrimaryOwner(w, r, false)
	}
}

func GetCompetitionConfigsHtmlByPrimaryOwnerHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionConfigService := dynamodb_service.NewCompetitionConfigService()
	handler := NewCompetitionConfigHandler(eventCompetitionConfigService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionConfigsByPrimaryOwner(w, r, true)
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
