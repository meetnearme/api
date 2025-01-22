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
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var competitionConfigTableName = helpers.GetDbTableName(helpers.CompetitionConfigTablePrefix)

func init() {
	competitionConfigTableName = helpers.GetDbTableName(helpers.CompetitionConfigTablePrefix)
}

type CompetitionConfigService struct{}

func NewCompetitionConfigService() internal_types.CompetitionConfigServiceInterface {
	return &CompetitionConfigService{}
}

func (s *CompetitionConfigService) InsertCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competitionConfig internal_types.CompetitionConfigInsert) (*internal_types.CompetitionConfig, error) {
	// Validate the competition object
	if err := validate.Struct(competitionConfig); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if competitionConfig.Id == "" {
		competitionConfig.Id = uuid.NewString()
	}

	item, err := attributevalue.MarshalMap(&competitionConfig)
	if err != nil {
		return nil, err
	}

	if competitionConfigTableName == "" {
		return nil, fmt.Errorf("ERR: competitionTableName is empty - table reference not retrieved.")
	}

	input := &dynamodb.PutItemInput{
		Item:                item,
		TableName:           aws.String(competitionConfigTableName),
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	}

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Print("error in put item dynamo")
		return nil, err
	}

	var insertedCompetitionConfig internal_types.CompetitionConfig
	err = attributevalue.UnmarshalMap(res.Attributes, &insertedCompetitionConfig)
	if err != nil {
		return nil, err
	}

	insertedCompetitionConfig.Id = competitionConfig.Id

	return &insertedCompetitionConfig, nil
}

func (s *CompetitionConfigService) GetCompetitionConfigByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, primaryOwner, id string) (*internal_types.CompetitionConfig, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(competitionConfigTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"primaryOwner": &dynamodb_types.AttributeValueMemberS{Value: primaryOwner},
			"id":           &dynamodb_types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var competition internal_types.CompetitionConfig
	err = attributevalue.UnmarshalMap(result.Item, &competition)
	if err != nil {
		return nil, err
	}

	return &competition, nil
}

func (s *CompetitionConfigService) UpdateCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, primaryOwner, id string, competitionConfig internal_types.CompetitionConfigUpdate) (*internal_types.CompetitionConfig, error) {
	if competitionConfigTableName == "" {
		return nil, fmt.Errorf("ERR: competitionTableName is empty")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(competitionConfigTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"primaryOwner": &dynamodb_types.AttributeValueMemberS{Value: primaryOwner},
			"id":           &dynamodb_types.AttributeValueMemberS{Value: id},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]dynamodb_types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
		ReturnValues:              dynamodb_types.ReturnValueAllNew,
	}

	// Add dynamic field updates
	if competitionConfig.Name != "" {
		input.ExpressionAttributeNames["#name"] = "name"
		input.ExpressionAttributeValues[":name"] = &dynamodb_types.AttributeValueMemberS{Value: competitionConfig.Name}
		*input.UpdateExpression += " #name = :name,"
	}

	if competitionConfig.ModuleType != "" {
		input.ExpressionAttributeNames["#moduleType"] = "moduleType"
		input.ExpressionAttributeValues[":moduleType"] = &dynamodb_types.AttributeValueMemberS{Value: competitionConfig.ModuleType}
		*input.UpdateExpression += " #moduleType = :moduleType,"
	}

	if competitionConfig.ScoringMethod != "" {
		input.ExpressionAttributeNames["#scoringMethod"] = "scoringMethod"
		input.ExpressionAttributeValues[":scoringMethod"] = &dynamodb_types.AttributeValueMemberS{Value: competitionConfig.ScoringMethod}
		*input.UpdateExpression += " #scoringMethod = :scoringMethod,"
	}

	if competitionConfig.ModuleType != "" {
		input.ExpressionAttributeNames["#moduleType"] = "moduleType"
		input.ExpressionAttributeValues[":moduleType"] = &dynamodb_types.AttributeValueMemberS{Value: competitionConfig.ModuleType}
		*input.UpdateExpression += " #moduleType = :moduleType,"
	}

	// TODO: need to check the update syntax needed for a []string below is an example of []UserDefinedType all four of these should be that
	if competitionConfig.AuxilaryOwners != "" {
		input.ExpressionAttributeNames["#auxilaryOwners"] = "auxilaryOwners"
		auxilaryOwners, err := attributevalue.MarshalList(competitionConfig.AuxilaryOwners)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":auxilaryOwners"] = &dynamodb_types.AttributeValueMemberL{Value: auxilaryOwners}
		*input.UpdateExpression += " #auxilaryOwners = :auxilaryOwners,"
	}

	if competitionConfig.EventIds != "" {
		input.ExpressionAttributeNames["#eventIds"] = "eventIds"
		eventIds, err := attributevalue.MarshalList(competitionConfig.EventIds)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":eventIds"] = &dynamodb_types.AttributeValueMemberL{Value: eventIds}
		*input.UpdateExpression += " #eventIds = :eventIds,"
	}
	// Rounds         string `json:"rounds,omitempty" dynamodbav:"rounds"`                 // JSON array string
	// Competitors    string `json:"competitors,omitempty" dynamodbav:"competitors"`       // JSON array string
	// Status         string `json:"status,omitempty" dynamodbav:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETE"`
	if competitionConfig.Rounds != "" {
		input.ExpressionAttributeNames["#rounds"] = "rounds"
		rounds, err := attributevalue.MarshalList(competitionConfig.Rounds)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":rounds"] = &dynamodb_types.AttributeValueMemberL{Value: rounds}
		*input.UpdateExpression += " #rounds = :rounds,"
	}

	if competitionConfig.Competitors != "" {
		input.ExpressionAttributeNames["#competitors"] = "competitors"
		competitors, err := attributevalue.MarshalList(competitionConfig.Competitors)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":competitors"] = &dynamodb_types.AttributeValueMemberL{Value: competitors}
		*input.UpdateExpression += " #competitors = :competitors,"
	}

	if competitionConfig.Status != "" {
		input.ExpressionAttributeNames["#status"] = "status"
		input.ExpressionAttributeValues[":status"] = &dynamodb_types.AttributeValueMemberS{Value: competitionConfig.Status}
		*input.UpdateExpression += " #status = :status,"
	}

	// Set the updatedAt field
	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatInt(currentTime, 10)}
	*input.UpdateExpression += " #updatedAt = :updatedAt"

	res, err := dynamodbClient.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var updatedCompetitionConfig internal_types.CompetitionConfig
	err = attributevalue.UnmarshalMap(res.Attributes, &updatedCompetitionConfig)
	if err != nil {
		return nil, err
	}

	return &updatedCompetitionConfig, nil
}

func (s *CompetitionConfigService) DeleteCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, primaryOwner, id string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(competitionConfigTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"primaryOwner": &dynamodb_types.AttributeValueMemberS{Value: primaryOwner},
			"id":           &dynamodb_types.AttributeValueMemberS{Value: id},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("competition config successfully deleted")
	return nil
}

// Mock service for testing
type MockCompetitionConfigService struct {
	InsertCompetitionConfigFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competition internal_types.CompetitionConfigInsert) (*internal_types.CompetitionConfig, error)
	GetCompetitionConfigsByPkFunc      func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) (*internal_types.CompetitionConfig, error)
	GetCompetitionConfigsByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.CompetitionConfig, error)
	UpdateCompetitionConfigFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string, competition internal_types.CompetitionConfigUpdate) (*internal_types.CompetitionConfig, error)
	DeleteCompetitionConfigFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) error
}

// Mock service implementation
func (m *MockCompetitionConfigService) InsertCompetitionConfigConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, competition internal_types.CompetitionConfigInsert) (*internal_types.CompetitionConfig, error) {
	return m.InsertCompetitionConfigFunc(ctx, dynamodbClient, competition)
}

func (m *MockCompetitionConfigService) GetCompetitionConfigConfigByPk(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) (*internal_types.CompetitionConfig, error) {
	return m.GetCompetitionConfigsByPkFunc(ctx, dynamodbClient, id)
}

func (m *MockCompetitionConfigService) GetCompetitionConfigsByEventID(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.CompetitionConfig, error) {
	return m.GetCompetitionConfigsByEventIDFunc(ctx, dynamodbClient, eventId)
}

func (m *MockCompetitionConfigService) UpdateCompetitionConfigConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string, competition internal_types.CompetitionConfigUpdate) (*internal_types.CompetitionConfig, error) {
	return m.UpdateCompetitionConfigFunc(ctx, dynamodbClient, id, competition)
}

func (m *MockCompetitionConfigService) DeleteCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) error {
	return m.DeleteCompetitionConfigFunc(ctx, dynamodbClient, id)
}
