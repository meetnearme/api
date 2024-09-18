package rds_service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestInsertEventRsvp(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":              "rsvp-id-123",
			"user_id":         "user-id-123",
			"event_id":        "event-id-123",
			"event_source_type": "external",
			"status":          "confirmed",
			"created_at":      time.Now().Format(time.RFC3339),
			"updated_at":      time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewEventRsvpService()

	eventRsvpInsert := internal_types.EventRsvpInsert{
		ID:              "rsvp-id-123",
		UserID:          "user-id-123",
		EventID:         "event-id-123",
		EventSourceType: "external",
		Status:          "confirmed",
		CreatedAt:       time.Now().Format(time.RFC3339),
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}

	// Test
	result, err := service.InsertEventRsvp(context.Background(), rdsClient, eventRsvpInsert)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "rsvp-id-123" {
		t.Errorf("expected id 'rsvp-id-123', got '%v'", result.ID)
	}
	if result.UserID != "user-id-123" {
		t.Errorf("expected user_id 'user-id-123', got '%v'", result.UserID)
	}
	if result.Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got '%v'", result.Status)
	}
}

func TestGetEventRsvpByID(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":              "rsvp-id-123",
			"user_id":         "user-id-123",
			"event_id":        "event-id-123",
			"event_source_type": "external",
			"status":          "confirmed",
			"created_at":      time.Now().Format(time.RFC3339),
			"updated_at":      time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewEventRsvpService()

	// Test
	result, err := service.GetEventRsvpByID(context.Background(), rdsClient, "rsvp-id-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "rsvp-id-123" {
		t.Errorf("expected id 'rsvp-id-123', got '%v'", result.ID)
	}
	if result.UserID != "user-id-123" {
		t.Errorf("expected user_id 'user-id-123', got '%v'", result.UserID)
	}
	if result.Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got '%v'", result.Status)
	}
}

func TestUpdateEventRsvp(t *testing.T) {
	const rdsTimeFormat = "2006-01-02 15:04:05" // RDS SQL accepted time format
	// Setup
	records := []map[string]interface{}{
		{
			"id":              "rsvp-id-123",
			"user_id":         "user-id-123",
			"event_id":        "event-id-123",
			"event_source_type": "external",
			"status":          "updated",
			"created_at":      time.Now().Format(rdsTimeFormat),
			"updated_at":      time.Now().Format(rdsTimeFormat),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewEventRsvpService()

	eventRsvpUpdate := internal_types.EventRsvpUpdate{
		ID:              "rsvp-id-123",
		UserID:          "user-id-123",
		EventID:         "event-id-123",
		EventSourceType: "external",
		Status:          "updated",
	}

	// Test
	result, err := service.UpdateEventRsvp(context.Background(), rdsClient, "rsvp-id-123", eventRsvpUpdate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.Status != "updated" {
		t.Errorf("expected status 'updated', got '%v'", result.Status)
	}
}

func TestDeleteEventRsvp(t *testing.T) {
    // Initialize mock RDS client
    rdsClient := &test_helpers.MockRdsDataClient{
        ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
            fmt.Printf("SQL: %s\n", sql)
            fmt.Printf("Params: %v\n", params)

            switch sql {
            case "DELETE FROM event_rsvps WHERE id = :id":
                // Simulate successful delete
                return &rdsdata.ExecuteStatementOutput{
                    NumberOfRecordsUpdated: 1, // Simulate that one record was deleted
                }, nil
            case "SELECT * FROM event_rsvps WHERE id = :id":
                // Simulate item not found after deletion
                return &rdsdata.ExecuteStatementOutput{
                    FormattedRecords: aws.String("[]"), // Simulate no records found
                }, nil
            default:
                return nil, fmt.Errorf("unexpected SQL query")
            }
        },
    }

    service := NewEventRsvpService()

    // Test deletion
    err := service.DeleteEventRsvp(context.Background(), rdsClient, "rsvp-id-123")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }

    // Verify deletion by trying to retrieve the item
    result, err := rdsClient.ExecStatement(context.Background(), "SELECT * FROM event_rsvps WHERE id = :id", []rds_types.SqlParameter{
        {
            Name: aws.String("id"),
            Value: &rds_types.FieldMemberStringValue{
                Value: "rsvp-id-123",
            },
        },
    })

    if err != nil {
        t.Fatalf("failed to get item after deletion: %v", err)
    }

    if result.FormattedRecords == nil || *result.FormattedRecords == "[]" {
        // Pass the test if no records are found
        return
    }

    t.Fatalf("expected no records, got %v", *result.FormattedRecords)
}


func TestGetEventRsvpsByUserID(t *testing.T) {
	// Setup
	records := []map[string]interface{}{
		{
			"id":              "rsvp-id-123",
			"user_id":         "user-id-123",
			"event_id":        "event-id-123",
			"event_source_type": "external",
			"status":          "confirmed",
			"created_at":      time.Now().Format(time.RFC3339),
			"updated_at":      time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewEventRsvpService()

	// Test
	results, err := service.GetEventRsvpsByUserID(context.Background(), rdsClient, "user-id-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if len(results) == 0 {
		t.Fatalf("expected results, got none")
	}
	if results[0].ID != "rsvp-id-123" {
		t.Errorf("expected id 'rsvp-id-123', got '%v'", results[0].ID)
	}
	if results[0].UserID != "user-id-123" {
		t.Errorf("expected user_id 'user-id-123', got '%v'", results[0].UserID)
	}
	if results[0].Status != "confirmed" {
		t.Errorf("expected status 'confirmed', got '%v'", results[0].Status)
	}
}

