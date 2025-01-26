package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type CompetitionWaitingRoomParticipantHandler struct {
	CompetitionWaitingRoomService internal_types.CompetitionWaitingRoomParticipantServiceInterface
}

func NewCompetitionWaitingRoomParticipantHandler(competitionWaitingRoomService internal_types.CompetitionWaitingRoomParticipantServiceInterface) *CompetitionWaitingRoomParticipantHandler {
	return &CompetitionWaitingRoomParticipantHandler{CompetitionWaitingRoomService: competitionWaitingRoomService}
}

func (h *CompetitionWaitingRoomParticipantHandler) PutCompetitionWaitingRoomParticipant(w http.ResponseWriter, r *http.Request) {
	var competitionWaitingRoomParticipantUpdate internal_types.CompetitionWaitingRoomParticipantUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	vars := mux.Vars(r)
	competitionId := vars["competitionId"]

	// ctx := r.Context()
	// userInfo := helpers.UserInfo{}
	// if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
	// 	userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	// }

	err = json.Unmarshal(body, &competitionWaitingRoomParticipantUpdate)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	competitionWaitingRoomParticipantUpdate.ExpiresOn = time.Now().Add(24 * time.Hour).Unix()
	// competitionWaitingRoomParticipantUpdate.UserId = userInfo.Sub
	competitionWaitingRoomParticipantUpdate.UserId = "111111111111111111"
	competitionWaitingRoomParticipantUpdate.CompetitionId = competitionId

	err = validate.Struct(&competitionWaitingRoomParticipantUpdate)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	res, err := h.CompetitionWaitingRoomService.PutCompetitionWaitingRoomParticipant(r.Context(), db, competitionWaitingRoomParticipantUpdate)
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

func (h *CompetitionWaitingRoomParticipantHandler) GetCompetitionWaitingRoomParticipants(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["competitionId"]
	if id == "" {
		transport.SendServerRes(w, []byte("Missing competitionId"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	competitionWaitingRoomParticipants, err := h.CompetitionWaitingRoomService.GetCompetitionWaitingRoomParticipants(r.Context(), db, id)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get waiting room participants: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if competitionWaitingRoomParticipants == nil {
		transport.SendServerRes(w, []byte("Waiting room paricipants not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(competitionWaitingRoomParticipants)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionWaitingRoomParticipantHandler) DeleteCompetitionWaitingRoomParticipant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	userId := vars["userId"]
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing userId"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	err := h.CompetitionWaitingRoomService.DeleteCompetitionWaitingRoomParticipant(r.Context(), db, competitionId, userId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionRound successfully deleted"), http.StatusOK, nil)
}

func PutCompetitionWaitingRoomParticipantHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	competitionWaitingRoomService := dynamodb_service.NewCompetitionWaitingRoomParticipantService()
	handler := NewCompetitionWaitingRoomParticipantHandler(competitionWaitingRoomService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PutCompetitionWaitingRoomParticipant(w, r)
	}
}

// Get all waiting room participants for as competition
func GetCompetitionWaitingRoomParticipantsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	competitionWaitingRoomService := dynamodb_service.NewCompetitionWaitingRoomParticipantService()
	handler := NewCompetitionWaitingRoomParticipantHandler(competitionWaitingRoomService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionWaitingRoomParticipants(w, r)
	}
}

func DeleteCompetitionWaitingRoomParticipantHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	competitionWaitingRoomService := dynamodb_service.NewCompetitionWaitingRoomParticipantService()
	handler := NewCompetitionWaitingRoomParticipantHandler(competitionWaitingRoomService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteCompetitionWaitingRoomParticipant(w, r)
	}
}
