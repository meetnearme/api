package dynamodb_service

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetRegistrationFieldsByEventID_Success(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			// Mock the response returned by DynamoDB
			return &dynamodb.GetItemOutput{
				Item: map[string]dynamodb_types.AttributeValue{
					"eventId": &dynamodb_types.AttributeValueMemberS{Value: "test-event-id"},
					"fields":  &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
						// Field 1: attendeeEmail
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "attendeeEmail"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "text"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: ""},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: "email@example.com"},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated email in case of any changes"},
							},
						},
						// Field 2: tshirtSize
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "tshirtSize"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "select"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: "large"},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: ""},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated tshirt size for the event"},
								"options": &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "small"},
									&dynamodb_types.AttributeValueMemberS{Value: "medium"},
									&dynamodb_types.AttributeValueMemberS{Value: "large"},
									&dynamodb_types.AttributeValueMemberS{Value: "XL"},
								}},
							},
						},
						// Field 3: sessionPreference
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "sessionPreference"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "select"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: "morning"},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: ""},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "Please choose your preferred session"},
								"options": &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "morning"},
									&dynamodb_types.AttributeValueMemberS{Value: "evening"},
								}},
							},
						},
					}},
				},
			}, nil
		},
	}

	service := NewRegistrationFieldsService()

	// Call the service function
	result, err := service.GetRegistrationFieldsByEventID(context.TODO(), mockDB, "test-event-id")

	// Assert no error
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Assert the result is not nil
	if result == nil {
		t.Errorf("expected result, got nil")
	}

	// Assert the correct event ID is returned
	if result.EventId != "test-event-id" {
		t.Errorf("expected EventId to be 'test-event-id', got %v", result.EventId)
	}

	// Assert the correct fields structure is returned
	expectedFields := []internal_types.RegistrationField{
		{
			Name:        "attendeeEmail",
			Type:        "text",
			Required:    true,
			Default:     "",
			Placeholder: "email@example.com",
			Description: "We need your updated email in case of any changes",
		},
		{
			Name:        "tshirtSize",
			Type:        "select",
			Required:    true,
			Default:     "large",
			Placeholder: "",
			Description: "We need your updated tshirt size for the event",
			Options:     []string{"small", "medium", "large", "XL"},
		},
		{
			Name:        "sessionPreference",
			Type:        "select",
			Required:    true,
			Default:     "morning",
			Placeholder: "",
			Description: "Please choose your preferred session",
			Options:     []string{"morning", "evening"},
		},
	}

	if !reflect.DeepEqual(result.Fields, expectedFields) {
		t.Errorf("expected fields %v, got %v", expectedFields, result.Fields)
	}
}

func TestInsertRegistrationFields_ValidationError(t *testing.T) {
	// Set up your mock client and service
	mockDB := &test_helpers.MockDynamoDBClient{}
	service := &RegistrationFieldsService{}

	// Create an invalid registrationFields object
	invalidFields := internal_types.RegistrationFieldsInsert{} // Assume this fails validation

	_, err := service.InsertRegistrationFields(context.TODO(), mockDB, invalidFields, "event-id")
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}


func TestUpdateRegistrationFields_Error(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, fmt.Errorf("update error")
		},
	}

	service := &RegistrationFieldsService{}
	_, err := service.UpdateRegistrationFields(context.TODO(), mockDB, "event-id", internal_types.RegistrationFieldsUpdate{})
	if err == nil || err.Error() != "update error" {
		t.Errorf("expected 'update error', got %v", err)
	}
}

func TestGetRegistrationFieldsByEventID(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			// Check if the correct table and eventId are used in the request
			if *input.TableName != registrationFieldsTableName {
				t.Errorf("expected table name %s, got %s", registrationFieldsTableName, *input.TableName)
			}
			if input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value != "test-event-id" {
				t.Errorf("expected eventId %s, got %s", "test-event-id", input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value)
			}

			// Return mock response
			return &dynamodb.GetItemOutput{
				Item: map[string]dynamodb_types.AttributeValue{
					"eventId": &dynamodb_types.AttributeValueMemberS{Value: "test-event-id"},
					"fields":  &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "attendeeEmail"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "text"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: ""},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: "email@example.com"},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated email in case of any changes"},
							},
						},
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "tshirtSize"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "select"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: "large"},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: ""},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated tshirt size for the event"},
								"options": &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "small"},
									&dynamodb_types.AttributeValueMemberS{Value: "medium"},
									&dynamodb_types.AttributeValueMemberS{Value: "large"},
									&dynamodb_types.AttributeValueMemberS{Value: "XL"},
								}},
							},
						},
					}},
				},
			}, nil
		},
	}

	// Create the service
	service := &RegistrationFieldsService{}

	// Call the handler function
	result, err := service.GetRegistrationFieldsByEventID(context.TODO(), mockDB, "test-event-id")

	// Assert no error
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Assert that the result is not nil
	if result == nil {
		t.Errorf("expected non-nil result, got nil")
	}

	// Assert the correct event ID
	if result.EventId != "test-event-id" {
		t.Errorf("expected eventId 'test-event-id', got %s", result.EventId)
	}

	// Assert the fields are correct
	expectedFields := []internal_types.RegistrationField{
		{
			Name:        "attendeeEmail",
			Type:        "text",
			Required:    true,
			Default:     "",
			Placeholder: "email@example.com",
			Description: "We need your updated email in case of any changes",
		},
		{
			Name:        "tshirtSize",
			Type:        "select",
			Required:    true,
			Default:     "large",
			Placeholder: "",
			Description: "We need your updated tshirt size for the event",
			Options:     []string{"small", "medium", "large", "XL"},
		},
	}

	// Check if the fields in the result match the expected fields
	if !reflect.DeepEqual(result.Fields, expectedFields) {
		t.Errorf("expected fields %v, got %v", expectedFields, result.Fields)
	}
}


func TestUpdateRegistrationFields(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, input *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			// Check if the correct table is used
			if *input.TableName != registrationFieldsTableName {
				t.Errorf("expected table name %s, got %s", registrationFieldsTableName, *input.TableName)
			}
			if input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value != "test-event-id" {
				t.Errorf("expected eventId %s, got %s", "test-event-id", input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value)
			}

			// Return mock response
			return &dynamodb.UpdateItemOutput{
				Attributes: map[string]dynamodb_types.AttributeValue{
					"eventId": &dynamodb_types.AttributeValueMemberS{Value: "test-event-id"},
					"fields":  &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "attendeeEmail"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "text"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: ""},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: "email@example.com"},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated email in case of any changes"},
							},
						},
						&dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"name":        &dynamodb_types.AttributeValueMemberS{Value: "tshirtSize"},
								"type":        &dynamodb_types.AttributeValueMemberS{Value: "select"},
								"required":    &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								"default":     &dynamodb_types.AttributeValueMemberS{Value: "large"},
								"placeholder": &dynamodb_types.AttributeValueMemberS{Value: ""},
								"description": &dynamodb_types.AttributeValueMemberS{Value: "We need your updated tshirt size for the event"},
								"options": &dynamodb_types.AttributeValueMemberL{Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "small"},
									&dynamodb_types.AttributeValueMemberS{Value: "medium"},
									&dynamodb_types.AttributeValueMemberS{Value: "large"},
									&dynamodb_types.AttributeValueMemberS{Value: "XL"},
								}},
							},
						},
					}},
					"updatedAt": &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
				},
			}, nil
		},
	}

	service := &RegistrationFieldsService{}

	// Test case: successful update
	registrationFieldsUpdate := internal_types.RegistrationFieldsUpdate{
		Fields: []internal_types.RegistrationField{
			{
				Name:        "attendeeEmail",
				Type:        "text",
				Required:    true,
				Default:     "",
				Placeholder: "email@example.com",
				Description: "We need your updated email in case of any changes",
			},
			{
				Name:        "tshirtSize",
				Type:        "select",
				Required:    true,
				Default:     "large",
				Placeholder: "",
				Description: "We need your updated tshirt size for the event",
				Options:     []string{"small", "medium", "large", "XL"},
			},
		},
	}

	result, err := service.UpdateRegistrationFields(context.TODO(), mockDB, "test-event-id", registrationFieldsUpdate)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result == nil {
		t.Errorf("expected non-nil result, got nil")
	}

	// Assert the correct event ID in the result
	if result.EventId != "test-event-id" {
		t.Errorf("expected eventId 'test-event-id', got %s", result.EventId)
	}

	// Assert the fields are as expected
	expectedFields := registrationFieldsUpdate.Fields
	if !reflect.DeepEqual(result.Fields, expectedFields) {
		t.Errorf("expected fields %v, got %v", expectedFields, result.Fields)
	}

	// Test case: empty table name
	registrationFieldsTableName = ""
	_, err = service.UpdateRegistrationFields(context.TODO(), mockDB, "test-event-id", registrationFieldsUpdate)
	if err == nil || err.Error() != "ERR: registrationFieldsTableName is empty" {
		t.Errorf("expected error 'ERR: registrationFieldsTableName is empty', got %v", err)
	}
}


func TestDeleteRegistrationFields(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		DeleteItemFunc: func(ctx context.Context, input *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			// Check if the correct table is used
			if *input.TableName != registrationFieldsTableName {
				t.Errorf("expected table name %s, got %s", registrationFieldsTableName, *input.TableName)
			}
			if input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value != "test-event-id" {
				t.Errorf("expected eventId %s, got %s", "test-event-id", input.Key["eventId"].(*dynamodb_types.AttributeValueMemberS).Value)
			}

			// Return mock response for successful deletion
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}

	service := &RegistrationFieldsService{}

	// Test case: successful deletion
	err := service.DeleteRegistrationFields(context.TODO(), mockDB, "test-event-id")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Test case: deletion error handling
	mockDB.DeleteItemFunc = func(ctx context.Context, input *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
		return nil, fmt.Errorf("delete error")
	}

	err = service.DeleteRegistrationFields(context.TODO(), mockDB, "test-event-id")
	if err == nil || err.Error() != "delete error" {
		t.Errorf("expected error 'delete error', got %v", err)
	}
}

func TestDeleteRegistrationFields_Error(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		DeleteItemFunc: func(ctx context.Context, input *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, fmt.Errorf("delete error")
		},
	}

	service := &RegistrationFieldsService{}
	err := service.DeleteRegistrationFields(context.TODO(), mockDB, "event-id")
	if err == nil || err.Error() != "delete error" {
		t.Errorf("expected 'delete error', got %v", err)
	}
}

