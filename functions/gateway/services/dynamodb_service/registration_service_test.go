package dynamodb_service

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestInsertRegistration(t *testing.T) {
	mockDynamoDBClient := &test_helpers.MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, input *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	service := NewRegistrationService()

	now := time.Now()
	registration := internal_types.RegistrationInsert{
		EventId: "eventId",
		UserId:  "userId",
		Responses: []map[string]interface{}{
			{"question1": "answer1"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := service.InsertRegistration(context.TODO(), mockDynamoDBClient, registration, registration.EventId, registration.UserId)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestGetRegistrationByPk(t *testing.T) {
	mockDynamoDBClient := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"eventId": &types.AttributeValueMemberS{Value: "eventId"},
					"userId":  &types.AttributeValueMemberS{Value: "userId"},
					"responses": &types.AttributeValueMemberL{
						Value: []types.AttributeValue{
							&types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"question1": &types.AttributeValueMemberS{Value: "answer1"},
								},
							},
						},
					},
					"createdAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
					"updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
				},
			}, nil
		},
	}
	service := NewRegistrationService()

	result, err := service.GetRegistrationByPk(context.TODO(), mockDynamoDBClient, "eventId", "userId")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
	// Validate returned fields
	if result.EventId != "eventId" || result.UserId != "userId" || len(result.Responses) == 0 {
		t.Errorf("unexpected result: %+v", result)
	}
}
func TestGetRegistrationsByEventID(t *testing.T) {
	mockDynamoDBClient := &test_helpers.MockDynamoDBClient{
		QueryFunc: func(ctx context.Context, input *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"eventId": &types.AttributeValueMemberS{Value: "eventId"},
						"userId":  &types.AttributeValueMemberS{Value: "userId"},
						"responses": &types.AttributeValueMemberL{
							Value: []types.AttributeValue{
								&types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"question1": &types.AttributeValueMemberS{Value: "answer1"},
									},
								},
							},
						},
						"createdAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
						"updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
					},
				},
			}, nil
		},
	}
	service := NewRegistrationService()

	results, err := service.GetRegistrationsByEventID(context.TODO(), mockDynamoDBClient, "eventId")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(results) == 0 {
		t.Error("expected non-empty results")
	}
}

func TestUpdateRegistration(t *testing.T) {
	mockDynamoDBClient := &test_helpers.MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return &dynamodb.UpdateItemOutput{
				Attributes: map[string]types.AttributeValue{
					"eventId": &types.AttributeValueMemberS{Value: "eventId"},
					"userId":  &types.AttributeValueMemberS{Value: "userId"},
					"responses": &types.AttributeValueMemberL{
						Value: []types.AttributeValue{
							&types.AttributeValueMemberM{
								Value: map[string]types.AttributeValue{
									"question1": &types.AttributeValueMemberS{Value: "updatedAnswer"},
								},
							},
						},
					},
					"createdAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
					"updatedAt": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
				},
			}, nil
		},
	}
	service := NewRegistrationService()

	updatedRegistration, err := service.UpdateRegistration(context.TODO(), mockDynamoDBClient, "eventId", "userId", internal_types.RegistrationUpdate{
		Responses: []map[string]interface{}{
			{"question1": "updatedAnswer"},
		},
		UpdatedAt: time.Now(),
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if updatedRegistration == nil {
		t.Error("expected non-nil result")
	}
	// Validate returned fields
	if updatedRegistration.UserId != "userId" || len(updatedRegistration.Responses) == 0 {
		t.Errorf("unexpected result: %+v", updatedRegistration)
	}
}

func TestDeleteRegistration(t *testing.T) {
	mockDynamoDBClient := &test_helpers.MockDynamoDBClient{
		DeleteItemFunc: func(ctx context.Context, input *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}
	service := NewRegistrationService()

	err := service.DeleteRegistration(context.TODO(), mockDynamoDBClient, "eventId", "userId")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
