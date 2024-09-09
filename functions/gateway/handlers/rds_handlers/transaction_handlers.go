package rds_handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type TransactionHandler struct {
	TransactionService internal_types.TransactionServiceInterface
}

func NewTransactionHandler(transactionService internal_types.TransactionServiceInterface) *TransactionHandler {
	return &TransactionHandler{TransactionService: transactionService}
}


func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var createTransaction internal_types.TransactionInsert
	body, err := io.ReadAll(r.Body)
	if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &createTransaction)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
	}

	err = validate.Struct(&createTransaction)
	if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
	}

	// Optional fields description and referenceId
	if createTransaction.Description == "" {
		createTransaction.Description = ""
	}

    now := time.Now().UTC().Format(time.RFC3339)
    createTransaction.CreatedAt = now
    createTransaction.UpdatedAt = now

	// Parse timestamps
	createdAtTime, err := time.Parse(time.RFC3339, createTransaction.CreatedAt)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid created_at timestamp: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	updatedAtTime := createdAtTime // Default to the same value if not provided
	if createTransaction.UpdatedAt != "" {
		updatedAtTime, err = time.Parse(time.RFC3339, createTransaction.UpdatedAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid updated_at timestamp: "+err.Error()), http.StatusBadRequest, err)
			return
		}
	}

	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format

	// Format timestamps for RDS
	createTransaction.CreatedAt = createdAtTime.Format(rdsTimeFormat)
	createTransaction.UpdatedAt = updatedAtTime.Format(rdsTimeFormat)

    db := transport.GetRdsDB()
    res, err := h.TransactionService.InsertTransaction(r.Context(), db, createTransaction)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to create transaction: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(res)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    log.Printf("Inserted new transaction: %+v", res)
    transport.SendServerRes(w, response, http.StatusCreated, nil)
}

func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing transaction ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.TransactionService.GetTransactionByID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("Transaction not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *TransactionHandler) GetTransactionsByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("vars get by user id: %v", vars)
	id := vars["user_id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing user_id ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    users, err := h.TransactionService.GetTransactionsByUserID(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get user's transactions: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    response, err := json.Marshal(users)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *TransactionHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing transaction ID"), http.StatusBadRequest, nil)
        return
    }

    var updateTransaction internal_types.TransactionUpdate
    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    err = json.Unmarshal(body, &updateTransaction)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
        return
    }

    err = validate.Struct(&updateTransaction)
    if err != nil {
        transport.SendServerRes(w, []byte("Invalid body: "+err.Error()), http.StatusBadRequest, err)
        return
    }

    db := transport.GetRdsDB()
    user, err := h.TransactionService.UpdateTransaction(r.Context(), db, id, updateTransaction)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to update transaction: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    if user == nil {
        transport.SendServerRes(w, []byte("Transaction not found"), http.StatusNotFound, nil)
        return
    }

    response, err := json.Marshal(user)
    if err != nil {
        transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, response, http.StatusOK, nil)
}

func (h *TransactionHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Printf("Vars in delete: %v", vars)
	id := vars["id"]
    if id == "" {
        transport.SendServerRes(w, []byte("Missing transaction ID"), http.StatusBadRequest, nil)
        return
    }

    db := transport.GetRdsDB()
    err := h.TransactionService.DeleteTransaction(r.Context(), db, id)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to delete transaction: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    transport.SendServerRes(w, []byte("Transaction successfully deleted"), http.StatusOK, nil)
}

func CreateTransactionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	transactionService := rds_service.NewTransactionService()
	handler := NewTransactionHandler(transactionService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.CreateTransaction(w, r)
	}
}


// GetTransactionHandler is a wrapper that creates the UserHandler and returns the handler function for getting a transaction by ID
func GetTransactionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	transactionService := rds_service.NewTransactionService()
	handler := NewTransactionHandler(transactionService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetTransaction(w, r)
	}
}

// GetTransactionsHandler is a wrapper that creates the UserHandler and returns the handler function for getting all transactions
func GetTransactionsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	transactionService := rds_service.NewTransactionService()
	handler := NewTransactionHandler(transactionService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetTransactionsByUserID(w, r)
	}
}

// UpdateTransactionHandler is a wrapper that creates the UserHandler and returns the handler function for updating a transaction
func UpdateTransactionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	transactionService := rds_service.NewTransactionService()
	handler := NewTransactionHandler(transactionService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateTransaction(w, r)
	}
}

// DeleteTransactionHandler is a wrapper that creates the UserHandler and returns the handler function for deleting a transaction
func DeleteTransactionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	transactionService := rds_service.NewTransactionService()
	handler := NewTransactionHandler(transactionService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.DeleteTransaction(w, r)
	}
}
