// TODO: change all fmt to log printout in new rds handlers and services
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

var rsvpTableName = helpers.GetDbTableName(helpers.RsvpsTablePrefix)

func init () {
	rsvpTableName = helpers.GetDbTableName(helpers.RsvpsTablePrefix)
}

type EventRsvpService struct{}

func NewEventRsvpService() internal_types.EventRsvpServiceInterface {
	return &EventRsvpService{}
}

func (s *EventRsvpService) InsertEventRsvp(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error) {
    // Validate the eventRsvp object
    if err := validate.Struct(eventRsvp); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

	item, err := attributevalue.MarshalMap(&eventRsvp)
	if err != nil {
		return nil, err
	}

	if (rsvpTableName == "") {
		return nil, fmt.Errorf("ERR: rsvpTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                                item,
		TableName:                           aws.String(registrationTableName),
		ConditionExpression: aws.String("attribute_not_exists(eventId) AND attribute_not_exists(userId)"),
	}


	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("htting error in put item dynamo")
		return nil, err
	}

	var insertedRegistration internal_types.EventRsvp

	err = attributevalue.UnmarshalMap(res.Attributes, &insertedRegistration)
	if err != nil {
		return nil, err
	}

    // return registration, nil
	return &insertedRegistration, nil
}


func (s *EventRsvpService) GetEventRsvpByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.EventRsvp, error) {
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

	var eventRsvp internal_types.EventRsvp
	err = attributevalue.UnmarshalMap(result.Item, &eventRsvp)
	if err != nil {
		return nil, err
	}

	return &eventRsvp, nil
}

func (s *EventRsvpService) GetEventRsvpsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.EventRsvp, error) {
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

	var eventRsvps []internal_types.EventRsvp
	err = attributevalue.UnmarshalListOfMaps(result.Items, &eventRsvps)
	if err != nil {
		return nil, err
	}

	return eventRsvps, nil
}

func (s *EventRsvpService) GetEventRsvpsByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.EventRsvp, error) {
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

	var eventRsvps []internal_types.EventRsvp
	err = attributevalue.UnmarshalListOfMaps(result.Items, &eventRsvps)
	if err != nil {
		return nil, err
	}

	return eventRsvps, nil
}

func (s *EventRsvpService) UpdateEventRsvp(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {
	if rsvpTableName == "" {
		return nil, fmt.Errorf("ERR: rsvpTableName is empty")
	}
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(rsvpTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	if eventRsvp.EventSourceID != "" {
		input.ExpressionAttributeNames["#eventSourceId"] = "eventSourceId"
		input.ExpressionAttributeValues[":eventSourceId"] = &dynamodb_types.AttributeValueMemberS{Value: eventRsvp.EventSourceID}
		*input.UpdateExpression += " #eventSourceId = :eventSourceId,"
	}

	if eventRsvp.EventSourceType != "" {
		input.ExpressionAttributeNames["#eventSourceType"] = "eventSourceType"
		input.ExpressionAttributeValues[":eventSourceType"] = &dynamodb_types.AttributeValueMemberS{Value: eventRsvp.EventSourceType}
		*input.UpdateExpression += " #eventSourceType = :eventSourceType,"
	}

	if eventRsvp.Status != "" {
		input.ExpressionAttributeNames["#status"] = "status"
		input.ExpressionAttributeValues[":status"] = &dynamodb_types.AttributeValueMemberS{Value: eventRsvp.Status}
		*input.UpdateExpression += " #status = :status,"
	}

	// Set the updatedAt field
	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(currentTime, 10)}
	*input.UpdateExpression += "#updatedAt = :updatedAt"

	// Execute the update
	res, err := dynamodbClient.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	// Unmarshal the updated registration
	var updatedEventRsvp internal_types.EventRsvp
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedEventRsvp)
	if err != nil {
		return nil, err
	}

	return &updatedEventRsvp, nil
}

func (s *EventRsvpService) DeleteEventRsvp(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string)  error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(registrationTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return  err
	}

	log.Printf("registration fields successfully deleted")
	return nil
}

type MockEventRsvpService struct {
	InsertEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert, eventId, userId string) (*internal_types.EventRsvp, error)
	GetEventRsvpByPkFunc func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string) (*internal_types.EventRsvp, error)
	GetEventRsvpsByUserIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) // New function
	GetEventRsvpsByEventIDFunc    func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) // New function
	UpdateEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error)
	DeleteEventRsvpFunc  func(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string)  error
}

func (m *MockEventRsvpService) InsertEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventRsvp internal_types.EventRsvpInsert, eventId, userId string) (*internal_types.EventRsvp, error) {
	return m.InsertEventRsvpFunc(ctx, rdsClient, eventRsvp, eventId, userId)
}

func (m *MockEventRsvpService) GetEventRsvpByPk(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string) (*internal_types.EventRsvp, error) {
	return m.GetEventRsvpByPkFunc(ctx, rdsClient, eventId, userId)
}

func (m *MockEventRsvpService) UpdateEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {
	return m.UpdateEventRsvpFunc(ctx, rdsClient, eventId, userId, eventRsvp)
}

func (m *MockEventRsvpService) DeleteEventRsvp(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventId, userId string)  error {
	return m.DeleteEventRsvpFunc(ctx, rdsClient, eventId, userId)
}

func (m *MockEventRsvpService) GetEventRsvpsByUserID(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) {
	return m.GetEventRsvpsByUserIDFunc(ctx, rdsClient, userID)
}

func (m *MockEventRsvpService) GetEventRsvpsByEventID(ctx context.Context, rdsClient internal_types.RDSDataAPI, eventID string) ([]internal_types.EventRsvp, error) {
	return m.GetEventRsvpsByEventIDFunc(ctx, rdsClient, eventID)
}
