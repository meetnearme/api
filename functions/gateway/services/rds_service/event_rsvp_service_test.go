package rds_service

import (
	"context"
	"errors"
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
			"event_source_id": "source-id-123",
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
		EventSourceID: "source-id-123",
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

func TestBuildSqlEventRsvpParams(t *testing.T) {
	tests := []struct {
		name      string
		parameters map[string]interface{}
		expected  []rds_types.SqlParameter
		expectErr bool
	}{
		{
			name: "valid parameters",
			parameters: map[string]interface{}{
				"id":                 "1234",
				"user_id":            "5678",
				"event_id":           "abcd",
				"status":             "attending",
				"event_source_type":  "web",
				"created_at":         "2024-01-01 12:00:00",
				"updated_at":         "2024-01-02 12:00:00",
			},
			expected: []rds_types.SqlParameter{
				{
					Name:     aws.String("id"),
					TypeHint: "UUID",
					Value: &rds_types.FieldMemberStringValue{
						Value: "1234",
					},
				},
				{
					Name:     aws.String("user_id"),
					TypeHint: "UUID",
					Value: &rds_types.FieldMemberStringValue{
						Value: "5678",
					},
				},
				{
					Name:     aws.String("event_id"),
					TypeHint: "UUID",
					Value: &rds_types.FieldMemberStringValue{
						Value: "abcd",
					},
				},
				{
					Name: aws.String("status"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "attending",
					},
				},
				{
					Name: aws.String("event_source_type"),
					Value: &rds_types.FieldMemberStringValue{
						Value: "web",
					},
				},
				{
					Name:     aws.String("created_at"),
					TypeHint: "TIMESTAMP",
					Value: &rds_types.FieldMemberStringValue{
						Value: "2024-01-01 12:00:00",
					},
				},
				{
					Name:     aws.String("updated_at"),
					TypeHint: "TIMESTAMP",
					Value: &rds_types.FieldMemberStringValue{
						Value: "2024-01-02 12:00:00",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "missing id",
			parameters: map[string]interface{}{
				"user_id":           "5678",
				"event_id":          "abcd",
				"status":            "attending",
				"event_source_type": "web",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "non-string id",
			parameters: map[string]interface{}{
				"id":                 1234,
				"user_id":            "5678",
				"event_id":           "abcd",
				"status":             "attending",
				"event_source_type":  "web",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "missing status",
			parameters: map[string]interface{}{
				"id":        "1234",
				"user_id":   "5678",
				"event_id":  "abcd",
				"event_source_type": "web",
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "non-string status",
			parameters: map[string]interface{}{
				"id":                 "1234",
				"user_id":            "5678",
				"event_id":           "abcd",
				"status":             0,
				"event_source_type":  "web",
			},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSqlEventRsvpParams(tt.parameters)
			if (err != nil) != tt.expectErr {
				t.Errorf("buildSqlEventRsvpParams() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if len(got) != len(tt.expected) {
				t.Errorf("buildSqlEventRsvpParams() = %v, want %v", got, tt.expected)
				return
			}
			for i, g := range got {
				e := tt.expected[i]
				if *g.Name != *e.Name || g.TypeHint != e.TypeHint || g.Value.(*rds_types.FieldMemberStringValue).Value != e.Value.(*rds_types.FieldMemberStringValue).Value {
					t.Errorf("buildSqlEventRsvpParams() = %v, want %v", got, tt.expected)
					return
				}
			}
		})
	}
}

func TestExtractAndMapSingleEventRsvpFromJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		expected  *internal_types.EventRsvp
		expectErr bool
	}{
		{
			name: "valid JSON",
			jsonInput: `[
				{
					"id": "1234",
					"user_id": "5678",
					"event_id": "abcd",
					"event_source_type": "web",
					"status": "attending",
					"created_at": "2024-01-01 12:00:00",
					"updated_at": "2024-01-02 12:00:00"
				}
			]`,
			expected: &internal_types.EventRsvp{
				ID:                "1234",
				UserID:            "5678",
				EventID:           "abcd",
				EventSourceType:   "web",
				Status:            "attending",
				CreatedAt:         parseTime("2024-01-01 12:00:00", t),
				UpdatedAt:         parseTime("2024-01-02 12:00:00", t),
			},
			expectErr: false,
		},
		{
			name: "empty JSON",
			jsonInput: `[]`,
			expected:  nil,
			expectErr: true,
		},
		{
			name: "invalid JSON",
			jsonInput: `[{invalid json}]`,
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractAndMapSingleEventRsvpFromJSON(tt.jsonInput)
			if (err != nil) != tt.expectErr {
				t.Errorf("extractAndMapSingleEventRsvpFromJSON() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.expected == nil {
				if got != nil {
					t.Errorf("extractAndMapSingleEventRsvpFromJSON() = %v, want %v", got, tt.expected)
				}
				return
			}
			if *got != *tt.expected {
				t.Errorf("extractAndMapSingleEventRsvpFromJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}
func TestMockInsertEventRsvp(t *testing.T) {
	mockService := &MockEventRsvpService{
		InsertEventRsvpFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error) {
			return &internal_types.EventRsvp{ID: "rsvp-id"}, nil
		},
	}

	eventRsvpInsert := internal_types.EventRsvpInsert{
		// Fill in with appropriate fields
	}

	result, err := mockService.InsertEventRsvp(context.Background(), nil, eventRsvpInsert)
	if err != nil {
		t.Errorf("InsertEventRsvpFunc returned an error: %v", err)
	}

	if result.ID != "rsvp-id" {
		t.Errorf("expected ID to be %s, got %s", "rsvp-id", result.ID)
	}
}

func TestMockGetEventRsvpByID(t *testing.T) {
	mockService := &MockEventRsvpService{
		GetEventRsvpByIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.EventRsvp, error) {
			if id == "rsvp-id" {
				return &internal_types.EventRsvp{ID: id}, nil
			}
			return nil, errors.New("rsvp not found")
		},
	}

	result, err := mockService.GetEventRsvpByID(context.Background(), nil, "rsvp-id")
	if err != nil {
		t.Errorf("GetEventRsvpByIDFunc returned an error: %v", err)
	}

	if result.ID != "rsvp-id" {
		t.Errorf("expected ID to be %s, got %s", "rsvp-id", result.ID)
	}

	// Test for a case where the RSVP is not found
	_, err = mockService.GetEventRsvpByID(context.Background(), nil, "non-existent-id")
	if err == nil {
		t.Errorf("expected an error for non-existent RSVP ID, got nil")
	}
}

func TestMockUpdateEventRsvp(t *testing.T) {
	mockService := &MockEventRsvpService{
		UpdateEventRsvpFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {
			if id == "rsvp-id" {
				return &internal_types.EventRsvp{ID: id}, nil
			}
			return nil, errors.New("rsvp not found")
		},
	}

	updateEventRsvp := internal_types.EventRsvpUpdate{
		// Fill in with appropriate fields
	}

	result, err := mockService.UpdateEventRsvp(context.Background(), nil, "rsvp-id", updateEventRsvp)
	if err != nil {
		t.Errorf("UpdateEventRsvpFunc returned an error: %v", err)
	}

	if result.ID != "rsvp-id" {
		t.Errorf("expected ID to be %s, got %s", "rsvp-id", result.ID)
	}
}

func TestMockDeleteEventRsvp(t *testing.T) {
	mockService := &MockEventRsvpService{
		DeleteEventRsvpFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) error {
			if id == "rsvp-id" {
				return nil
			}
			return errors.New("rsvp not found")
		},
	}

	// Test for successful deletion
	err := mockService.DeleteEventRsvp(context.Background(), nil, "rsvp-id")
	if err != nil {
		t.Errorf("DeleteEventRsvpFunc returned an error: %v", err)
	}

	// Test for RSVP not found case
	err = mockService.DeleteEventRsvp(context.Background(), nil, "non-existent-id")
	if err == nil {
		t.Errorf("expected an error for non-existent RSVP ID, got nil")
	}
}

func TestMockGetEventRsvpsByUserID(t *testing.T) {
	mockService := &MockEventRsvpService{
		GetEventRsvpsByUserIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) {
			if userID == "test-user-id" {
				return []internal_types.EventRsvp{
					{ID: "rsvp-1"},
					{ID: "rsvp-2"},
				}, nil
			}
			return nil, errors.New("no rsvps found for user")
		},
	}

	result, err := mockService.GetEventRsvpsByUserID(context.Background(), nil, "test-user-id")
	if err != nil {
		t.Errorf("GetEventRsvpsByUserIDFunc returned an error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 RSVPs, got %d", len(result))
	}

	// Test for a case where no RSVPs are found
	_, err = mockService.GetEventRsvpsByUserID(context.Background(), nil, "non-existent-user-id")
	if err == nil {
		t.Errorf("expected an error for non-existent user ID, got nil")
	}
}

func TestMockGetEventRsvpsByEventID(t *testing.T) {
	mockService := &MockEventRsvpService{
		GetEventRsvpsByEventIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) {
			if eventID == "test-event-id" {
				return []internal_types.EventRsvp{
					{ID: "rsvp-1"},
					{ID: "rsvp-2"},
				}, nil
			}
			return nil, errors.New("no rsvps found for event")
		},
	}

	result, err := mockService.GetEventRsvpsByEventID(context.Background(), nil, "test-event-id")
	if err != nil {
		t.Errorf("GetEventRsvpsByEventIDFunc returned an error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 RSVPs, got %d", len(result))
	}

	// Test for a case where no RSVPs are found
	_, err = mockService.GetEventRsvpsByEventID(context.Background(), nil, "non-existent-event-id")
	if err == nil {
		t.Errorf("expected an error for non-existent event ID, got nil")
	}
}
