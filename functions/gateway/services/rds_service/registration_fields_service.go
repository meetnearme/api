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

type RegistrationFieldsService struct{}

func NewRegistrationFieldsService() internal_types.RegistrationFieldsServiceInterface {
	return &RegistrationFieldsService{}
}

func (s *RegistrationFieldsService) InsertRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, registrationFields internal_types.RegistrationFieldsInsert) (*internal_types.RegistrationFields, error) {
    // Generate a new UUID if not provided
    if registrationFields.ID == "" {
        registrationFields.ID = uuid.New().String()
    }

    // Validate the registrationFields object
    if err := validate.Struct(registrationFields); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Construct the SQL query
    query := `
        INSERT INTO registration_fields (
            id, name, type, options, required, default_val, placeholder, description,
			created_at, updated_at
        )
        VALUES (
			:id, :name, :type, :options, :required, :default_val, :placeholder, :description,
			NOW(), NOW()
        )
        RETURNING
            id, name, type, options, required, default_val, placeholder, description,
			created_at, updated_at
    `

    // Prepare the parameters for the query
    params := map[string]interface{}{
        "id":                  registrationFields.ID,
        "name":                  registrationFields.Name,
        "type":                  registrationFields.Type,
        "options":                  registrationFields.Options,
        "default_val":                  registrationFields.Default,
        "placeholder":                  registrationFields.Placeholder,
        "description":                  registrationFields.Description,
        "required":                  registrationFields.Required,
    }

	log.Printf("params in create registrationFields: %v", params)

	paramsRdsFormat, err := buildSqlRegistrationFieldsParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the statement
    result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to insert registrationFields: %w", err)
    }

    // Extract the inserted registrationFields data
    insertedRegistrationFields, err := extractAndMapSingleRegistrationFieldsFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map registrationFields after insert: %w", err)
    }

	return insertedRegistrationFields, nil
}


func (s *RegistrationFieldsService) GetRegistrationFieldsByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.RegistrationFields, error) {
	query := "SELECT * FROM registration_fields WHERE id = :id"
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
		return nil, fmt.Errorf("failed to get registrationFields: %w", err)
	}

	log.Printf("Result in getby id: %v", result)
	log.Printf("Result formatted result: %v", result.FormattedRecords)

    // Extract the inserted registrationFields data
    registrationFields, err := extractAndMapSingleRegistrationFieldsFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map registrationFields after insert: %w", err)
    }

    // return registrationFields, nil
	return registrationFields, nil
}


func (s *RegistrationFieldsService) UpdateRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error) {

    params := map[string]interface{}{
        "id":                  id,
		"name": registrationFields.Name,
		"type": registrationFields.Type,
		"options": registrationFields.Options,
		"default_val": registrationFields.Default,
		"placeholder": registrationFields.Placeholder,
		"description": registrationFields.Description,
		"required": registrationFields.Required,
    }

	query, sqlParams := buildUpdateRegistrationFieldsQuery(params)
    if query == "" {
        return nil, fmt.Errorf("no fields provided for update")
    }
	log.Printf("sqlParams return: %v", sqlParams)

    // Convert parameters to RDS types
	rdsParams, err := buildSqlRegistrationFieldsParams(sqlParams)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the SQL statement
    result, err := rdsClient.ExecStatement(ctx, query, rdsParams)
    if err != nil {
        return nil, fmt.Errorf("failed to update registration fields: %w", err)
    }

    // Extract the updated registrationFields from the formattedRecords JSON
    updatedRegistrationFields, err := extractAndMapSingleRegistrationFieldsFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to extract updated registrationFields: %w", err)
    }

    // Since we are expecting one registrationFields in the result
    return updatedRegistrationFields, nil
}

func (s *RegistrationFieldsService) DeleteRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
 	query := "DELETE FROM registration_fields WHERE id = :id"
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
		return fmt.Errorf("failed to delete registrationFields: %w", err)
	}
	// Check if any rows were affected by the delete operation
	if result.NumberOfRecordsUpdated == 0 {
		// No rows were affected, meaning the registrationFields was not found
		return fmt.Errorf("event_rsvp not found")
	}

	// No need to return registrationFields info; just return nil to indicate success.
	return nil
}

type MockRegistrationFieldsService struct {
	InsertRegistrationFieldsFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, registrationFields internal_types.RegistrationFieldsInsert) (*internal_types.RegistrationFields, error)
	GetRegistrationFieldsByIDFunc func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.RegistrationFields, error)
	GetRegistrationFieldssByUserIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.RegistrationFields, error) // New function
	GetRegistrationFieldssByEventIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.RegistrationFields, error) // New function
	UpdateRegistrationFieldsFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error)
	DeleteRegistrationFieldsFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error
}

func (m *MockRegistrationFieldsService) InsertRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, registrationFields internal_types.RegistrationFieldsInsert) (*internal_types.RegistrationFields, error) {
	return m.InsertRegistrationFieldsFunc(ctx, rdsClient, registrationFields)
}

func (m *MockRegistrationFieldsService) GetRegistrationFieldsByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.RegistrationFields, error) {
	return m.GetRegistrationFieldsByIDFunc(ctx, rdsClient, id)
}

func (m *MockRegistrationFieldsService) UpdateRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error) {
	return m.UpdateRegistrationFieldsFunc(ctx, rdsClient, id, registrationFields)
}

func (m *MockRegistrationFieldsService) DeleteRegistrationFields(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
	return m.DeleteRegistrationFieldsFunc(ctx, rdsClient, id)
}




