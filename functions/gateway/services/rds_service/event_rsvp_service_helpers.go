package rds_service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func buildSqlEventRsvpParams(parameters map[string]interface{}) ([]rds_types.SqlParameter, error) {
	var params []rds_types.SqlParameter

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

	// Might need the reference
	// UserID (UUID)
	// Check and add the user_id parameter if present and valid
	if userIdValue, ok := parameters["user_id"].(string); ok && userIdValue != "" {
		userID := rds_types.SqlParameter{
			Name:     aws.String("user_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: userIdValue,
			},
		}
		params = append(params, userID)
	}

	// Event Id
	if eventIdValue, ok := parameters["event_id"].(string); ok && eventIdValue != "" {
		eventID := rds_types.SqlParameter{
			Name:     aws.String("event_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: eventIdValue,
			},
		}
		params = append(params, eventID)
	}

	if eventSourceIdValue, ok := parameters["event_source_id"].(string); ok && eventSourceIdValue != "" {
		eventSourceID := rds_types.SqlParameter{
			Name:     aws.String("event_source_id"),
			TypeHint: "UUID",
			Value: &rds_types.FieldMemberStringValue{
				Value: eventSourceIdValue,
			},
		}
		params = append(params, eventSourceID)
	}

	// TODO: do we need check in enum before DB?
	// Status
	statusValue, ok := parameters["status"].(string)
	if !ok {
		return nil, fmt.Errorf("status is not a valid string")
	}
	status := rds_types.SqlParameter{
		Name: aws.String("status"),
		Value: &rds_types.FieldMemberStringValue{
			Value: statusValue,
		},
	}
	params = append(params, status)

	// EventSourceType
	eventSourceTypeValue, ok := parameters["event_source_type"].(string)
	if !ok {
		return nil, fmt.Errorf("event source type is not a valid string")
	}
	eventSourceType := rds_types.SqlParameter{
		Name: aws.String("event_source_type"),
		Value: &rds_types.FieldMemberStringValue{
			Value: eventSourceTypeValue,
		},
	}
	params = append(params, eventSourceType)

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

	return params, nil
}

func extractAndMapSingleEventRsvpFromJSON(formattedRecords string) (*internal_types.EventRsvp, error) {
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON records: %v", err)
	}

	// Example assuming only one record for simplicity
	if len(records) == 0 {
		return nil, fmt.Errorf("no records found")
	}

	record := records[0]


	eventRsvp := internal_types.EventRsvp{
		ID:                           getString(record, "id"),
		UserID:                       getString(record, "user_id"),
		EventID:                       getString(record, "event_id"),
		EventSourceType:                       getString(record, "event_source_type"),
		EventSourceID:                       getString(record, "event_source_id"),
		Status:                       getString(record, "status"),
		CreatedAt:                    getTime(record, "created_at"),
		UpdatedAt:                    getTime(record, "updated_at"),
	}


	return &eventRsvp, nil
}

func extractEventRsvpsFromJson(formattedRecords string) ([]internal_types.EventRsvp, error) {
	var purchasables []internal_types.EventRsvp

	// Parse formattedRecords as a JSON array
	var records []map[string]interface{}
	if err := json.Unmarshal([]byte(formattedRecords), &records); err != nil {
		return nil, fmt.Errorf("error unmarshaling formatted records: %w", err)
	}
	for _, record := range records {
		var purchasable internal_types.EventRsvp

		// Map fields from record to purchasable struct
		purchasable.ID = getString(record, "id")
		purchasable.UserID = getString(record, "user_id")
		purchasable.EventID = getString(record, "event_id")
		purchasable.EventSourceType = getString(record, "event_source_type")
		purchasable.EventSourceID = getString(record, "event_source_id")
		purchasable.Status = getString(record, "status")
		purchasable.CreatedAt = getTime(record, "createdAt")
		purchasable.UpdatedAt = getTime(record, "updatedAt")

		purchasables = append(purchasables, purchasable)
	}

	return purchasables, nil
}


func buildUpdateEventRsvpQuery(params map[string]interface{}) (string, map[string]interface{}) {
    // Initialize the SQL query parts
    setClauses := []string{}
    sqlParams := map[string]interface{}{}

    // Iterate through the params map
    for key, value := range params {
        if value != nil && value != "" {
            // Build the SET clause dynamically
            setClauses = append(setClauses, fmt.Sprintf("%s = :%s", key, key))
            sqlParams[key] = value
        } else {
			sqlParams[key] = ""
		}
    }

    // If no fields are provided, return an error or an empty query
    if len(setClauses) == 0 {
        return "", nil
    }

    // Construct the full SQL query
    query := fmt.Sprintf(`
        UPDATE event_rsvps
        SET %s,
            updated_at = now()
        WHERE id = :id
        RETURNING id, user_id, event_id, event_source_type, event_source_id, status,
			created_at, updated_at`,
        strings.Join(setClauses, ", "))

    // Ensure 'id' is always included in the parameters
    if _, ok := sqlParams["id"]; !ok {
        return "", nil // or return an error if `id` is a required field
    }

    return query, sqlParams
}


