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

var purchasablesTableName = helpers.GetDbTableName(helpers.PurchasablesTablePrefix)

func init () {
	purchasablesTableName = helpers.GetDbTableName(helpers.PurchasablesTablePrefix)
}

type PurchasableService struct{}

func NewPurchasableService() internal_types.PurchasableServiceInterface {
	return &PurchasableService{}
}

func (s *PurchasableService) InsertPurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasable internal_types.PurchasableInsert) (*internal_types.Purchasable, error) {
    // Generate a new UUID if not provided
    // Validate the purchasable object
    if err := validate.Struct(purchasable); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

	item, err := attributevalue.MarshalMap(&purchasable)
	if err != nil {
		return nil, err
	}

	log.Printf("item in purchase: %v", item)
	if (purchasablesTableName == "") {
		return nil, fmt.Errorf("ERR: purchasablesTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                                item,
		TableName:                           aws.String(purchasablesTableName),
		ConditionExpression: aws.String("attribute_not_exists(eventId) AND attribute_not_exists(userId)"),
	}


	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var insertedPurchasables internal_types.Purchasable

	err = attributevalue.UnmarshalMap(res.Attributes, &insertedPurchasables)
	if err != nil {
		return nil, err
	}

	return &insertedPurchasables, nil
}


func (s *PurchasableService) GetPurchasablesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.Purchasable, error) {
	queryInput := &dynamodb.GetItemInput{
		TableName: aws.String(purchasablesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("no item found with eventId: %s", eventId)
	}

	var purchasable internal_types.Purchasable
	err = attributevalue.UnmarshalMap(result.Item, &purchasable)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %v", err)
	}

	return &purchasable, nil
}

func (s *PurchasableService) UpdatePurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasable internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {
	if purchasablesTableName == "" {
		return nil, fmt.Errorf("ERR: rsvpTableName is empty")
	}
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(purchasablesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: purchasable.EventId},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	if purchasable.PurchasableItems != nil {
		input.ExpressionAttributeNames["#purchasableItems"] = "purchasableItems"
		purchasableItems, err := attributevalue.MarshalList(purchasable.PurchasableItems)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":purchasableItems"] = &dynamodb_types.AttributeValueMemberL{Value: purchasableItems}
		*input.UpdateExpression += " #purchasableItems = :purchasableItems,"
	}
	if purchasable.RegistrationFieldsNames != nil {
		input.ExpressionAttributeNames["#registrationFieldsNames"] = "registrationFieldsNames"
		registrationFieldsNames, err := attributevalue.MarshalList(purchasable.PurchasableItems)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":registrationFieldsNames"] = &dynamodb_types.AttributeValueMemberL{Value: registrationFieldsNames}
		*input.UpdateExpression += " #registrationFieldsNames = :registrationFieldsNames,"
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
	var updatedPurchasable internal_types.Purchasable
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedPurchasable)
	if err != nil {
		return nil, err
	}

	return &updatedPurchasable, nil
}

func (s *PurchasableService) DeletePurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string)  error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(purchasablesTableName),
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

type MockPurchasableService struct {
	InsertPurchasableFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasables internal_types.PurchasableInsert) (*internal_types.Purchasable, error)
	GetPurchasablesByEventIDFunc  func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.Purchasable, error)
	UpdatePurchasableFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI,  purchasables internal_types.PurchasableUpdate) (*internal_types.Purchasable, error)
	DeletePurchasableFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error
}

// Implement the required methods

func (m *MockPurchasableService) InsertPurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasable internal_types.PurchasableInsert) (*internal_types.Purchasable, error) {
	if m.InsertPurchasableFunc != nil {
		return m.InsertPurchasableFunc(ctx, dynamodbClient, purchasable)
	}
	return nil, nil
}


func (m *MockPurchasableService) GetPurchasablesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasableID string) (*internal_types.Purchasable, error) {
	if m.GetPurchasablesByEventIDFunc != nil {
		return m.GetPurchasablesByEventIDFunc(ctx, dynamodbClient, purchasableID)
	}
	return nil, nil
}

func (m *MockPurchasableService) UpdatePurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasable internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {
	if m.UpdatePurchasableFunc != nil {
		return m.UpdatePurchasableFunc(ctx, dynamodbClient, purchasable)
	}
	return nil, nil
}

func (m *MockPurchasableService) DeletePurchasable(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error {
	if m.DeletePurchasableFunc != nil {
		return m.DeletePurchasableFunc(ctx, dynamodbClient, eventId)
	}
	return nil
}


