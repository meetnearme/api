package rds_service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type TransactionService struct{}

func NewTransactionService() internal_types.TransactionServiceInterface {
	return &TransactionService{}
}

func (s *TransactionService) InsertTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, transaction internal_types.TransactionInsert) (*internal_types.Transaction, error) {
    // Generate a new UUID if not provided
    if transaction.ID == "" {
        transaction.ID = uuid.New().String()
    }

    // Validate the transaction object
    if err := validate.Struct(transaction); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Construct the SQL query
    query := `
        INSERT INTO transactions (
            id, user_id, amount, currency, transaction_type,
            status, description, created_at, updated_at
        )
        VALUES (
            :id, :user_id, :amount, :currency, :transaction_type,
			:status, :description, NOW(), NOW()
        )
        RETURNING id, user_id, amount, currency, transaction_type,
            status, description, created_at, updated_at
    `
	log.Printf("here is the desc in service: %v", transaction.Description)

    // Prepare the parameters for the query
    params := map[string]interface{}{
        "id":                  transaction.ID,
        "user_id":                  transaction.UserID,
        "amount":                  transaction.Amount,
        "currency":                  transaction.Currency,
		"transaction_type":		transaction.TransactionType,
		"status":		transaction.Status,
		"description":		transaction.Description,
    }


	paramsRdsFormat, err := buildSqlTransactionParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the statement
    result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to insert transaction: %w", err)
    }

    // Extract the inserted transaction data
    insertedTransaction, err := extractAndMapSingleTransactionFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map transaction after insert: %w", err)
    }

    // return transaction, nil
	return insertedTransaction, nil
}


func (s *TransactionService) GetTransactionByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Transaction, error) {
	query := "SELECT * FROM transactions WHERE id = :id"
	params := map[string]interface{}{
		"id": id,
	}

	var paramsRdsFormat []rds_types.SqlParameter

	// ID (UUID)
	idValue, ok := params["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is not a valid string")
	}
	idRds := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	paramsRdsFormat = append(paramsRdsFormat, idRds)

	result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	log.Printf("Result in getby id: %v", result)
	log.Printf("Result formatted result: %v", result.FormattedRecords)

    // Extract the inserted transaction data
    transaction, err := extractAndMapSingleTransactionFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map transaction after insert: %w", err)
    }

    // return transaction, nil
	return transaction, nil
}

func (s *TransactionService) GetTransactionsByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.Transaction, error) {
    // Updated query to filter transactions by user_id and limit to 10 records
	query := `
		SELECT id, user_id, transaction_type, amount, currency, status, created_at, updated_at
		FROM transactions
		WHERE user_id = :user_id
		LIMIT 10
	`

	// UserID (UUID)
	params := []rds_types.SqlParameter{
		{
			Name:  aws.String("user_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: userID,
			},
		},
	}


    // Execute the SQL query
    result, err := rdsClient.ExecStatement(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("failed to get transactions: %w", err)
    }

    var transactions []internal_types.Transaction

    // Check if formattedRecords is available
    if result.FormattedRecords != nil {
        transactions, err = extractTransactionsFromJson(*result.FormattedRecords)
        if err != nil {
            return nil, fmt.Errorf("error extracting transactions from JSON: %w", err)
        }
    } else {
        return nil, fmt.Errorf("no formatted records found")
    }

    return transactions, nil
}

func (s *TransactionService) UpdateTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, transaction internal_types.TransactionUpdate) (*internal_types.Transaction, error) {
    // Build the SQL query to update transaction information
	query := `
		UPDATE transactions
		SET
			transaction_type = :transaction_type,
			amount = :amount,
			currency = :currency,
			status = :status,
			updated_at = now()
		WHERE id = :id
		RETURNING id, user_id, transaction_type, amount, currency, status, created_at, updated_at`

	params := map[string]interface{}{
		"id":              id,
		"transaction_type": transaction.TransactionType,
		"amount":          transaction.Amount,
		"currency":        transaction.Currency,
		"status":          transaction.Status,
	}

    // Convert parameters to RDS types
	rdsParams, err := buildSqlTransactionParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the SQL statement
    result, err := rdsClient.ExecStatement(ctx, query, rdsParams)
    if err != nil {
        return nil, fmt.Errorf("failed to update transaction: %w", err)
    }

    // Extract the updated transaction from the formattedRecords JSON
    updatedTransaction, err := extractAndMapSingleTransactionFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to extract updated transaction: %w", err)
    }

    // Since we are expecting one transaction in the result
    return updatedTransaction, nil
}

func (s *TransactionService) DeleteTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
 	query := "DELETE FROM transactions WHERE id = :id"
	params := map[string]interface{}{
		"id": id,
	}

	var paramsRdsFormat []rds_types.SqlParameter

	// ID (UUID)
	idValue, ok := params["id"].(string)
	if !ok {
		return fmt.Errorf("id is not a valid string")
	}
	idRds := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	paramsRdsFormat = append(paramsRdsFormat, idRds)

	result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
	log.Printf("Err from exec delete: %v", err)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	// Check if any rows were affected by the delete operation
	if result.NumberOfRecordsUpdated == 0 {
		// No rows were affected, meaning the transaction was not found
		return fmt.Errorf("transaction not found")
	}

	// No need to return transaction info; just return nil to indicate success.
	return nil
}

type MockTransactionService struct {
	InsertTransactionFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, transaction internal_types.TransactionInsert) (*internal_types.Transaction, error)
	GetTransactionByIDFunc func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Transaction, error)
	UpdateTransactionFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, transaction internal_types.TransactionUpdate) (*internal_types.Transaction, error)
	DeleteTransactionFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error
	GetTransactionsFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.Transaction, error) // New function
}

func (m *MockTransactionService) InsertTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, transaction internal_types.TransactionInsert) (*internal_types.Transaction, error) {
	return m.InsertTransactionFunc(ctx, rdsClient, transaction)
}

func (m *MockTransactionService) GetTransactionByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Transaction, error) {
	return m.GetTransactionByIDFunc(ctx, rdsClient, id)
}

func (m *MockTransactionService) UpdateTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, transaction internal_types.TransactionUpdate) (*internal_types.Transaction, error) {
	return m.UpdateTransactionFunc(ctx, rdsClient, id, transaction)
}

func (m *MockTransactionService) DeleteTransaction(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
	return m.DeleteTransactionFunc(ctx, rdsClient, id)
}

func (m *MockTransactionService) GetTransactionsByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI) ([]internal_types.Transaction, error) {
	return m.GetTransactionsFunc(ctx, rdsClient)
}

