package dynamodb_service

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var competitionWaitingRoomParticipantTableName = helpers.GetDbTableName(helpers.CompetitionWaitingRoomParticipantTablePrefix)

func init() {
	competitionWaitingRoomParticipantTableName = helpers.GetDbTableName(helpers.CompetitionWaitingRoomParticipantTablePrefix)
}

type CompetitionWaitingRoomParticipantService struct{}

func NewCompetitionWaitingRoomParticipantService() internal_types.CompetitionWaitingRoomParticipantServiceInterface {
	return &CompetitionWaitingRoomParticipantService{}
}

func (s *CompetitionWaitingRoomParticipantService) PutCompetitionWaitingRoomParticipant(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, waitingRoomParticipant internal_types.CompetitionWaitingRoomParticipantUpdate) (dynamodb.PutItemOutput, error) {
	if competitionWaitingRoomParticipantTableName == "" {
		return dynamodb.PutItemOutput{}, fmt.Errorf("ERR: competitionWaitingRoomParticipantTableName is empty: check for SST binding.")
	}

	// Validate the competition object
	if err := validate.Struct(waitingRoomParticipant); err != nil {
		return dynamodb.PutItemOutput{}, fmt.Errorf("validation failed: %w", err)
	}

	item, err := attributevalue.MarshalMap(&waitingRoomParticipant)
	if err != nil {
		return dynamodb.PutItemOutput{}, err
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(competitionWaitingRoomParticipantTableName),
	}

	log.Printf("Item before Item put waiting: %+v", input)

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("error in put item dynamo")
		return dynamodb.PutItemOutput{}, err
	}

	return *res, nil
}

func (s *CompetitionWaitingRoomParticipantService) GetCompetitionWaitingRoomParticipants(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionId string) ([]internal_types.CompetitionWaitingRoomParticipant, error) {
	// Validate input
	if competitionId == "" {
		return nil, fmt.Errorf("competitionId cannot be empty")
	}

	keyEx := expression.Key("competitionId").Equal(expression.Value(competitionId))

	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(competitionWaitingRoomParticipantTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		log.Printf("ERROR: Query failed with error: %v", err)
		return nil, fmt.Errorf("failed to query rounds: %w", err)
	}

	log.Printf("Query returned %d items", len(result.Items))

	// If no items found, return empty slice
	if len(result.Items) == 0 {
		log.Printf("No items found for competitionId: %s", competitionId)
		return []internal_types.CompetitionWaitingRoomParticipant{}, nil
	}

	var competitionWaitingRoomParticipants []internal_types.CompetitionWaitingRoomParticipant
	err = attributevalue.UnmarshalListOfMaps(result.Items, &competitionWaitingRoomParticipants)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal items: %v", err)
		return nil, fmt.Errorf("failed to unmarshal items: %v", err)
	}

	return competitionWaitingRoomParticipants, nil
}

func (s *CompetitionWaitingRoomParticipantService) DeleteCompetitionWaitingRoomParticipant(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionId, userId string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(competitionWaitingRoomParticipantTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"competitionId": &dynamodb_types.AttributeValueMemberS{Value: competitionId},
			"userId":        &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("competition round successfully deleted")
	return nil
}
