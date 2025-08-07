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

var votesTableName = helpers.GetDbTableName(helpers.VotesTablePrefix)

func init() {
	votesTableName = helpers.GetDbTableName(helpers.VotesTablePrefix)
}

type CompetitionVoteService struct{}

func NewCompetitionVoteService() internal_types.CompetitionVoteServiceInterface {
	return &CompetitionVoteService{}
}

func (s *CompetitionVoteService) PutCompetitionVote(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, vote internal_types.CompetitionVoteUpdate) (dynamodb.PutItemOutput, error) {
	if err := validate.Struct(vote); err != nil {
		return dynamodb.PutItemOutput{}, fmt.Errorf("validation failed: %w", err)
	}

	item, err := attributevalue.MarshalMap(&vote)
	if err != nil {
		return dynamodb.PutItemOutput{}, err
	}

	if votesTableName == "" {
		return dynamodb.PutItemOutput{}, fmt.Errorf("ERR: votesTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(votesTableName),
		// ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	}

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		return dynamodb.PutItemOutput{}, err
	}

	return *res, nil
}

func (s *CompetitionVoteService) GetCompetitionVotesByCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, compositePartitionKey string) ([]internal_types.CompetitionVote, error) {
	if votesTableName == "" {
		return nil, fmt.Errorf("ERR: votesTableName is empty")
	}

	keyEx := expression.Key("compositePartitionKey").Equal(expression.Value(compositePartitionKey))

	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(votesTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query rounds: %w", err)
	}

	// If no items found, return empty slice
	if len(result.Items) == 0 {
		log.Printf("No items found for compositePartitionKey: %s", compositePartitionKey)
		return []internal_types.CompetitionVote{}, nil
	}

	var competitionRoundVotes []internal_types.CompetitionVote
	err = attributevalue.UnmarshalListOfMaps(result.Items, &competitionRoundVotes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %v", err)
	}

	return competitionRoundVotes, nil
}

func (s *CompetitionVoteService) DeleteCompetitionVote(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, compositePartitionKey, userId string) error {
	if votesTableName == "" {
		return fmt.Errorf("ERR: votesTableName is empty")
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(votesTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"compositePartitionKey": &dynamodb_types.AttributeValueMemberS{Value: compositePartitionKey},
			"userId":                &dynamodb_types.AttributeValueMemberS{Value: userId},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	return nil
}
