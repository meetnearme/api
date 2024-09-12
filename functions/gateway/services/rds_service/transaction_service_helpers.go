package rds_service

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func buildSqlTransactionParams(parameters map[string]interface{}) ([]rds_types.SqlParameter, error) {
	var params []rds_types.SqlParameter

	log.Printf("parameters transactions: %v", parameters["description"])
	// ID (UUID)
	idValue, ok := parameters["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is not a valid string")
	}
	id := rds_types.SqlParameter{
		Name:     aws.String("id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: idValue,
		},
	}
	params = append(params, id)

	// UserID (UUID)
	userIdValue, ok := parameters["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("userId is not a valid string")
	}
	userID := rds_types.SqlParameter{
		Name: aws.String("user_id"),
		TypeHint: "UUID",
		Value: &rds_types.FieldMemberStringValue{
			Value: userIdValue,
		},
	}
	params = append(params, userID)

	// Amount
	amountValue, ok := parameters["amount"].(string)
	if !ok {
		return nil, fmt.Errorf("amount is not a valid string")
	}
	// Convert the string to float64
	amountFloat, err := strconv.ParseFloat(amountValue, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount to float64: %w", err)
	}

	amount := rds_types.SqlParameter{
		Name: aws.String("amount"),
		Value: &rds_types.FieldMemberDoubleValue{
			Value: amountFloat,
		},
	}
	params = append(params, amount)

	// Currency
	currencyValue, ok := parameters["currency"].(string)
	if !ok {
		return nil, fmt.Errorf("currency is not a valid string")
	}
	currency := rds_types.SqlParameter{
		Name: aws.String("currency"),
		Value: &rds_types.FieldMemberStringValue{
			Value: currencyValue,
		},
	}
	params = append(params, currency)

	// TransactionType
	transactionTypeValue, ok := parameters["transaction_type"].(string)
	if !ok {
		return nil, fmt.Errorf("transaction type is not a valid string")
	}
	transactionType := rds_types.SqlParameter{
		Name: aws.String("transaction_type"),
		Value: &rds_types.FieldMemberStringValue{
			Value: transactionTypeValue,
		},
	}
	params = append(params, transactionType)

	// Status
	statusValue, ok := parameters["status"].(string)
	if !ok {
		return nil, fmt.Errorf("status date is not a valid string")
	}
	status := rds_types.SqlParameter{
		Name: aws.String("status"),
		Value: &rds_types.FieldMemberStringValue{
			Value: statusValue,
		},
	}
	params = append(params, status)

	// Convert string timestamps to SQL parameters if they are provided
	if createdAtValue, ok := parameters["created_at"].(string); ok && createdAtValue != "" {
		createdAt := rds_types.SqlParameter{
			Name:     aws.String("created_at"),
			TypeHint: "TIMESTAMP",
			Value: &rds_types.FieldMemberStringValue{
				Value: createdAtValue,
			},
		}
		params = append(params, createdAt)
	}

	if updatedAtValue, ok := parameters["updated_at"].(string); ok && updatedAtValue != "" {
		updatedAt := rds_types.SqlParameter{
			Name:     aws.String("updated_at"),
			TypeHint: "TIMESTAMP",
			Value: &rds_types.FieldMemberStringValue{
				Value: updatedAtValue,
			},
		}
		params = append(params, updatedAt)
	}

	if descriptionValue, ok := parameters["description"].(string); ok && descriptionValue != "" {
		updatedAt := rds_types.SqlParameter{
			Name:     aws.String("description"),
			Value: &rds_types.FieldMemberStringValue{
				Value: descriptionValue,
			},
		}
		params = append(params, updatedAt)
	}


	return params, nil
}

func extractAndMapSingleTransactionFromJSON(formattedRecords string) (*internal_types.Transaction, error) {
	log.Printf("formatted records from JSON: %v", formattedRecords)
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON records: %v", err)
	}

	// Example assuming only one record for simplicity
	if len(records) == 0 {
		return nil, fmt.Errorf("no records found")
	}

	record := records[0]

	transaction := internal_types.Transaction{
		ID:              getString(record, "id"),
		UserID:          getString(record, "user_id"),
		Currency:        getString(record, "currency"),
		TransactionType: getString(record, "transaction_type"),
		Status:          getString(record, "status"),
		Description:     getString(record, "description"),
		CreatedAt:       getTime(record, "created_at"),
		UpdatedAt:       getTime(record, "updated_at"),
	}
	log.Printf("Transaction item from extractions: %v", transaction)

	return &transaction, nil
}

func extractTransactionsFromJson(formattedRecords string) ([]internal_types.Transaction, error) {
	var transactions []internal_types.Transaction

	// Parse formattedRecords as a JSON array
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling formatted records: %w", err)
	}

	// Iterate over each record and extract transaction information
	for _, record := range records {
		var transaction internal_types.Transaction

		// Map fields from record to transaction struct
		transaction.ID = getString(record, "id")
		transaction.UserID = getString(record, "user_id")
		transaction.Currency = getString(record, "currency")
		transaction.Amount = getFloat64(record, "amount")
		transaction.TransactionType = getString(record, "transactiontype")
		transaction.Status = getString(record, "status")
		transaction.Description = getString(record, "description")
		transaction.CreatedAt = getTime(record, "createdAt")
		transaction.UpdatedAt = getTime(record, "updatedAt")

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}


