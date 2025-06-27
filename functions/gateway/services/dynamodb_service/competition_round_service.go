package dynamodb_service

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var competitionRoundsTableName = helpers.GetDbTableName(helpers.CompetitionRoundsTablePrefix)

func init() {
	competitionRoundsTableName = helpers.GetDbTableName(helpers.CompetitionRoundsTablePrefix)
}

type CompetitionRoundService struct{}

func NewCompetitionRoundService() internal_types.CompetitionRoundServiceInterface {
	return &CompetitionRoundService{}
}

func (s *CompetitionRoundService) PutCompetitionRounds(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, rounds *[]internal_types.CompetitionRoundUpdate) (dynamodb.BatchWriteItemOutput, error) {
	if rounds == nil || len(*rounds) < 1 {
		return dynamodb.BatchWriteItemOutput{}, fmt.Errorf("rounds cannot be empty or nil")
	}
	if competitionRoundsTableName == "" {
		log.Printf("Service ERROR: competitionRoundsTableName is empty")
		return dynamodb.BatchWriteItemOutput{}, fmt.Errorf("ERR: competitionRoundsTableName is empty")
	}

	var writeRequests []dynamodb_types.WriteRequest

	// Convert each round to a PutRequest
	for i, round := range *rounds {
		// Marshal the item
		item, err := attributevalue.MarshalMap(round)
		if err != nil {
			log.Printf("Service ERROR: Failed to marshal round %d: %v", i+1, err)
			return dynamodb.BatchWriteItemOutput{}, fmt.Errorf("failed to marshal round: %w", err)
		}
		log.Printf("Service: Successfully marshaled round %d: %+v", i+1, round)

		// Create PutRequest.
		writeRequests = append(writeRequests, dynamodb_types.WriteRequest{
			PutRequest: &dynamodb_types.PutRequest{
				Item: item,
			},
		})
	}

	// Create BatchWriteItemInput
	input := &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]dynamodb_types.WriteRequest{
			competitionRoundsTableName: writeRequests,
		},
	}

	result, err := dynamodbClient.BatchWriteItem(ctx, input)
	if err != nil {
		log.Printf("Service ERROR: BatchWriteItem failed: %v", err)
		return dynamodb.BatchWriteItemOutput{}, fmt.Errorf("failed to batch write items: %w", err)
	}

	if len(result.UnprocessedItems) > 0 {
		log.Printf("Service WARNING: Some items were not processed: %v", result.UnprocessedItems)
	} else {
		log.Printf("Service: Successfully processed all items")
	}

	log.Printf("BatchItemOUtput: %+v", result)

	return *result, nil
}

func (s *CompetitionRoundService) BatchPatchCompetitionRounds(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, updates []internal_types.CompetitionRoundUpdate, keysToUpdate []string) error {
	// Create error channel for goroutines
	errChan := make(chan error, len(updates))
	var wg sync.WaitGroup
	log.Printf("~ BatchPatchCompetitionRounds ~ keysToUpdate: %+v", keysToUpdate)
	log.Printf("~ BatchPatchCompetitionRounds ~ updates: %+v", updates)
	
	// Process each update in parallel
	for _, update := range updates {
		wg.Add(1)
		go func(update internal_types.CompetitionRoundUpdate) {
			defer wg.Done()

			input := &dynamodb.UpdateItemInput{
				TableName: aws.String(competitionRoundsTableName),
				Key: map[string]dynamodb_types.AttributeValue{
					"competitionId": &dynamodb_types.AttributeValueMemberS{Value: update.CompetitionId},
					"roundNumber":   &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(update.RoundNumber, 10)},
				},
				ExpressionAttributeNames:  make(map[string]string),
				ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
				UpdateExpression:          aws.String("SET"),
			}

			// Build update expression based on keysToUpdate
			for _, key := range keysToUpdate {
				switch key {
				case "eventId":
					if update.EventId != "" {
						input.ExpressionAttributeNames["#eventId"] = "eventId"
						input.ExpressionAttributeValues[":eventId"] = &dynamodb_types.AttributeValueMemberS{Value: update.EventId}
						*input.UpdateExpression += " #eventId = :eventId,"
					}
				case "roundName":
					if update.RoundName != "" {
						input.ExpressionAttributeNames["#roundName"] = "roundName"
						input.ExpressionAttributeValues[":roundName"] = &dynamodb_types.AttributeValueMemberS{Value: update.RoundName}
						*input.UpdateExpression += " #roundName = :roundName,"
					}
				case "competitorA":
					if update.CompetitorA != "" {
						input.ExpressionAttributeNames["#competitorA"] = "competitorA"
						input.ExpressionAttributeValues[":competitorA"] = &dynamodb_types.AttributeValueMemberS{Value: update.CompetitorA}
						*input.UpdateExpression += " #competitorA = :competitorA,"
					}
				case "competitorAScore":
					input.ExpressionAttributeNames["#competitorAScore"] = "competitorAScore"
					input.ExpressionAttributeValues[":competitorAScore"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatFloat(update.CompetitorAScore, 'f', -1, 64)}
					*input.UpdateExpression += " #competitorAScore = :competitorAScore,"
				case "competitorB":
					if update.CompetitorB != "" {
						input.ExpressionAttributeNames["#competitorB"] = "competitorB"
						input.ExpressionAttributeValues[":competitorB"] = &dynamodb_types.AttributeValueMemberS{Value: update.CompetitorB}
						*input.UpdateExpression += " #competitorB = :competitorB,"
					}
				case "competitorBScore":
					input.ExpressionAttributeNames["#competitorBScore"] = "competitorBScore"
					input.ExpressionAttributeValues[":competitorBScore"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatFloat(update.CompetitorBScore, 'f', -1, 64)}
					*input.UpdateExpression += " #competitorBScore = :competitorBScore,"
				case "matchup":
					if update.Matchup != "" {
						input.ExpressionAttributeNames["#matchup"] = "matchup"
						input.ExpressionAttributeValues[":matchup"] = &dynamodb_types.AttributeValueMemberS{Value: update.Matchup}
						*input.UpdateExpression += " #matchup = :matchup,"
					}
				case "status":
					if update.Status != "" {
						input.ExpressionAttributeNames["#status"] = "status"
						input.ExpressionAttributeValues[":status"] = &dynamodb_types.AttributeValueMemberS{Value: update.Status}
						*input.UpdateExpression += " #status = :status,"
					}
				case "isPending":
					input.ExpressionAttributeNames["#isPending"] = "isPending"
					input.ExpressionAttributeValues[":isPending"] = &dynamodb_types.AttributeValueMemberBOOL{Value: update.IsPending}
					*input.UpdateExpression += " #isPending = :isPending,"
				case "isVotingOpen":
					input.ExpressionAttributeNames["#isVotingOpen"] = "isVotingOpen"
					input.ExpressionAttributeValues[":isVotingOpen"] = &dynamodb_types.AttributeValueMemberBOOL{Value: update.IsVotingOpen}
					*input.UpdateExpression += " #isVotingOpen = :isVotingOpen,"
				case "description":
					if update.Description != "" {
						input.ExpressionAttributeNames["#description"] = "description"
						input.ExpressionAttributeValues[":description"] = &dynamodb_types.AttributeValueMemberS{Value: update.Description}
						*input.UpdateExpression += " #description = :description,"
					}
				}
			}

			// Always update updatedAt
			input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
			input.ExpressionAttributeValues[":updatedAt"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)}
			*input.UpdateExpression += " #updatedAt = :updatedAt"

			// Execute the update
			// TODO: this logging is temporary for debugging
			log.Printf("Updating round %s-%d", update.CompetitionId, update.RoundNumber)
			log.Printf("Input: %+v", input)
			_, err := dynamodbClient.UpdateItem(ctx, input)
			if err != nil {
				errChan <- fmt.Errorf("failed to update round %s-%d: %w", update.CompetitionId, update.RoundNumber, err)
				return
			}
		}(update)
	}

	// Wait for all updates to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	var errors []error
	for err := range errChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Return combined error if any occurred
	if len(errors) > 0 {
		return fmt.Errorf("batch update errors: %v", errors)
	}

	return nil
}

func (s *CompetitionRoundService) GetCompetitionRoundByPrimaryKey(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionId, roundNumber string) (*internal_types.CompetitionRound, error) {
	queryInput := &dynamodb.GetItemInput{
		TableName: aws.String(competitionRoundsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"competitionId": &dynamodb_types.AttributeValueMemberS{Value: competitionId},
			"roundNumber":   &dynamodb_types.AttributeValueMemberN{Value: roundNumber},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("no round found with PK: %s and SK: %s", competitionId, roundNumber)
	}

	var round internal_types.CompetitionRound
	err = attributevalue.UnmarshalMap(result.Item, &round)
	if err != nil {
		// Handle string to slice conversion for competitors
		var tempRound struct {
			CompetitionId      string   `dynamodbav:"competitionId"`
			RoundNumber        int64    `dynamodbav:"roundNumber"`
			EventId            string   `dynamodbav:"eventId"`
			AssociatedEventGSI string   `dynamodbav:"associatedEventGSI"`
			RoundName          string   `dynamodbav:"roundName"`
			CompetitorA        string   `dynamodbav:"competitorA"`
			CompetitorAScore   float64  `dynamodbav:"competitorAScore"`
			CompetitorB        string   `dynamodbav:"competitorB"`
			CompetitorBScore   float64  `dynamodbav:"competitorBScore"`
			Matchup            string   `dynamodbav:"matchup"`
			Status             string   `dynamodbav:"status"`
			Competitors        []string `dynamodbav:"competitors"` // This is the key difference
			IsPending          bool     `dynamodbav:"isPending"`
			IsVotingOpen       bool     `dynamodbav:"isVotingOpen"`
			CreatedAt          int64    `dynamodbav:"createdAt"`
			UpdatedAt          int64    `dynamodbav:"updatedAt"`
		}

		err = attributevalue.UnmarshalMap(result.Item, &tempRound)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal item: %w", err)
		}

		round.CompetitionId = tempRound.CompetitionId
		round.RoundNumber = tempRound.RoundNumber
		round.EventId = tempRound.EventId
		round.RoundName = tempRound.RoundName
		round.CompetitorA = tempRound.CompetitorA
		round.CompetitorAScore = tempRound.CompetitorAScore
		round.CompetitorB = tempRound.CompetitorB
		round.CompetitorBScore = tempRound.CompetitorBScore
		round.Matchup = tempRound.Matchup
		round.Status = tempRound.Status
		round.IsPending = tempRound.IsPending
		round.IsVotingOpen = tempRound.IsVotingOpen
		round.CreatedAt = tempRound.CreatedAt
		round.UpdatedAt = tempRound.UpdatedAt
	}

	return &round, nil
}

func (s *CompetitionRoundService) GetCompetitionRoundsByEventId(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*[]internal_types.CompetitionRound, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(competitionRoundsTableName),
		IndexName:              aws.String("belongsToEvent"), // GSI name
		KeyConditionExpression: aws.String("eventId = :eventId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":eventId": &dynamodb_types.AttributeValueMemberS{Value: eventId},
		},
	}

	result, err := dynamodbClient.Query(context.TODO(), input)
	if err != nil {
		log.Fatalf("Query to belongsToEvent GSI failed, %v", err)
	}

	var competitionRounds []internal_types.CompetitionRound
	err = attributevalue.UnmarshalListOfMaps(result.Items, &competitionRounds)
	if err != nil {
		return nil, err
	}

	return &competitionRounds, nil
}

func (s *CompetitionRoundService) GetCompetitionRounds(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionId string) (*[]internal_types.CompetitionRound, error) {

	// Validate input
	if competitionId == "" {
		log.Printf("Service ERROR: Empty competitionId provided")
		return nil, fmt.Errorf("competitionId cannot be empty")
	}

	keyEx := expression.Key("competitionId").Equal(expression.Value(competitionId))

	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		log.Printf("Service ERROR: Failed to build expression: %v", err)
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}
	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(competitionRoundsTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	// Perform the actual query
	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		log.Printf("ERROR: Query failed with error: %v", err)
		return nil, fmt.Errorf("failed to query rounds: %w", err)
	}

	// If no items found, return empty slice
	if len(result.Items) == 0 {
		log.Printf("No items found for competitionId: %s", competitionId)
		return &[]internal_types.CompetitionRound{}, nil
	}

	var rounds []internal_types.CompetitionRound
	err = attributevalue.UnmarshalListOfMaps(result.Items, &rounds)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal items: %v", err)
		return nil, fmt.Errorf("failed to unmarshal items: %v", err)
	}

	return &rounds, nil
}

func (s *CompetitionRoundService) DeleteCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionId, roundNumber string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(competitionRoundsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"competitionId": &dynamodb_types.AttributeValueMemberS{Value: competitionId},
			"roundNumber":   &dynamodb_types.AttributeValueMemberN{Value: roundNumber},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("competition round successfully deleted")
	return nil
}

// TODO: these are pretty incorrect and must be corrected

// Mock service for testing
// type MockCompetitionRoundService struct {
// 	InsertCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, round internal_types.CompetitionRoundInsert) (*internal_types.CompetitionRound, error)
// 	GetCompetitionRoundByPkFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) (*internal_types.CompetitionRound, error)
// 	GetCompetitionRoundsByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, eventId string) ([]internal_types.CompetitionRound, error)
// 	UpdateCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string, round internal_types.CompetitionRoundUpdate) (*internal_types.CompetitionRound, error)
// 	DeleteCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) error
// }
//
// func (m *MockCompetitionRoundService) InsertCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, round internal_types.CompetitionRoundInsert) (*internal_types.CompetitionRound, error) {
// 	return m.InsertCompetitionRoundFunc(ctx, dynamodbClient, round)
// }
//
// func (m *MockCompetitionRoundService) GetCompetitionRoundByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) (*internal_types.CompetitionRound, error) {
// 	return m.GetCompetitionRoundByPkFunc(ctx, dynamodbClient, pk, sk)
// }
//
// func (m *MockCompetitionRoundService) GetCompetitionRoundsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, eventId string) ([]internal_types.CompetitionRound, error) {
// 	return m.GetCompetitionRoundsByEventIDFunc(ctx, dynamodbClient, pk, eventId)
// }
//
// func (m *MockCompetitionRoundService) UpdateCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string, round internal_types.CompetitionRoundUpdate) (*internal_types.CompetitionRound, error) {
// 	return m.UpdateCompetitionRoundFunc(ctx, dynamodbClient, pk, sk, round)
// }
//
// func (m *MockCompetitionRoundService) DeleteCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) error {
// 	return m.DeleteCompetitionRoundFunc(ctx, dynamodbClient, pk, sk)
// }
//
