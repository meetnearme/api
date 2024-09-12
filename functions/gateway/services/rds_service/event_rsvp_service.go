// TODO: change all fmt to log printout in new rds handlers and services
package rds_service

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type EventRsvpService struct{}

func NewEventRsvpService() internal_types.EventRsvpServiceInterface {
	return &EventRsvpService{}
}

func (s *EventRsvpService) InsertEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error) {
    // Generate a new UUID if not provided
    if eventRsvp.ID == "" {
        eventRsvp.ID = uuid.New().String()
    }

    // Validate the eventRsvp object
    if err := validate.Struct(eventRsvp); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Construct the SQL query
    query := `
        INSERT INTO event_rsvps (
            id, user_id, event_id, event_source_type, status,
			created_at, updated_at
        )
        VALUES (
			:id, :user_id, :event_id, :event_source_type, :status,
			NOW(), NOW()
        )
        RETURNING
            id, user_id, event_id, event_source_type, status,
			created_at, updated_at
    `

    // Prepare the parameters for the query
    params := map[string]interface{}{
        "id":                  eventRsvp.ID,
		"user_id": eventRsvp.UserID,
		"event_id": eventRsvp.EventID,
		"event_source_type": eventRsvp.EventSourceType,
		"status": eventRsvp.Status,
    }

	log.Printf("params in create eventRsvp: %v", params)

	paramsRdsFormat, err := buildSqlEventRsvpParams(params)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the statement
    result, err := rdsClient.ExecStatement(ctx, query, paramsRdsFormat)
    if err != nil {
        return nil, fmt.Errorf("failed to insert eventRsvp: %w", err)
    }

    // Extract the inserted eventRsvp data
    insertedEventRsvp, err := extractAndMapSingleEventRsvpFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map eventRsvp after insert: %w", err)
    }

	return insertedEventRsvp, nil
}


func (s *EventRsvpService) GetEventRsvpByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.EventRsvp, error) {
	query := "SELECT * FROM event_rsvps WHERE id = :id"
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
		return nil, fmt.Errorf("failed to get eventRsvp: %w", err)
	}

	log.Printf("Result in getby id: %v", result)
	log.Printf("Result formatted result: %v", result.FormattedRecords)

    // Extract the inserted eventRsvp data
    eventRsvp, err := extractAndMapSingleEventRsvpFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to map eventRsvp after insert: %w", err)
    }

    // return eventRsvp, nil
	return eventRsvp, nil
}

func (s *EventRsvpService) GetEventRsvpsByEventID(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) {
    // Updated query to filter eventRsvps by user_id and limit to 10 records
	query := `
		SELECT *
		FROM event_rsvps
		WHERE event_id = :event_id
	`

	// UserID (UUID)
	params := []rds_types.SqlParameter{
		{
			Name:  aws.String("event_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: eventID,
			},
		},
	}


    // Execute the SQL query
    result, err := rdsClient.ExecStatement(ctx, query, params)
    if err != nil {
        return nil, fmt.Errorf("failed to get eventRsvps: %w", err)
    }

    var eventRsvps []internal_types.EventRsvp

    // Check if formattedRecords is available
    if result.FormattedRecords != nil {
        eventRsvps, err = extractEventRsvpsFromJson(*result.FormattedRecords)
        if err != nil {
            return nil, fmt.Errorf("error extracting eventRsvps from JSON: %w", err)
        }
		log.Printf("eventRsvps form get all by user id: %v", eventRsvps)
    } else {
        return nil, fmt.Errorf("no formatted records found")
    }

    return eventRsvps, nil
}

func (s *EventRsvpService) GetEventRsvpsByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) {
    // Updated query to filter eventRsvps by user_id and limit to 10 records
	query := `
		SELECT *
		FROM event_rsvps
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
        return nil, fmt.Errorf("failed to get eventRsvps: %w", err)
    }

    var eventRsvps []internal_types.EventRsvp

    // Check if formattedRecords is available
    if result.FormattedRecords != nil {
        eventRsvps, err = extractEventRsvpsFromJson(*result.FormattedRecords)
        if err != nil {
            return nil, fmt.Errorf("error extracting eventRsvps from JSON: %w", err)
        }
		log.Printf("this in service func eventRsvps form get all by user id: %v", eventRsvps)
    } else {
        return nil, fmt.Errorf("no formatted records found")
    }

    return eventRsvps, nil
}

func (s *EventRsvpService) UpdateEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {

    params := map[string]interface{}{
        "id":                  id,
		"user_id": eventRsvp.UserID,
		"event_id": eventRsvp.EventID,
		"event_source_type": eventRsvp.EventSourceType,
		"status": eventRsvp.Status,
    }
	log.Printf("params in updtae rsvp: %v", reflect.TypeOf(params["user_id"]))

	query, sqlParams := buildUpdateEventRsvpQuery(params)
    if query == "" {
        return nil, fmt.Errorf("no fields provided for update")
    }
	log.Printf("sqlParams return: %v", sqlParams)

    // Convert parameters to RDS types
	rdsParams, err := buildSqlEventRsvpParams(sqlParams)
	if err != nil {
		return nil, fmt.Errorf("Error in building RDS formatted SQL Parameters: %w", err)
	}

    // Execute the SQL statement
    result, err := rdsClient.ExecStatement(ctx, query, rdsParams)
    if err != nil {
        return nil, fmt.Errorf("failed to update eventRsvp: %w", err)
    }

    // Extract the updated eventRsvp from the formattedRecords JSON
    updatedEventRsvp, err := extractAndMapSingleEventRsvpFromJSON(*result.FormattedRecords)
    if err != nil {
        return nil, fmt.Errorf("failed to extract updated eventRsvp: %w", err)
    }

    // Since we are expecting one eventRsvp in the result
    return updatedEventRsvp, nil
}

func (s *EventRsvpService) DeleteEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
 	query := "DELETE FROM event_rsvps WHERE id = :id"
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
		return fmt.Errorf("failed to delete eventRsvp: %w", err)
	}
	// Check if any rows were affected by the delete operation
	if result.NumberOfRecordsUpdated == 0 {
		// No rows were affected, meaning the eventRsvp was not found
		return fmt.Errorf("event_rsvp not found")
	}

	// No need to return eventRsvp info; just return nil to indicate success.
	return nil
}

type MockEventRsvpService struct {
	InsertEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error)
	GetEventRsvpByIDFunc func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.EventRsvp, error)
	GetEventRsvpsByUserIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) // New function
	GetEventRsvpsByEventIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) // New function
	UpdateEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error)
	DeleteEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error
}

func (m *MockEventRsvpService) InsertEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error) {
	return m.InsertEventRsvpFunc(ctx, rdsClient, eventRsvp)
}

func (m *MockEventRsvpService) GetEventRsvpByID(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.EventRsvp, error) {
	return m.GetEventRsvpByIDFunc(ctx, rdsClient, id)
}

func (m *MockEventRsvpService) UpdateEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {
	return m.UpdateEventRsvpFunc(ctx, rdsClient, id, eventRsvp)
}

func (m *MockEventRsvpService) DeleteEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string)  error {
	return m.DeleteEventRsvpFunc(ctx, rdsClient, id)
}

func (m *MockEventRsvpService) GetEventRsvpsByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) {
	return m.GetEventRsvpsByUserIDFunc(ctx, rdsClient, userID)
}

func (m *MockEventRsvpService) GetEventRsvpsByEventID(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) {
	return m.GetEventRsvpsByEventIDFunc(ctx, rdsClient, eventID)
}



