package dynamodb_handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type CompetitionVoteHandler struct {
	CompetitionVoteService internal_types.CompetitionVoteServiceInterface
}

func NewCompetitionVoteHandler(eventCompetitionVoteService internal_types.CompetitionVoteServiceInterface) *CompetitionVoteHandler {
	return &CompetitionVoteHandler{CompetitionVoteService: eventCompetitionVoteService}
}

func (h *CompetitionVoteHandler) PutCompetitionVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["roundNumber"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing round number "), http.StatusBadRequest, nil)
		return
	}

	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	userId := userInfo.Sub

	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	var createCompetitionVote internal_types.CompetitionVoteUpdate
	err = json.Unmarshal(body, &createCompetitionVote)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	createCompetitionVote.ExpiresOn = time.Now().Add(24 * time.Hour).Unix()
	createCompetitionVote.CompositePartitionKey = fmt.Sprintf("%s_%s", competitionId, roundNumber)
	createCompetitionVote.UserId = userId

	err = validate.Struct(&createCompetitionVote)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	res, err := h.CompetitionVoteService.PutCompetitionVote(r.Context(), db, createCompetitionVote)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionVote: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusCreated, nil)
}

// Pk is <competitionId>_<roundNumber>
func (h *CompetitionVoteHandler) GetCompetitionVotesByCompetitionRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["roundNumber"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing round number "), http.StatusBadRequest, nil)
		return
	}

	compositePartitionKey := fmt.Sprintf("%s_%s", competitionId, roundNumber)

	db := transport.GetDB()
	eventCompetitionVote, err := h.CompetitionVoteService.GetCompetitionVotesByCompetitionRound(r.Context(), db, compositePartitionKey)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionVote == nil {
		transport.SendServerRes(w, []byte("CompetitionVote not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventCompetitionVote)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionVoteHandler) DeleteCompetitionVote(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Access the keys
	competitionId := data["competitionId"].(string)
	userId := data["userId"].(string)

	roundNumber := fmt.Sprintf("%d", int(data["roundNumber"].(float64)))

	// Validate keys
	if competitionId == "" || roundNumber == "" || userId == "" {
		http.Error(w, "Missing required keys in the payload", http.StatusBadRequest)
		return
	}

	compositePartitionKey := fmt.Sprintf("%s_%s", competitionId, roundNumber)

	db := transport.GetDB()
	err = h.CompetitionVoteService.DeleteCompetitionVote(r.Context(), db, compositePartitionKey, userId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionVote: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionVote successfully deleted"), http.StatusOK, nil)
}

func GetCompetitionVotesTallyForRoundHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		competitionId := vars["competitionId"]
		if competitionId == "" {
			transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
			return
		}
		roundNumber := vars["roundNumber"]
		if roundNumber == "" {
			transport.SendServerRes(w, []byte("Missing round number "), http.StatusBadRequest, nil)
			return
		}

		compositePartitionKey := fmt.Sprintf("%s_%s", competitionId, roundNumber)

		db := transport.GetDB()
		service := dynamodb_service.NewCompetitionVoteService()
		eventCompetitionVotes, err := service.GetCompetitionVotesByCompetitionRound(r.Context(), db, compositePartitionKey)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		if eventCompetitionVotes == nil {
			transport.SendServerRes(w, []byte("CompetitionVotes not found for competitionId and roundNumber"), http.StatusNotFound, nil)
			return
		}

		var voteTally = map[string]int64{}
		for _, vote := range eventCompetitionVotes {
			voteRecipientId := vote.VoteRecipientId
			// check if below throws error with hard coded value that does not exist
			if _, ok := voteTally[voteRecipientId]; ok {
				voteTally[voteRecipientId] += 1
			} else {
				voteTally[voteRecipientId] = 1
			}
		}

		response, err := json.Marshal(voteTally)
		if err != nil {
			transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
			return
		}

		transport.SendServerRes(w, response, http.StatusOK, nil)
	}
}

// PutCompetitionVoteHandler is a wrapper that creates the UserHandler and returns the handler function for updating a eventCompetitionVote
func PutCompetitionVoteHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionVoteService := dynamodb_service.NewCompetitionVoteService()
	handler := NewCompetitionVoteHandler(eventCompetitionVoteService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PutCompetitionVote(w, r)
	}
}

// // GetCompetitionVoteHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventCompetitionVote by ID
func GetCompetitionVotesByRoundHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionVoteService := dynamodb_service.NewCompetitionVoteService()
	handler := NewCompetitionVoteHandler(eventCompetitionVoteService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionVotesByCompetitionRound(w, r)
	}
}

func DeleteCompetitionVoteHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionVoteService := dynamodb_service.NewCompetitionVoteService()
	handler := NewCompetitionVoteHandler(eventCompetitionVoteService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteCompetitionVote(w, r)
	}
}
