package types

import (
	"context"
	"time"
)

type TransactionInsert struct {
	ID string `json:"id"`
	UserID string `json:"userId" validate:"required"`
	Amount string `json:"amount" validate:"required"`
	Currency string `json:"currency" validate:"required"`
	TransactionType string `json:"transaction_type" validate:"required"`
	Status string `json:"status" validate:"required"`
	Description string `json:"description"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Transaction struct {
	ID string `json:"id"`
	UserID string `json:"user_id"`
	Currency string `json:"currency" `
	TransactionType string `json:"transaction_type" validate:"required"`
	Status string `json:"status" `
	Description string `json:"description"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TransactionUpdate struct {
	UserID string `json:"user_id"`
	Amount string `json:"amount"`
	Currency string `json:"currency" `
	TransactionType string `json:"transaction_type"`
	Status string `json:"status" `
	Description string `json:"description"`
}

type TransactionServiceInterface interface {
	InsertTransaction(ctx context.Context, rdsClient RDSDataAPI, transaction TransactionInsert) (*Transaction, error)
	GetTransactionByID(ctx context.Context, rdsClient RDSDataAPI, id string) (*Transaction, error)
	GetTransactionsByUserID(ctx context.Context, rdsClient RDSDataAPI, userID string) ([]Transaction, error)
	UpdateTransaction(ctx context.Context, rdsClient RDSDataAPI, id string, transaction TransactionUpdate) (*Transaction, error)
	DeleteTransaction(ctx context.Context, rdsClient RDSDataAPI, id string) error
}
