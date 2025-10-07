package dynamodb_service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-playground/validator"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var registrationFieldsTableName = helpers.GetDbTableName(constants.RegistrationFieldsTablePrefix)

func init() {
	registrationFieldsTableName = helpers.GetDbTableName(constants.RegistrationFieldsTablePrefix)
}

// Validator instance for struct validation
var validate *validator.Validate = validator.New()

// RegistrationFieldsServiceInterface defines the methods required for the user service.
type RegistrationFieldsServiceInterface interface {
	InsertRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registrationFields internal_types.RegistrationFieldsInsert, eventId string) (*internal_types.RegistrationFields, error)
	GetRegistrationFieldsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.RegistrationFields, error)
	UpdateRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error)
	DeleteRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error
}

// RegistrationFieldsService is the concrete implementation of the RegistrationFieldsServiceInterface.
type RegistrationFieldsService struct{}

func NewRegistrationFieldsService() RegistrationFieldsServiceInterface {
	return &RegistrationFieldsService{}
}

func (s *RegistrationFieldsService) InsertRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registrationFields internal_types.RegistrationFieldsInsert, eventId string) (*internal_types.RegistrationFields, error) {
	// Validate the registrationFields object
	if err := validate.Struct(registrationFields); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	item, err := attributevalue.MarshalMap(&registrationFields)
	if err != nil {
		return nil, err
	}

	if registrationFieldsTableName == "" {
		return nil, fmt.Errorf("ERR: registrationFieldsTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                item,
		TableName:           aws.String(registrationFieldsTableName),
		ConditionExpression: aws.String("attribute_not_exists(eventId)"),
	}

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("hitting error in put item dynamo")
		return nil, err
	}

	var insertedRegistrationFields internal_types.RegistrationFields

	err = attributevalue.UnmarshalMap(res.Attributes, &insertedRegistrationFields)
	if err != nil {
		return nil, err
	}

	// return registrationFields, nil
	return &insertedRegistrationFields, nil
}

func (s *RegistrationFieldsService) GetRegistrationFieldsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.RegistrationFields, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(registrationFieldsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var registrationFields internal_types.RegistrationFields
	err = attributevalue.UnmarshalMap(result.Item, &registrationFields)
	if err != nil {
		return nil, err
	}

	return &registrationFields, nil
}

func (s *RegistrationFieldsService) UpdateRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error) {
	if registrationFieldsTableName == "" {
		return nil, fmt.Errorf("ERR: registrationFieldsTableName is empty")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(registrationFieldsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	if registrationFields.Fields != nil {
		input.ExpressionAttributeNames["#fields"] = "fields"
		fields, err := attributevalue.MarshalList(registrationFields.Fields)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":fields"] = &dynamodb_types.AttributeValueMemberL{Value: fields}
		*input.UpdateExpression += " #fields = :fields,"
	}

	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatFloat(float64(currentTime), 'f', -1, 64)}
	*input.UpdateExpression += " #updatedAt = :updatedAt"

	res, err := dynamodbClient.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var updatedRegistrationFields internal_types.RegistrationFields

	err = attributevalue.UnmarshalMap(res.Attributes, &updatedRegistrationFields)
	if err != nil {
		return nil, err
	}

	// return registrationFields, nil
	return &updatedRegistrationFields, nil
}

func (s *RegistrationFieldsService) DeleteRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(registrationFieldsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

type MockRegistrationFieldsService struct {
	InsertRegistrationFieldsFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registrationFields internal_types.RegistrationFieldsInsert) (*internal_types.RegistrationFields, error)
	GetRegistrationFieldsByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.RegistrationFields, error)
	UpdateRegistrationFieldsFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error)
	DeleteRegistrationFieldsFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error
}

func (m *MockRegistrationFieldsService) InsertRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registrationFields internal_types.RegistrationFieldsInsert, eventId string) (*internal_types.RegistrationFields, error) {
	return m.InsertRegistrationFieldsFunc(ctx, dynamodbClient, registrationFields)
}

func (m *MockRegistrationFieldsService) GetRegistrationFieldsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.RegistrationFields, error) {
	return m.GetRegistrationFieldsByEventIDFunc(ctx, dynamodbClient, eventId)
}

func (m *MockRegistrationFieldsService) UpdateRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, registrationFields internal_types.RegistrationFieldsUpdate) (*internal_types.RegistrationFields, error) {
	return m.UpdateRegistrationFieldsFunc(ctx, dynamodbClient, eventId, registrationFields)
}

func (m *MockRegistrationFieldsService) DeleteRegistrationFields(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error {
	return m.DeleteRegistrationFieldsFunc(ctx, dynamodbClient, eventId)
}
