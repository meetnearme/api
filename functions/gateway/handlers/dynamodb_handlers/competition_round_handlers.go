package dynamodb_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
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

func (h *CompetitionRoundHandler) PutCompetitionRounds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	log.Printf("Handler: Starting PutCompetitionRound for competitionId: %s", competitionId)

	var createCompetitionRounds []internal_types.CompetitionRoundUpdate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Handler ERROR: Failed to read request body: %v", err)
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	log.Printf("Handler: Request body received: %s", string(body))

	err = json.Unmarshal(body, &createCompetitionRounds)
	if err != nil {
		log.Printf("Handler ERROR: Failed to unmarshal JSON: %v", err)
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}
	log.Printf("ATTEN: %+v", createCompetitionRounds)

	log.Printf("Handler: Unmarshaled %d competition rounds", len(createCompetitionRounds))

	createCompetitionRounds, err = helpers.NormalizeCompetitionRounds(createCompetitionRounds)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to normalize competition rounds: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	db := transport.GetDB()
	log.Printf("Handler: Calling service layer PutCompetitionRounds with %d rounds", len(createCompetitionRounds))

	res, err := h.CompetitionRoundService.PutCompetitionRounds(r.Context(), db, &createCompetitionRounds)
	if err != nil {
		log.Printf("Handler ERROR: Service layer failed: %v", err)
		transport.SendServerRes(w, []byte("Failed to create eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	response, err := json.Marshal(res)
	if err != nil {
		log.Printf("Handler ERROR: Failed to marshal response: %v", err)
		transport.SendServerRes(w, []byte("Error marshaling JSON >> 70"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *CompetitionRoundHandler) GetAllCompetitionRounds(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	competitionId := vars["competitionId"]

	if competitionId == "" {
		log.Printf("Handler ERROR: Missing competitionId in request")
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()

	eventCompetitionRound, err := h.CompetitionRoundService.GetCompetitionRounds(r.Context(), db, competitionId)
	if err != nil {
		log.Printf("Handler ERROR: Service layer returned error: %v", err)
		transport.SendServerRes(w, []byte("Failed to get competition rounds: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	if eventCompetitionRound == nil {
		log.Printf("Handler: No rounds found for competitionId: '%s'", competitionId)
		transport.SendServerRes(w, []byte("CompetitionRound not found"), http.StatusNotFound, nil)
		return
	}

	response, err := json.Marshal(eventCompetitionRound)
	if err != nil {
		log.Printf("Handler ERROR: Failed to marshal response: %v", err)
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *CompetitionRoundHandler) GetCompetitionRoundsByEventId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventId := vars["eventId"]
	if eventId == "eventId" {
		transport.SendServerRes(w, []byte("Missing eventId"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	eventCompetitionRound, err := h.CompetitionRoundService.GetCompetitionRoundsByEventId(r.Context(), db, eventId)
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

func (h *CompetitionRoundHandler) GetCompetitionRoundByPrimaryKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["roundNumber"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing roundNumber"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	eventCompetitionRound, err := h.CompetitionRoundService.GetCompetitionRoundByPrimaryKey(r.Context(), db, competitionId, roundNumber)
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

func (h *CompetitionRoundHandler) DeleteCompetitionRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	competitionId := vars["competitionId"]
	if competitionId == "" {
		transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
		return
	}
	roundNumber := vars["roundNumber"]
	if roundNumber == "" {
		transport.SendServerRes(w, []byte("Missing round_number"), http.StatusBadRequest, nil)
		return
	}

	db := transport.GetDB()
	err := h.CompetitionRoundService.DeleteCompetitionRound(r.Context(), db, competitionId, roundNumber)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete eventCompetitionRound: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("CompetitionRound successfully deleted"), http.StatusOK, nil)
}

func GetCompetitionRoundsScoreSums(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		competitionId := vars["competitionId"]
		if competitionId == "" {
			transport.SendServerRes(w, []byte("Missing competition ID"), http.StatusBadRequest, nil)
			return
		}

		db := transport.GetDB()
		service := dynamodb_service.NewCompetitionRoundService()
		competitionRounds, err := service.GetCompetitionRounds(r.Context(), db, competitionId)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to retrieve competition rounds for score sum: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		var scoreSums = map[string]float64{}
		for _, round := range *competitionRounds {
			competitorA := round.CompetitorA
			competitorB := round.CompetitorB
			if _, ok := scoreSums[competitorA]; ok {
				scoreSums[competitorA] += round.CompetitorAScore
			} else {
				scoreSums[competitorA] = round.CompetitorAScore
			}

			if _, ok := scoreSums[competitorB]; ok {
				scoreSums[competitorB] += round.CompetitorBScore
			} else {
				scoreSums[competitorB] = round.CompetitorBScore
			}
		}

		jsonResponse, err := json.Marshal(scoreSums)
		if err != nil {
			transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
			return
		}

		transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
	}
}

func PutCompetitionRoundsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	log.Print("Hit round handler wrapper")
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PutCompetitionRounds(w, r)
	}
}

// GetCompetitionRoundHandler is a wrapper that creates the UserHandler and returns the handler function for getting a eventCompetitionRound by ID
func GetCompetitionRoundByPrimaryKeyHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionRoundByPrimaryKey(w, r)
	}
}

func GetCompetitionRoundsByEventIdHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetCompetitionRoundsByEventId(w, r)
	}
}

func GetAllCompetitionRoundsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	eventCompetitionRoundService := dynamodb_service.NewCompetitionRoundService()
	handler := NewCompetitionRoundHandler(eventCompetitionRoundService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetAllCompetitionRounds(w, r)
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
