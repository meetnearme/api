package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/constants"
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

func TestGetSeshuSessionAppliesDefaults(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"url":               &types.AttributeValueMemberS{Value: "https://defaults.com"},
					"status":            &types.AttributeValueMemberS{Value: "draft"},
					"locationLatitude":  &types.AttributeValueMemberN{Value: "0"},
					"locationLongitude": &types.AttributeValueMemberN{Value: "0"},
				},
			}, nil
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionGet{Url: "https://defaults.com"}

	result, err := GetSeshuSession(ctx, mockDB, seshuPayload)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Fatalf("Expected non-nil result")
	}
	if result.LocationLatitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("Expected latitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, result.LocationLatitude)
	}
	if result.LocationLongitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("Expected longitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, result.LocationLongitude)
	}
	if result.EventCandidates == nil {
		t.Fatalf("Expected event candidates to be initialized, got nil")
	}
	if len(result.EventCandidates) != 0 {
		t.Fatalf("Expected event candidates to be empty, got %d", len(result.EventCandidates))
	}
	if result.EventValidations == nil {
		t.Fatalf("Expected event validations to be initialized, got nil")
	}
	if len(result.EventValidations) != 0 {
		t.Fatalf("Expected event validations to be empty, got %d", len(result.EventValidations))
	}
}

func TestGetSeshuSessionDBError(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return nil, fmt.Errorf("dynamo failure")
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionGet{Url: "https://test.com"}

	result, err := GetSeshuSession(ctx, mockDB, seshuPayload)

	if err == nil || !strings.Contains(err.Error(), "dynamo failure") {
		t.Fatalf("Expected dynamo failure, got result=%v err=%v", result, err)
	}
}

func TestGetSeshuSessionUnmarshalError(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"url":    &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
					"status": &types.AttributeValueMemberS{Value: "draft"},
				},
			}, nil
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionGet{Url: "https://test.com"}

	result, err := GetSeshuSession(ctx, mockDB, seshuPayload)

	if err == nil {
		t.Fatalf("Expected unmarshal error, got result=%v err=%v", result, err)
	}
}

func TestGetSeshuSessionMissingItem(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{}, nil
		},
	}

	ctx := context.Background()
	seshuPayload := internal_types.SeshuSessionGet{Url: "https://missing.com"}

	result, err := GetSeshuSession(ctx, mockDB, seshuPayload)
	if err != nil {
		t.Fatalf("Did not expect error for missing item, got %v", err)
	}
	if result == nil {
		t.Fatalf("Expected zero-value session, got nil")
	}
	if result.Url != "" || result.Status != "" {
		t.Fatalf("Expected zero-value fields for missing item, got %+v", result)
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

func TestInsertSeshuSessionEmptyTableName(t *testing.T) {
	originalTable := seshuSessionsTableName
	seshuSessionsTableName = ""
	defer func() {
		seshuSessionsTableName = originalTable
	}()

	mockDB := &test_helpers.MockDynamoDBClient{}

	ctx := context.Background()
	input := internal_types.SeshuSessionInput{SeshuSession: internal_types.SeshuSession{OwnerId: "owner", Url: "https://test.com"}}

	result, err := InsertSeshuSession(ctx, mockDB, input)
	if err == nil || !strings.Contains(err.Error(), "seshuSessionsTableName is empty") {
		t.Fatalf("Expected table name error, got result=%v err=%v", result, err)
	}
}

func TestInsertSeshuSessionPutError(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, fmt.Errorf("put failure")
		},
	}

	ctx := context.Background()
	input := internal_types.SeshuSessionInput{SeshuSession: internal_types.SeshuSession{OwnerId: "owner", Url: "https://test.com"}}

	result, err := InsertSeshuSession(ctx, mockDB, input)
	if err == nil || !strings.Contains(err.Error(), "put failure") {
		t.Fatalf("Expected put error, got result=%v err=%v", result, err)
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

func TestUpdateSeshuSessionEmptyTableName(t *testing.T) {
	originalTable := seshuSessionsTableName
	seshuSessionsTableName = ""
	defer func() {
		seshuSessionsTableName = originalTable
	}()

	mockDB := &test_helpers.MockDynamoDBClient{}
	ctx := context.Background()
	update := internal_types.SeshuSessionUpdate{Url: "https://test.com"}

	result, err := UpdateSeshuSession(ctx, mockDB, update)
	if err == nil || !strings.Contains(err.Error(), "seshuSessionsTableName is empty") {
		t.Fatalf("Expected table name error, got result=%v err=%v", result, err)
	}
}

func TestUpdateSeshuSessionPartialFields(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			if !strings.Contains(*params.UpdateExpression, "#status = :status") {
				t.Fatalf("Expected status field to be present in update expression, got %s", *params.UpdateExpression)
			}
			if strings.Contains(*params.UpdateExpression, "#urlDomain") {
				t.Fatalf("Did not expect urlDomain to be present when not provided")
			}
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}

	ctx := context.Background()
	update := internal_types.SeshuSessionUpdate{Url: "https://test.com", Status: "completed"}

	_, err := UpdateSeshuSession(ctx, mockDB, update)
	if err != nil {
		t.Fatalf("Expected no error for partial update, got %v", err)
	}
}
