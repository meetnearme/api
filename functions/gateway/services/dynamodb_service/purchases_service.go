// TODO: change all fmt to log printout in new rds handlers and services
package dynamodb_service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var purchasesTableName = helpers.GetDbTableName(helpers.PurchasesTablePrefix)

func init() {
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

	if purchase.CreatedAt == 0 {
		purchase.CreatedAt = time.Now().Unix()
	}

	item, err := attributevalue.MarshalMap(&purchase)
	if err != nil {
		return nil, err
	}

	if purchasesTableName == "" {
		return nil, fmt.Errorf("ERR: purchasesTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:                item,
		TableName:           aws.String(purchasesTableName),
		ConditionExpression: aws.String("attribute_not_exists(compositeKey)"),
	}

	_, err = dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("error inserting item in database")
		return nil, err
	}

	var insertedPurchase internal_types.Purchase
	err = attributevalue.UnmarshalMap(item, &insertedPurchase)
	if err != nil {
		return nil, err
	}

	return &insertedPurchase, nil
}

func (s *PurchaseService) GetPurchaseByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"compositeKey": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("%s_%s_%s", eventId, userId, createdAt)},
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

func (s *PurchaseService) GetPurchasesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, limit int32, startKey string) ([]internal_types.PurchaseDangerous, map[string]dynamodb_types.AttributeValue, error) {
	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(purchasesTableName),
		IndexName: aws.String("eventIdIndex"), // Use the eventIdIndex GSI
		Limit:     aws.Int32(limit),
		KeyConditions: map[string]dynamodb_types.Condition{
			"eventId": {
				ComparisonOperator: dynamodb_types.ComparisonOperatorEq,
				AttributeValueList: []dynamodb_types.AttributeValue{
					&dynamodb_types.AttributeValueMemberS{Value: eventId},
				},
			},
		},
	}

	// If startKey is provided, use it for pagination
	if startKey != "" {
		// Extract createdAtString from the composite key (value after second '_')
		parts := strings.Split(startKey, "_")
		if len(parts) != 3 {
			return nil, nil, fmt.Errorf("invalid startKey format")
		}
		createdAtString := parts[2]

		queryInput.ExclusiveStartKey = map[string]dynamodb_types.AttributeValue{
			"compositeKey":    &dynamodb_types.AttributeValueMemberS{Value: startKey},
			"eventId":         &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"createdAtString": &dynamodb_types.AttributeValueMemberS{Value: createdAtString},
		}
	}

	// Run the query with the constructed QueryInput
	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		return nil, nil, err
	}

	var purchases []internal_types.PurchaseDangerous
	err = attributevalue.UnmarshalListOfMaps(result.Items, &purchases)
	if err != nil {
		return nil, nil, err
	}

	userIds := []string{}
	for _, purchase := range purchases {
		if helpers.ArrFindFirst(userIds, []string{purchase.UserID}) == "" {
			userIds = append(userIds, purchase.UserID)
		}
	}

	users, err := helpers.SearchUsersByIDs(userIds, true)
	if err != nil {
		return nil, nil, err
	}

	userMap := map[string]types.UserSearchResultDangerous{}
	for _, user := range users {
		userMap[user.UserID] = types.UserSearchResultDangerous{
			UserID:      user.UserID,
			DisplayName: user.DisplayName,
			Email:       user.Email,
		}
	}

	for i, purchase := range purchases {
		purchases[i].UserID = userMap[purchase.UserID].UserID
		purchases[i].UserEmail = userMap[purchase.UserID].Email
		purchases[i].UserDisplayName = userMap[purchase.UserID].DisplayName
	}

	return purchases, result.LastEvaluatedKey, nil
}

func (s *PurchaseService) GetPurchasesByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string, limit int32, startKey string) ([]internal_types.Purchase, map[string]dynamodb_types.AttributeValue, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(purchasesTableName),
		IndexName:              aws.String("userIdIndex"),
		KeyConditionExpression: aws.String("userId = :userId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":userId": &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
		Limit: aws.Int32(limit),
	}

	// If startKey is provided, use it for pagination
	if startKey != "" {
		// Extract createdAtString from the composite key (value after second '_')
		parts := strings.Split(startKey, "_")
		if len(parts) != 3 {
			return nil, nil, fmt.Errorf("invalid startKey format")
		}
		createdAtString := parts[2]

		input.ExclusiveStartKey = map[string]dynamodb_types.AttributeValue{
			"compositeKey":    &dynamodb_types.AttributeValueMemberS{Value: startKey},
			"userId":          &dynamodb_types.AttributeValueMemberS{Value: userId},
			"createdAtString": &dynamodb_types.AttributeValueMemberS{Value: createdAtString},
		}
	}

	result, err := dynamodbClient.Query(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	var purchases []internal_types.Purchase
	err = attributevalue.UnmarshalListOfMaps(result.Items, &purchases)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	// Return the LastEvaluatedKey which can be used as the ExclusiveStartKey for the next query
	return purchases, result.LastEvaluatedKey, nil
}

func (s *PurchaseService) UpdatePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAtString string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
	if purchasesTableName == "" {
		return nil, fmt.Errorf("ERR: purchasesTableName is empty")
	}
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"compositeKey": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("%s_%s_%s", eventId, userId, createdAtString)},
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

	if purchase.StripeTransactionId != "" {
		input.ExpressionAttributeNames["#stripeTransactionId"] = "stripeTransactionId"
		input.ExpressionAttributeValues[":stripeTransactionId"] = &dynamodb_types.AttributeValueMemberS{Value: purchase.StripeTransactionId}
		*input.UpdateExpression += " #stripeTransactionId = :stripeTransactionId,"
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

func (s *PurchaseService) DeletePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(purchasesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
			"userId":  &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("purchase fields successfully deleted")
	return nil
}

func (s *PurchaseService) HasPurchaseForEvent(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, childEventId, parentEventId, userId string) (bool, error) {
	selectInput := &dynamodb.ExecuteStatementInput{
		Statement: aws.String(fmt.Sprintf(
			`SELECT * FROM "%s"
             WHERE begins_with(compositeKey, '%s_%s')
             OR begins_with(compositeKey, '%s_%s')`,
			purchasesTableName, // Note: changed from purchasablesTableName
			childEventId, userId,
			parentEventId, userId,
		)),
	}

	result, err := dynamodbClient.ExecuteStatement(ctx, selectInput)
	if err != nil {
		return false, fmt.Errorf("query failed: %w", err)
	}

	log.Printf("result: %+v", result)

	var purchases []internal_types.Purchase
	err = attributevalue.UnmarshalListOfMaps(result.Items, &purchases)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	return len(purchases) > 0, nil
}

type MockPurchaseService struct {
	InsertPurchaseFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchase internal_types.PurchaseInsert) (*internal_types.Purchase, error)
	GetPurchaseByPkFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error)
	GetPurchasesByUserIDFunc  func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userID string, limit int32, startKey string) ([]internal_types.Purchase, map[string]dynamodb_types.AttributeValue, error)
	GetPurchasesByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventID string, limit int32, startKey string) ([]internal_types.PurchaseDangerous, map[string]dynamodb_types.AttributeValue, error)
	UpdatePurchaseFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAtString string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error)
	DeletePurchaseFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error
}

func (m *MockPurchaseService) InsertPurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchase internal_types.PurchaseInsert) (*internal_types.Purchase, error) {
	return m.InsertPurchaseFunc(ctx, dynamodbClient, purchase)
}

func (m *MockPurchaseService) GetPurchaseByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAt string) (*internal_types.Purchase, error) {
	return m.GetPurchaseByPkFunc(ctx, dynamodbClient, eventId, userId, createdAt)
}

func (m *MockPurchaseService) UpdatePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId, createdAtString string, purchase internal_types.PurchaseUpdate) (*internal_types.Purchase, error) {
	return m.UpdatePurchaseFunc(ctx, dynamodbClient, eventId, userId, createdAtString, purchase)
}

func (m *MockPurchaseService) DeletePurchase(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error {
	return m.DeletePurchaseFunc(ctx, dynamodbClient, eventId, userId)
}

func (m *MockPurchaseService) GetPurchasesByUserID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userID string, limit int32, startKey string) ([]internal_types.Purchase, map[string]dynamodb_types.AttributeValue, error) {
	return m.GetPurchasesByUserIDFunc(ctx, dynamodbClient, userID, limit, startKey)
}

func (m *MockPurchaseService) GetPurchasesByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventID string, limit int32, startKey string) ([]internal_types.PurchaseDangerous, map[string]dynamodb_types.AttributeValue, error) {
	return m.GetPurchasesByEventIDFunc(ctx, dynamodbClient, eventID, limit, startKey)
}
