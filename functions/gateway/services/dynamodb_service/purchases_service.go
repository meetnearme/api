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

var purchasesTableName = helpers.GetDbTableName(helpers.PurchasesTablePrefix)

func init () {
	purchasesTableName = helpers.GetDbTableName(helpers.PurchasesTablePrefix)
}

type PurchaseService struct{}

func NewPurchaseService() internal_types.PurchaseServiceInterface {
	return &PurchaseService{}
}

func (s *PurchaseService) InsertPurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchase internal_types.PurchaseInsert) (*internal_types.Purchase, error) {
    // Validate the purchase object
    if err := validate.Struct(purchase); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

	item, err := attributevalue.MarshalMap(&purchase)
	if err != nil {
		return nil, err
	}

	if (purchasesTableName == "") {
		return nil, fmt.Errorf("ERR: purchasesTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                                item,
		TableName:                           aws.String(purchasesTableName),
		ConditionExpression: aws.String("attribute_not_exists(eventId) AND attribute_not_exists(userId)"),
	}

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("htting error in put item dynamo")
		return nil, err
	}

	var insertedPurchase internal_types.Purchase

	err = attributevalue.UnmarshalMap(res.Attributes, &insertedPurchase)
	if err != nil {
		return nil, err
	}

    // return purchase, nil
	return &insertedPurchase, nil
}


func (s *PurchaseService) GetPurchaseByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Purchase, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var purchase internal_types.Purchase
	err = attributevalue.UnmarshalMap(result.Item, &purchase)
	if err != nil {
		return nil, err
	}

	return &purchase, nil
}

func (s *PurchaseService) GetPurchasesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.Purchase, error) {
	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(purchasesTableName),
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

	var purchases []internal_types.Purchase
	err = attributevalue.UnmarshalListOfMaps(result.Items, &purchases)
	if err != nil {
		return nil, err
	}

	return purchases, nil
}

func (s *PurchaseService) GetPurchasesByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Purchase, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(purchasesTableName),
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
		TableName: aws.String(purchasesTableName),
		IndexName: aws.String("userIdGsi"), // Scan the GSI
	}

	resultScan, err := dynamodbClient.Scan(ctx, inputScan)
	if err != nil {
		log.Fatalf("Scan GSI failed: %v", err)
	}

	log.Printf("GSI scan result: %v", resultScan.Items)

	var purchases []internal_types.Purchase
	err = attributevalue.UnmarshalListOfMaps(result.Items, &purchases)
	if err != nil {
		return nil, err
	}

	return purchases, nil
}

func (s *PurchaseService) UpdatePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
	if purchasesTableName == "" {
		return nil, fmt.Errorf("ERR: purchasesTableName is empty")
	}
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	if purchase.Status != "" {
		input.ExpressionAttributeNames["#status"] = "status"
		input.ExpressionAttributeValues[":status"] = &dynamodb_types.AttributeValueMemberS{Value: purchase.Status}
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

	// Unmarshal the updated purchase
	var updatedPurchase internal_types.Purchase
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedPurchase)
	if err != nil {
		return nil, err
	}

	return &updatedPurchase, nil
}

func (s *PurchaseService) DeletePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string)  error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return  err
	}

	log.Printf("purchase fields successfully deleted")
	return nil
}

type MockPurchaseService struct {
	InsertPurchaseFunc  func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchase internal_types.PurchaseInsert) (*internal_types.Purchase, error)
	GetPurchaseByPkFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Purchase, error)
	GetPurchasesByUserIDFunc    func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userID string) ([]internal_types.Purchase, error) // New function
	GetPurchasesByEventIDFunc    func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventID string) ([]internal_types.Purchase, error) // New function
	UpdatePurchaseFunc  func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error)
	DeletePurchaseFunc  func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string)  error
}

func (m *MockPurchaseService) InsertPurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchase internal_types.PurchaseInsert) (*internal_types.Purchase, error) {
	return m.InsertPurchaseFunc(ctx, dynamodbClient, purchase)
}

func (m *MockPurchaseService) GetPurchaseByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.Purchase, error) {
	return m.GetPurchaseByPkFunc(ctx, dynamodbClient, eventId, userId)
}

func (m *MockPurchaseService) UpdatePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
	return m.UpdatePurchaseFunc(ctx, dynamodbClient, eventId, userId, purchase)
}

func (m *MockPurchaseService) DeletePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string)  error {
	return m.DeletePurchaseFunc(ctx, dynamodbClient, eventId, userId)
}

func (m *MockPurchaseService) GetPurchasesByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userID string) ([]internal_types.Purchase, error) {
	return m.GetPurchasesByUserIDFunc(ctx, dynamodbClient, userID)
}

func (m *MockPurchaseService) GetPurchasesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventID string) ([]internal_types.Purchase, error) {
	return m.GetPurchasesByEventIDFunc(ctx, dynamodbClient, eventID)
}
