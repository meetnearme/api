package services

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetSeshuSession(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"url":    &types.AttributeValueMemberS{Value: "https://test.com"},
					"status": &types.AttributeValueMemberS{Value: "draft"},
				},
			}, nil
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionGet{Url: "https://test.com"}

	result, err := GetSeshuSession(ctx, mockDB, seshuPayload)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result, got nil")
	}
	if result.Url != "https://test.com" {
		t.Errorf("Expected URL 'https://test.com', got '%s'", result.Url)
	}
	if result.Status != "draft" {
		t.Errorf("Expected status 'draft', got '%s'", result.Status)
	}
}

func TestInsertSeshuSession(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionInput{
		SeshuSession: internal_types.SeshuSession{
			OwnerId:           "testowner",
			Url:               "https://test.com",
			UrlDomain:         "test.com",
			Html:              "<html></html>",
			EventValidations:  []internal_types.EventBoolValid{{EventValidateTitle: true, EventValidateLocation: true, EventValidateStartTime: true}},
			EventCandidates:   []internal_types.EventInfo{{EventTitle: "Test Event", EventLocation: "Nowhere", EventStartTime: "1234567890"}},
			LocationAddress:   "1234 Nowhere St",
			LocationLatitude:  39.8616981506,
			LocationLongitude: -104.672996521,
			UrlQueryParams:    map[string][]string{"test": {"value"}},
		},
	}

	result, err := InsertSeshuSession(ctx, mockDB, seshuPayload)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result, got nil")
	}
	if result.OwnerId != "testowner" {
		t.Errorf("Expected OwnerId 'testowner', got '%s'", result.OwnerId)
	}
	if result.Url != "https://test.com" {
		t.Errorf("Expected URL 'https://test.com', got '%s'", result.Url)
	}
	if result.Status != "draft" {
		t.Errorf("Expected status 'draft', got '%s'", result.Status)
	}
}

func TestUpdateSeshuSession(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}

	ctx := context.Background()
	lat := 39.8616981506
	lon := -104.672996521
	seshuPayload := internal_types.SeshuSessionUpdate{
		Url:               "https://test.com",
		Status:            "completed",
		EventValidations:  []internal_types.EventBoolValid{{EventValidateTitle: true, EventValidateLocation: true, EventValidateStartTime: true}},
		EventCandidates:   []internal_types.EventInfo{{EventTitle: "Test Event", EventLocation: "Nowhere", EventStartTime: "1234567890"}},
		LocationAddress:   "1234 Nowhere St",
		LocationLatitude:  &lat,
		LocationLongitude: &lon,
		UrlQueryParams:    map[string][]string{"test": {"value"}},
	}

	_, err := UpdateSeshuSession(ctx, mockDB, seshuPayload)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
