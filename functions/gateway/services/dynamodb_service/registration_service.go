package dynamodb_service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var registrationTableName = helpers.GetDbTableName(helpers.RegistrationsTablePrefix)

func init () {
	registrationTableName = helpers.GetDbTableName(helpers.RegistrationsTablePrefix)
}


// RegistrationService is the concrete implementation of the RegistrationServiceInterface.
type RegistrationService struct{}

func NewRegistrationService() internal_types.RegistrationServiceInterface {
	return &RegistrationService{}
}

func (s *RegistrationService) InsertRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registration internal_types.RegistrationInsert, eventId, userId string) (*internal_types.Registration, error) {
    // Validate the registration object
    if err := validate.Struct(registration); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

	log.Printf("reg fields in service insert: %v", registration)

	item, err := attributevalue.MarshalMap(&registration)
	if err != nil {
		return nil, err
	}

	log.Printf("item in insert: %v", item)

	if (registrationTableName == "") {
		return nil, fmt.Errorf("ERR: registrationTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                                item,
		TableName:                           aws.String(registrationTableName),
		ConditionExpression: aws.String("attribute_not_exists(eventId) AND attribute_not_exists(userId)"),
	}

	log.Printf("input in insert reg fields: %v", input)


	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("htting error in put item dynamo")
		return nil, err
	}

	log.Printf("res form put item: %v", res)

	var insertedRegistration internal_types.Registration


	err = attributevalue.UnmarshalMap(res.Attributes, &insertedRegistration)
	if err != nil {
		return nil, err
	}

	log.Printf("inserted reg fields after creation: %v", insertedRegistration)

    // return registration, nil
	return &insertedRegistration, nil
}

func (s *RegistrationService) GetRegistrationByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Registration, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(registrationTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var registration internal_types.Registration
	err = attributevalue.UnmarshalMap(result.Item, &registration)
	if err != nil {
		return nil, err
	}

	return &registration, nil
}


func (s *RegistrationService) GetRegistrationsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.Registration, error) {
	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(registrationTableName),
		KeyConditions: map[string]dynamodb_types.Condition{
			"eventId": {
				ComparisonOperator: dynamodb_types.ComparisonOperatorEq,
				AttributeValueList: []dynamodb_types.AttributeValue{
					&dynamodb_types.AttributeValueMemberS{Value: eventId},
				},
			},
		},
	}

	// Run the query with the constructed QueryInput
	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	var registrations []internal_types.Registration
	err = attributevalue.UnmarshalListOfMaps(result.Items, &registrations)
	if err != nil {
		return nil, err
	}

	return registrations, nil
}

func (s *RegistrationService) GetRegistrationsByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Registration, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(registrationTableName),
		IndexName:              aws.String("userIdGsi"), // GSI name
		KeyConditionExpression: aws.String("userId = :userId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	result, err := dynamodbClient.Query(context.TODO(), input)
	if err != nil {
		log.Fatalf("Query GSI failed, %v", err)
	}
	log.Printf("query gsi: %v", result)

	inputScan := &dynamodb.ScanInput{
		TableName: aws.String(registrationTableName),
		IndexName: aws.String("userIdGsi"), // Scan the GSI
	}

	resultScan, err := dynamodbClient.Scan(ctx, inputScan)
	if err != nil {
		log.Fatalf("Scan GSI failed: %v", err)
	}

	log.Printf("GSI scan result: %v", resultScan.Items)

	var registrations []internal_types.Registration
	err = attributevalue.UnmarshalListOfMaps(result.Items, &registrations)
	if err != nil {
		return nil, err
	}

	return registrations, nil
}

func (s *RegistrationService) UpdateRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, registration internal_types.RegistrationUpdate) (*internal_types.Registration, error) {
	if registrationTableName == "" {
		return nil, fmt.Errorf("ERR: registrationTableName is empty")
	}

	// Build the UpdateItemInput with composite key
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(registrationTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId":  &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	// Check if responses need to be updated
	if len(registration.Responses) > 0 {
		input.ExpressionAttributeNames["#responses"] = "responses"
		responses, err := attributevalue.MarshalList(registration.Responses)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":responses"] = &dynamodb_types.AttributeValueMemberL{Value: responses}
		*input.UpdateExpression += " #responses = :responses,"
	}

	// Set the updatedAt field
	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(currentTime, 10)}
	*input.UpdateExpression += " #updatedAt = :updatedAt"

	// Execute the update
	res, err := dynamodbClient.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	// Unmarshal the updated registration
	var updatedRegistration internal_types.Registration
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedRegistration)
	if err != nil {
		return nil, err
	}

	return &updatedRegistration, nil
}


func (s *RegistrationService) DeleteRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string)  error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(registrationTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return  err
	}

	log.Printf("registration fields successfully deleted")
	return nil
}

type MockRegistrationService struct {
    InsertRegistrationFunc          func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registration internal_types.RegistrationInsert, eventId, userId string) (*internal_types.Registration, error)
    GetRegistrationByPkFunc         func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Registration, error)
    GetRegistrationsByEventIDFunc   func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.Registration, error)
    GetRegistrationsByUserIDFunc    func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Registration, error)
    UpdateRegistrationFunc          func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, registration internal_types.RegistrationUpdate) (*internal_types.Registration, error)
    DeleteRegistrationFunc          func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error
}

func (m *MockRegistrationService) InsertRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, registration internal_types.RegistrationInsert, eventId, userId string) (*internal_types.Registration, error) {
    return m.InsertRegistrationFunc(ctx, dynamodbClient, registration, eventId, userId)
}

func (m *MockRegistrationService) GetRegistrationByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Registration, error) {
    return m.GetRegistrationByPkFunc(ctx, dynamodbClient, eventId, userId)
}

func (m *MockRegistrationService) GetRegistrationsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.Registration, error) {
    return m.GetRegistrationsByEventIDFunc(ctx, dynamodbClient, eventId)
}

func (m *MockRegistrationService) GetRegistrationsByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Registration, error) {
    return m.GetRegistrationsByUserIDFunc(ctx, dynamodbClient, userId)
}

func (m *MockRegistrationService) UpdateRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, registration internal_types.RegistrationUpdate) (*internal_types.Registration, error) {
    return m.UpdateRegistrationFunc(ctx, dynamodbClient, eventId, userId, registration)
}

func (m *MockRegistrationService) DeleteRegistration(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error {
    return m.DeleteRegistrationFunc(ctx, dynamodbClient, eventId, userId)
}
