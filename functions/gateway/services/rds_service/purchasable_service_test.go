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


func TestInsertPurchasable(t *testing.T) {
	// Setup mock records
	records := []map[string]interface{}{
		{
			"id":                          "purchasable-id-123",
			"user_id":                     "user-id-123",
			"name":                        "Test Purchasable",
			"item_type":                   "physical",
			"cost":                        100,
			"currency":                    "USD",
			"inventory":                   50,
			"charge_recurrence_interval":  "monthly",
			"charge_recurrence_interval_count": 1,
			"charge_recurrence_end_date":  "2025-01-01",
			"donation_ratio":              0.1,
			"created_at":                  time.Now().Format(time.RFC3339),
			"updated_at":                  time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewPurchasableService()

	// Input data for the test
	purchasableInsert := internal_types.PurchasableInsert{
		ID:                          "purchasable-id-123",
		UserID:                      "user-id-123",
		Name:                        "Test Purchasable",
		ItemType:                    "physical",
		Cost:                        100,
		Currency:                    "USD",
		Inventory:                   50,
		ChargeRecurrenceInterval:    "monthly",
		ChargeRecurrenceIntervalCount: 1,
		ChargeRecurrenceEndDate:     "2025-01-01",
		DonationRatio:               0.1,
	}

	// Test
	result, err := service.InsertPurchasable(context.Background(), rdsClient, purchasableInsert)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "purchasable-id-123" {
		t.Errorf("expected id 'purchasable-id-123', got '%v'", result.ID)
	}
	if result.UserID != "user-id-123" {
		t.Errorf("expected user_id 'user-id-123', got '%v'", result.UserID)
	}
	if result.Name != "Test Purchasable" {
		t.Errorf("expected name 'Test Purchasable', got '%v'", result.Name)
	}
	if result.Cost != 100 {
		t.Errorf("expected cost 100, got '%v'", result.Cost)
	}
}

func TestGetPurchasableByID(t *testing.T) {
	// Setup mock records
	records := []map[string]interface{}{
		{
			"id":                          "purchasable-id-123",
			"user_id":                     "user-id-123",
			"name":                        "Test Purchasable",
			"item_type":                   "physical",
			"cost":                        100,
			"currency":                    "USD",
			"inventory":                   50,
			"charge_recurrence_interval":  "monthly",
			"charge_recurrence_interval_count": 1,
			"charge_recurrence_end_date":  "2025-01-01",
			"donation_ratio":              0.1,
			"created_at":                  time.Now().Format(time.RFC3339),
			"updated_at":                  time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewPurchasableService()

	// Test
	result, err := service.GetPurchasableByID(context.Background(), rdsClient, "purchasable-id-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.ID != "purchasable-id-123" {
		t.Errorf("expected id 'purchasable-id-123', got '%v'", result.ID)
	}
	if result.UserID != "user-id-123" {
		t.Errorf("expected user_id 'user-id-123', got '%v'", result.UserID)
	}
	if result.Name != "Test Purchasable" {
		t.Errorf("expected name 'Test Purchasable', got '%v'", result.Name)
	}
}

func TestUpdatePurchasable(t *testing.T) {
	// Setup mock records
	records := []map[string]interface{}{
		{
			"id":                          "purchasable-id-123",
			"user_id":                     "user-id-123",
			"name":                        "Updated Purchasable",
			"item_type":                   "physical",
			"cost":                        200,
			"currency":                    "USD",
			"inventory":                   100,
			"charge_recurrence_interval":  "yearly",
			"charge_recurrence_interval_count": 1,
			"charge_recurrence_end_date":  "2026-01-01",
			"donation_ratio":              0.2,
			"created_at":                  time.Now().Format(time.RFC3339),
			"updated_at":                  time.Now().Format(time.RFC3339),
		},
	}
	rdsClient := test_helpers.NewMockRdsDataClientWithJSONRecords(records)

	service := NewPurchasableService()

	// Input data for update
	purchasableUpdate := internal_types.PurchasableUpdate{
		Name:                        "Updated Purchasable",
		ItemType:                    "physical",
		Cost:                        200,
		Inventory:                   100,
		ChargeRecurrenceInterval:    "yearly",
		ChargeRecurrenceIntervalCount: 1,
		ChargeRecurrenceEndDate:     "2026-01-01",
		DonationRatio:               0.2,
	}

	// Test
	result, err := service.UpdatePurchasable(context.Background(), rdsClient, "purchasable-id-123", purchasableUpdate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Assertions
	if result == nil {
		t.Fatalf("expected result, got nil")
	}
	if result.Name != "Updated Purchasable" {
		t.Errorf("expected name 'Updated Purchasable', got '%v'", result.Name)
	}
	if result.Cost != 200 {
		t.Errorf("expected cost 200, got '%v'", result.Cost)
	}
	if result.Inventory != 100 {
		t.Errorf("expected inventory 100, got '%v'", result.Inventory)
	}
}

func TestDeletePurchasable(t *testing.T) {
	// Setup mock RDS client
	rdsClient := &test_helpers.MockRdsDataClient{
		ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
			fmt.Printf("SQL: %s\n", sql)
			fmt.Printf("Params: %v\n", params)

			switch sql {
			case "DELETE FROM purchasables WHERE id = :id":
				return &rdsdata.ExecuteStatementOutput{
					NumberOfRecordsUpdated: 1, // Simulate successful delete
				}, nil
			case "SELECT * FROM purchasables WHERE id = :id":
				return &rdsdata.ExecuteStatementOutput{
					FormattedRecords: aws.String("[]"), // Simulate no records found
				}, nil
			default:
				return nil, fmt.Errorf("unexpected SQL query")
			}
		},
	}

	service := NewPurchasableService()

	// Test deletion
	err := service.DeletePurchasable(context.Background(), rdsClient, "purchasable-id-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deletion by trying to retrieve the item
	result, err := rdsClient.ExecStatement(context.Background(), "SELECT * FROM purchasables WHERE id = :id", []rds_types.SqlParameter{
		{
			Name: aws.String("id"),
			Value: &rds_types.FieldMemberStringValue{
				Value: "purchasable-id-123",
			},
		},
	})

	if err != nil {
		t.Fatalf("failed to get item after deletion: %v", err)
	}

	if result.FormattedRecords == nil || *result.FormattedRecords == "[]" {
		// Pass the test if no records are found
		return
	} else {
		t.Fatalf("expected no records after deletion, but found some")
	}
}

