package dynamodb_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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

func (s *CompetitionRoundService) InsertCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, round internal_types.CompetitionRoundInsert) (*internal_types.CompetitionRound, error) {
	log.Printf("DEBUG: Starting InsertCompetitionRound with table name: %s", competitionRoundsTableName)

	if competitionRoundsTableName == "" {
		log.Printf("ERROR: competitionRoundsTableName is empty")
		return nil, fmt.Errorf("ERR: competitionRoundsTableName is empty")
	}

	// Set PK and SK
	round.PK = fmt.Sprintf("OWNER_%s", round.OwnerId)
	round.SK = fmt.Sprintf("COMPETITION_%s_ROUND_%s", round.EventId, strconv.Itoa(int(round.RoundNumber)))
	log.Printf("DEBUG: Generated PK: %s, SK: %s", round.PK, round.SK)

	// Set timestamps if not provided
	if round.CreatedAt == 0 {
		round.CreatedAt = time.Now().Unix()
	}
	if round.UpdatedAt == 0 {
		round.UpdatedAt = round.CreatedAt
	}
	log.Printf("DEBUG: Set timestamps - CreatedAt: %d, UpdatedAt: %d", round.CreatedAt, round.UpdatedAt)

	// Generate matchup
	round.Matchup = formatMatchup(round.CompetitorA, round.CompetitorB)
	log.Printf("DEBUG: Generated matchup: %s", round.Matchup)

	item, err := attributevalue.MarshalMap(&round)
	if err != nil {
		log.Printf("ERROR: Failed to marshal round: %v", err)
		return nil, fmt.Errorf("failed to marshal round: %w", err)
	}
	log.Printf("DEBUG: Marshaled item: %+v", item)

	input := &dynamodb.PutItemInput{
		Item:         item,
		TableName:    aws.String(competitionRoundsTableName),
		ReturnValues: dynamodb_types.ReturnValueAllOld,
	}
	log.Printf("DEBUG: DynamoDB PutItem input: %+v", input)

	result, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Printf("ERROR: Failed to put item in DynamoDB: %v", err)
		return nil, fmt.Errorf("failed to put item: %w", err)
	}
	log.Printf("DEBUG: DynamoDB PutItem result: %+v", result)

	var insertedRound internal_types.CompetitionRound
	err = attributevalue.UnmarshalMap(result.Attributes, &insertedRound)
	if err != nil {
		log.Printf("ERROR: Failed to unmarshal result attributes: %v", err)
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &insertedRound, err
}

func formatDynamoDBInput(input *dynamodb.GetItemInput) string {
	if input == nil {
		return "nil input"
	}

	// Extract values from AttributeValueMemberS
	pk := ""
	sk := ""
	if pkAttr, ok := input.Key["PK"].(*dynamodb_types.AttributeValueMemberS); ok {
		pk = pkAttr.Value
	}
	if skAttr, ok := input.Key["SK"].(*dynamodb_types.AttributeValueMemberS); ok {
		sk = skAttr.Value
	}

	return fmt.Sprintf(
		"GetItemInput{\n"+
			"  TableName: %s\n"+
			"  Key: {\n"+
			"    PK: %s\n"+
			"    SK: %s\n"+
			"  }\n"+
			"}",
		*input.TableName,
		pk,
		sk,
	)
}
func formatDynamoDBQueryInput(input *dynamodb.QueryInput) string {
	if input == nil {
		return "nil QueryInput"
	}

	var details []string
	details = append(details, fmt.Sprintf("TableName: %s", *input.TableName))

	// Format KeyConditionExpression
	if input.KeyConditionExpression != nil {
		details = append(details, fmt.Sprintf("KeyConditionExpression: %s", *input.KeyConditionExpression))
	}

	// Format ExpressionAttributeValues
	if len(input.ExpressionAttributeValues) > 0 {
		details = append(details, "ExpressionAttributeValues: {")
		for k, v := range input.ExpressionAttributeValues {
			switch attr := v.(type) {
			case *dynamodb_types.AttributeValueMemberS:
				details = append(details, fmt.Sprintf("    %s: (String) %s", k, attr.Value))
			case *dynamodb_types.AttributeValueMemberN:
				details = append(details, fmt.Sprintf("    %s: (Number) %s", k, attr.Value))
			default:
				details = append(details, fmt.Sprintf("    %s: (Unknown Type) %v", k, v))
			}
		}
		details = append(details, "}")
	}

	// Format ExpressionAttributeNames
	if len(input.ExpressionAttributeNames) > 0 {
		details = append(details, "ExpressionAttributeNames: {")
		for k, v := range input.ExpressionAttributeNames {
			if v != "" {
				details = append(details, fmt.Sprintf("    %s: %s", k, &v))
			}
		}
		details = append(details, "}")
	}

	// Format FilterExpression
	if input.FilterExpression != nil {
		details = append(details, fmt.Sprintf("FilterExpression: %s", *input.FilterExpression))
	}

	// Format IndexName
	if input.IndexName != nil {
		details = append(details, fmt.Sprintf("IndexName: %s", *input.IndexName))
	}

	return "QueryInput{\n  " + strings.Join(details, "\n  ") + "\n}"
}

func (s *CompetitionRoundService) GetCompetitionRoundByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, competitionId, roundNumber string) (*internal_types.CompetitionRound, error) {
	queryInput := &dynamodb.GetItemInput{
		TableName: aws.String(competitionRoundsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("%s", pk)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("COMPETITION_%s_ROUND_%s", competitionId, roundNumber)},
		},
	}

	log.Printf("Query INput round: \n%s", formatDynamoDBInput(queryInput))

	log.Printf("about to call")

	result, err := dynamodbClient.GetItem(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	log.Printf("Result in round service: %v", result)

	if result.Item == nil {
		sk := fmt.Sprintf("COMPETITION_%s_ROUND_%s", competitionId, roundNumber)
		return nil, fmt.Errorf("no round found with PK: %s and SK: %s", pk, sk)
	}

	var round internal_types.CompetitionRound
	err = attributevalue.UnmarshalMap(result.Item, &round)
	if err != nil {
		// Handle string to slice conversion for competitors
		var tempRound struct {
			PK               string `dynamodbav:"PK"`
			SK               string `dynamodbav:"SK"`
			OwnerId          string `dynamodbav:"ownerId"`
			EventId          string `dynamodbav:"eventId"`
			RoundName        string `dynamodbav:"roundName"`
			RoundNumber      int64  `dynamodbav:"roundNumber"`
			CompetitorA      string `dynamodbav:"competitorA"`
			CompetitorAScore int64  `dynamodbav:"competitorAScore"`
			CompetitorB      string `dynamodbav:"competitorB"`
			CompetitorBScore int64  `dynamodbav:"competitorBScore"`
			Matchup          string `dynamodbav:"matchup"`
			Status           string `dynamodbav:"status"`
			Competitors      string `dynamodbav:"competitors"` // This is the key difference
			IsPending        string `dynamodbav:"isPending"`
			IsVotingOpen     string `dynamodbav:"isVotingOpen"`
			CreatedAt        int64  `dynamodbav:"createdAt"`
			UpdatedAt        int64  `dynamodbav:"updatedAt"`
		}

		err = attributevalue.UnmarshalMap(result.Item, &tempRound)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal item: %w", err)
		}

		round.PK = tempRound.PK
		round.SK = tempRound.SK
		round.OwnerId = tempRound.OwnerId
		round.EventId = tempRound.EventId
		round.RoundName = tempRound.RoundName
		round.RoundNumber = tempRound.RoundNumber
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

		// Parse competitors JSON string into slice
		if tempRound.Competitors != "" {
			err = json.Unmarshal([]byte(tempRound.Competitors), &round.Competitors)
			if err != nil {
				return nil, fmt.Errorf("failed to parse competitors: %w", err)
			}
		} else {
			round.Competitors = []string{} // Initialize empty slice if no competitors
		}

	}

	return &round, nil
}

func (s *CompetitionRoundService) GetCompetitionRounds(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, competitionId string) (*[]internal_types.CompetitionRound, error) {
	// Build expression for PK and SK begins_with
	keyEx := expression.Key("PK").Equal(expression.Value(fmt.Sprintf("%s", pk)))
	keyEx = keyEx.And(expression.Key("SK").BeginsWith(fmt.Sprintf("COMPETITION_%s_ROUND", competitionId)))

	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 aws.String(competitionRoundsTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	log.Printf("Query INput round: \n%s", formatDynamoDBQueryInput(queryInput))

	result, err := dynamodbClient.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query rounds: %w", err)
	}

	var rounds []internal_types.CompetitionRound
	err = attributevalue.UnmarshalListOfMaps(result.Items, &rounds)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %v", err)
	}

	return &rounds, nil
}

func (s *CompetitionRoundService) UpdateCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string, round internal_types.CompetitionRoundUpdate) (*internal_types.CompetitionRound, error) {
	if competitionRoundsTableName == "" {
		return nil, fmt.Errorf("ERR: competitionRoundsTableName is empty")
	}

	update := expression.UpdateBuilder{}

	// Build dynamic update expression for all possible fields
	if round.RoundName != "" {
		update = update.Set(expression.Name("roundName"), expression.Value(round.RoundName))
	}
	if round.Status != "" {
		update = update.Set(expression.Name("status"), expression.Value(round.Status))
	}
	if round.Competitors != "" {
		update = update.Set(expression.Name("competitors"), expression.Value(round.Competitors))
	}

	if round.CompetitorA != "" {
		update = update.Set(expression.Name("competitorA"), expression.Value(round.CompetitorA))
		// Update matchup when competitorA changes
		if round.CompetitorB != "" {
			matchup := formatMatchup(round.CompetitorA, round.CompetitorB)
			update = update.Set(expression.Name("matchup"), expression.Value(matchup))
		}
	}
	if round.CompetitorB != "" {
		update = update.Set(expression.Name("competitorB"), expression.Value(round.CompetitorB))
		// Update matchup when competitorB changes
		if round.CompetitorA != "" {
			matchup := formatMatchup(round.CompetitorA, round.CompetitorB)
			update = update.Set(expression.Name("matchup"), expression.Value(matchup))
		}
	}
	// For scores, we update even if they're 0 since that's a valid score
	update = update.Set(expression.Name("competitorAScore"), expression.Value(round.CompetitorAScore))
	update = update.Set(expression.Name("competitorBScore"), expression.Value(round.CompetitorBScore))

	if round.Matchup != "" {
		update = update.Set(expression.Name("matchup"), expression.Value(round.Matchup))
	}
	if round.IsPending != "" {
		update = update.Set(expression.Name("isPending"), expression.Value(round.IsPending))
	}
	if round.IsVotingOpen != "" {
		update = update.Set(expression.Name("isVotingOpen"), expression.Value(round.IsVotingOpen))
	}

	// Always update the updatedAt timestamp
	update = update.Set(expression.Name("updatedAt"), expression.Value(round.UpdatedAt))

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(competitionRoundsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: pk},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	result, err := dynamodbClient.UpdateItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update round: %w", err)
	}

	var updatedRound internal_types.CompetitionRound
	err = attributevalue.UnmarshalMap(result.Attributes, &updatedRound)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated round: %w", err)
	}

	return &updatedRound, nil
}

func (s *CompetitionRoundService) DeleteCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(competitionRoundsTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: pk},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: sk},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("competition round successfully deleted")
	return nil
}

// Mock service for testing
type MockCompetitionRoundService struct {
	InsertCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, round internal_types.CompetitionRoundInsert) (*internal_types.CompetitionRound, error)
	GetCompetitionRoundByPkFunc       func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) (*internal_types.CompetitionRound, error)
	GetCompetitionRoundsByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, eventId string) ([]internal_types.CompetitionRound, error)
	UpdateCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string, round internal_types.CompetitionRoundUpdate) (*internal_types.CompetitionRound, error)
	DeleteCompetitionRoundFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) error
}

func (m *MockCompetitionRoundService) InsertCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, round internal_types.CompetitionRoundInsert) (*internal_types.CompetitionRound, error) {
	return m.InsertCompetitionRoundFunc(ctx, dynamodbClient, round)
}

func (m *MockCompetitionRoundService) GetCompetitionRoundByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) (*internal_types.CompetitionRound, error) {
	return m.GetCompetitionRoundByPkFunc(ctx, dynamodbClient, pk, sk)
}

func (m *MockCompetitionRoundService) GetCompetitionRoundsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, eventId string) ([]internal_types.CompetitionRound, error) {
	return m.GetCompetitionRoundsByEventIDFunc(ctx, dynamodbClient, pk, eventId)
}

func (m *MockCompetitionRoundService) UpdateCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string, round internal_types.CompetitionRoundUpdate) (*internal_types.CompetitionRound, error) {
	return m.UpdateCompetitionRoundFunc(ctx, dynamodbClient, pk, sk, round)
}

func (m *MockCompetitionRoundService) DeleteCompetitionRound(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, pk, sk string) error {
	return m.DeleteCompetitionRoundFunc(ctx, dynamodbClient, pk, sk)
}

func formatMatchup(competitorA, competitorB string) string {
	return fmt.Sprintf("%s_%s", competitorA, competitorB)
}
