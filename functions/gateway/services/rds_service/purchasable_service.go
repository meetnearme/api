// TODO: change all fmt to log printout in new rds handlers and services
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

type PurchasableService struct{}

func NewPurchasableService() internal_types.PurchasableServiceInterface {
	return &PurchasableService{}
}

func (s *PurchasableService) InsertPurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, purchasable internal_types.PurchasableInsert) (*internal_types.Purchasable, error) {
    // Generate a new UUID if not provided
    if purchasable.ID == "" {
        purchasable.ID = uuid.New().String()
    }

    // Validate the purchasable object
    if err := validate.Struct(purchasable); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Construct the SQL query
    query := `
        INSERT INTO purchasables (
            id, user_id, name, item_type, cost, currency, inventory,
			charge_recurrence_interval, charge_recurrence_interval_count,
			charge_recurrence_end_date,
			donation_ratio,
			created_at, updated_at
        )
        VALUES (
			:id, :user_id, :name, :item_type, :cost, :currency, :inventory,
			:charge_recurrence_interval, :charge_recurrence_interval_count,
			:charge_recurrence_end_date,
			:donation_ratio,
			NOW(), NOW()
        )
        RETURNING
            id, user_id, name, item_type, cost, currency, inventory,
			charge_recurrence_interval, charge_recurrence_interval_count,
			charge_recurrence_end_date,
			donation_ratio,
			created_at, updated_at
    `

    // Prepare the parameters for the query
    params := map[string]interface{}{
        "id":                  purchasable.ID,
		"user_id": purchasable.UserID,
        "name":                  purchasable.Name,
        "item_type":                  purchasable.ItemType,
		"cost": purchasable.Cost,
		"inventory": purchasable.Inventory,
        "currency":                  purchasable.Currency,
		"charge_recurrence_interval":		purchasable.ChargeRecurrenceInterval,
		"charge_recurrence_interval_count":		purchasable.ChargeRecurrenceIntervalCount,
		"charge_recurrence_end_date":		purchasable.ChargeRecurrenceEndDate,
		"donation_ratio": purchasable.DonationRatio,
    }

	log.Printf("params in create purchasable: %v", params)

	paramsRdsFormat, err := buildSqlPurchasableParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the statement
    result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to insert purchasable: %w", err)
    }

    // Extract the inserted purchasable data
    insertedPurchasable, err := extractAndMapSinglePurchasableFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map purchasable after insert: %w", err)
    }

	return insertedPurchasable, nil
}


func (s *PurchasableService) GetPurchasableByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error) {
	query := "SELECT * FROM purchasables WHERE id = :id"
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
		return nil, fmt.Errorf("failed to get purchasable: %w", err)
	}

	log.Printf("Result in getby id: %v", result)
	log.Printf("Result formatted result: %v", result.FormattedRecords)

    // Extract the inserted purchasable data
    purchasable, err := extractAndMapSinglePurchasableFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map purchasable after insert: %w", err)
    }

    // return purchasable, nil
	return purchasable, nil
}

func (s *PurchasableService) GetPurchasablesByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.Purchasable, error) {
    // Updated query to filter purchasables by user_id and limit to 10 records
	query := `
		SELECT *
		FROM purchasables
		WHERE user_id = :user_id
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
        return nil, fmt.Errorf("failed to get purchasables: %w", err)
    }

    var purchasables []internal_types.Purchasable

    // Check if formattedRecords is available
    if result.FormattedRecords != nil {
        purchasables, err = extractPurchasablesFromJson(*result.FormattedRecords)
        if err != nil {
            return nil, fmt.Errorf("error extracting purchasables from JSON: %w", err)
        }
    } else {
        return nil, fmt.Errorf("no formatted records found")
    }

    return purchasables, nil
}

func (s *PurchasableService) UpdatePurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, purchasable internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {

    params := map[string]interface{}{
        "id":                  id,
        "name":                  purchasable.Name,
        "item_type":                  purchasable.ItemType,
		"cost": purchasable.Cost,
        "currency":                  purchasable.Currency,
        "inventory":                  purchasable.Inventory,
		"charge_recurrence_interval":		purchasable.ChargeRecurrenceInterval,
		"charge_recurrence_interval_count":		purchasable.ChargeRecurrenceIntervalCount,
		"charge_recurrence_end_date":		purchasable.ChargeRecurrenceEndDate,
		"donation_ratio": purchasable.DonationRatio,
    }

	query, sqlParams := buildUpdatePurchasablesQuery(params)
    if query == "" {
        return nil, fmt.Errorf("no fields provided for update")
    }
	log.Printf("sqlParams return: %v", query)

    // Convert parameters to RDS types
	rdsParams, err := buildSqlPurchasableParams(sqlParams)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the SQL statement
    result, err := rdsClient.ExecStatement(ctx, query, rdsParams)
    if err != nil {
        return nil, fmt.Errorf("failed to update purchasable: %w", err)
    }

    // Extract the updated purchasable from the formattedRecords JSON
    updatedPurchasable, err := extractAndMapSinglePurchasableFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to extract updated purchasable: %w", err)
    }

    // Since we are expecting one purchasable in the result
    return updatedPurchasable, nil
}

func (s *PurchasableService) DeletePurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
 	query := "DELETE FROM purchasables WHERE id = :id"
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
		return fmt.Errorf("failed to delete purchasable: %w", err)
	}
	// Check if any rows were affected by the delete operation
	if result.NumberOfRecordsUpdated == 0 {
		// No rows were affected, meaning the purchasable was not found
		return fmt.Errorf("purchasable not found")
	}

	// No need to return purchasable info; just return nil to indicate success.
	return nil
}

type MockPurchasableService struct {
	InsertPurchasableFunc        func(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.PurchasableInsert) (*internal_types.Purchasable, error)
	GetPurchasableByIDFunc       func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error)
	GetPurchasablesByUserIDFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.Purchasable, error)
	UpdatePurchasableFunc        func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.PurchasableUpdate) (*internal_types.Purchasable, error)
	DeletePurchasableFunc        func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error
}

// Implement the required methods

func (m *MockPurchasableService) InsertPurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, user internal_types.PurchasableInsert) (*internal_types.Purchasable, error) {
	if m.InsertPurchasableFunc != nil {
		return m.InsertPurchasableFunc(ctx, rdsClient, user)
	}
	return nil, nil
}

func (m *MockPurchasableService) GetPurchasableByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.Purchasable, error) {
	if m.GetPurchasableByIDFunc != nil {
		return m.GetPurchasableByIDFunc(ctx, rdsClient, id)
	}
	return nil, nil
}

func (m *MockPurchasableService) GetPurchasablesByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.Purchasable, error) {
	if m.GetPurchasablesByUserIDFunc != nil {
		return m.GetPurchasablesByUserIDFunc(ctx, rdsClient, userID)
	}
	return nil, nil
}

func (m *MockPurchasableService) UpdatePurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, user internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {
	if m.UpdatePurchasableFunc != nil {
		return m.UpdatePurchasableFunc(ctx, rdsClient, id, user)
	}
	return nil, nil
}

func (m *MockPurchasableService) DeletePurchasable(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error {
	if m.DeletePurchasableFunc != nil {
		return m.DeletePurchasableFunc(ctx, rdsClient, id)
	}
	return nil
}

