package dynamodb_handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type CompetitionRoundHandler struct {
	CompetitionRoundService internal_types.CompetitionRoundServiceInterface
}

func NewCompetitionRoundHandler(eventCompetitionRoundService internal_types.CompetitionRoundServiceInterface) *CompetitionRoundHandler {
	return &CompetitionRoundHandler{CompetitionRoundService: eventCompetitionRoundService}
}
func (h *CompetitionRoundHandler) CreateCompetitionRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("DEBUG: Received vars from request: %+v", vars)

	competitionId := vars["competition_id"]
	roundNumber := vars["round_number"]
	primaryOwner := vars["primary_owner"]

	roundNumberInt, err := strconv.ParseInt(roundNumber, 10, 64)
	if err != nil {
		log.Printf("ERROR: Invalid round number format: %v", err)
		transport.SendServerRes(w, []byte("Invalid round number format"), http.StatusBadRequest, err)
		return
	}

	log.Printf("DEBUG: Extracted path params - competitionId: %s, roundNumber: %s, primaryOwner: %s",
		competitionId, roundNumber, primaryOwner)

	var createCompetitionRound internal_types.CompetitionRoundInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read request body: %v", err)
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	log.Printf("DEBUG: Received request body: %s", string(body))
	createCompetitionRound.RoundNumber = roundNumberInt

	err = json.Unmarshal(body, &createCompetitionRound)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal request body: %v", err)
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	log.Printf("DEBUG: Unmarshaled request body: %+v", createCompetitionRound)

	createCompetitionRound.OwnerId = primaryOwner

	now := time.Now().Unix()
	createCompetitionRound.CreatedAt = now
	createCompetitionRound.UpdatedAt = now
	createCompetitionRound.PK = fmt.Sprintf("OWNER_%s", primaryOwner)
	createCompetitionRound.SK = fmt.Sprintf("COMPETITION_%s_ROUND_%s", competitionId, roundNumber)

	log.Printf("DEBUG: Final competition round object before validation: %+v", createCompetitionRound)

	err = validate.Struct(&createCompetitionRound)
	if err != nil {
		log.Printf("ERROR: Validation failed: %v", err)
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	db := transport.GetDB()
	log.Printf("DEBUG: Calling InsertCompetitionRound with PK: %s, SK: %s", createCompetitionRound.PK, createCompetitionRound.SK)

	res, err := h.CompetitionRoundService.InsertCompetitionRound(r.Context(), db, createCompetitionRound)
	if err != nil {
		log.Printf("ERROR: Failed to insert competition round: %v", err)
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	log.Printf("DEBUG: Successfully inserted competition round. Result: %+v", res)

	response, err := json.Marshal(res)
	if err != nil {
		log.Printf("ERROR: Failed to marshal response: %v", err)
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *CompetitionRoundHandler) GetCompetitionRounds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competition_id"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primary owner user ID"), http.StatusBadRequest, nil)
		return
	}

	partitionKey := fmt.Sprintf("OWNER_%s", primaryOwner)

	db := transport.GetDB()
	eventCompetitionRound, err := h.CompetitionRoundService.GetCompetitionRounds(r.Context(), db, partitionKey, competitionId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get competition rounds: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionRound == nil {
		transport.SendServerRes(w, []byte("CompetitionRound not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventCompetitionRound)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionRoundHandler) GetCompetitionRoundByPrimary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competition_id"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["round_number"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing round_number"), http.StatusBadRequest, nil)
		return
	}
	primaryOwner := vars["primary_owner"]
	if primaryOwner == "" {
		transport.SendServerRes(w, []byte("Missing primary owner user ID"), http.StatusBadRequest, nil)
		return
	}

	partitionKey := fmt.Sprintf("OWNER_%s", primaryOwner)

	db := transport.GetDB()
	eventCompetitionRound, err := h.CompetitionRoundService.GetCompetitionRoundByPk(r.Context(), db, partitionKey, competitionId, roundNumber)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get competition round: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionRound == nil {
		transport.SendServerRes(w, []byte("CompetitionRound not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventCompetitionRound)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionRoundHandler) UpdateCompetitionRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competition_id"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["round_number"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing round_number"), http.StatusBadRequest, nil)
		return
	}

	userId := vars["user_id"]
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
		return
	}

	var updateCompetitionRound internal_types.CompetitionRoundUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &updateCompetitionRound)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&updateCompetitionRound)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	partitionKey := fmt.Sprintf("OWNER_%s", userId)
	sortKey := fmt.Sprintf("COMPETITION_%s_ROUND_%s", competitionId)

	db := transport.GetDB()
	user, err := h.CompetitionRoundService.UpdateCompetitionRound(r.Context(), db, partitionKey, sortKey, updateCompetitionRound)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if user == nil {
		transport.SendServerRes(w, []byte("CompetitionRound not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(user)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionRoundHandler) DeleteCompetitionRound(w http.ResponseWriter, r *http.Request) {
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
	err := h.CompetitionRoundService.DeleteCompetitionRound(r.Context(), db, eventId, userId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionRound successfully deleted"), http.StatusOK, nil)
}

func CreateCompetitionRoundHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateCompetitionRound(w, r)
	}
}

// GetCompetitionRoundHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventCompetitionRound by ID
func GetCompetitionRoundByPrimaryKeyHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionRoundByPrimary(w, r)
	}
}

func GetCompetitionRoundsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionRounds(w, r)
	}
}

// UpdateCompetitionRoundHandler is a wrapper that creates the UserHandler and returns the handler function for updating a eventCompetitionRound
func UpdateCompetitionRoundHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateCompetitionRound(w, r)
	}
}

// DeleteCompetitionRoundHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a eventCompetitionRound
func DeleteCompetitionRoundHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteCompetitionRound(w, r)
	}
}
