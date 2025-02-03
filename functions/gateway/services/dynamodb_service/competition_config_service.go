package dynamodb_service

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
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

func (s *CompetitionConfigService) UpdateCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string, competitionConfig internal_types.CompetitionConfigUpdate) (*internal_types.CompetitionConfig, error) {
	var competitionConfigResponse internal_types.CompetitionConfig
	// Validate the competition object
	if err := validate.Struct(competitionConfig); err != nil {
		return &competitionConfigResponse, fmt.Errorf("validation failed: %w", err)
	}

	if competitionConfig.Id == "" {
		competitionConfig.Id = uuid.NewString()
	} else {
		var getCompetitionConfigResponse internal_types.CompetitionConfigResponse
		db := transport.GetDB()

		service := NewCompetitionConfigService()
		getCompetitionConfigResponse, err := service.GetCompetitionConfigById(ctx, db, competitionConfig.Id)
		if err != nil {
			return &competitionConfigResponse, err
			// transport.SendServerRes(w, []byte("Failed to get competitionConfig: "+err.Error()), http.StatusInternalServerError, err)
		}

		allOwners := []string{getCompetitionConfigResponse.PrimaryOwner}
		allOwners = append(allOwners, getCompetitionConfigResponse.AuxilaryOwners...)

		userInfo := helpers.UserInfo{}
		if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(helpers.UserInfo)
		}

		roleClaims := []helpers.RoleClaim{}
		if _, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
			roleClaims = ctx.Value("roleClaims").([]helpers.RoleClaim)
		}

		canEdit := helpers.CanEditCompetition(getCompetitionConfigResponse.CompetitionConfig, &userInfo, roleClaims)
		if !canEdit {
			return &competitionConfigResponse, fmt.Errorf("You are not an authorized editor of this competition")
		}
	}

	log.Printf("competitionConfigResponse: %+v", competitionConfigResponse)

	item, err := attributevalue.MarshalMap(&competitionConfig)
	if err != nil {
		return &competitionConfigResponse, err
	}

	// Ensure string arrays are properly formatted as StringSets
	if len(competitionConfig.AuxilaryOwners) > 0 {
		item["auxilaryOwners"] = &dynamodb_types.AttributeValueMemberSS{
			Value: competitionConfig.AuxilaryOwners,
		}
	}

	if len(competitionConfig.EventIds) > 0 {
		item["eventIds"] = &dynamodb_types.AttributeValueMemberSS{
			Value: competitionConfig.EventIds,
		}
	}

	if len(competitionConfig.Competitors) > 0 {
		item["competitors"] = &dynamodb_types.AttributeValueMemberSS{
			Value: competitionConfig.Competitors,
		}
	}

	if competitionConfigTableName == "" {
		return &competitionConfigResponse, fmt.Errorf("ERR: competitionTableName is empty - table reference not retrieved.")
	}

	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(competitionConfigTableName),
		// NOTE: we dropped POST so this is conditionally for POST -- and -- PUT
		// ConditionExpression: aws.String("attribute_not_exists(id)"),
	}

	res, err := dynamodbClient.PutItem(ctx, input)
	if err != nil {
		log.Printf("error in put item dynamo: %v", err)
		return &competitionConfigResponse, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &competitionConfig)
	if err != nil {
		return &competitionConfigResponse, err
	}

	// Use reflection to copy fields
	sourceVal := reflect.ValueOf(competitionConfig)
	targetVal := reflect.ValueOf(&competitionConfigResponse).Elem()

	// copy the valid properties from the source, but omit invalid ones in the new struct
	for i := 0; i < targetVal.NumField(); i++ {
		fieldName := targetVal.Type().Field(i).Name
		if sourceField := sourceVal.FieldByName(fieldName); sourceField.IsValid() {
			targetVal.Field(i).Set(sourceField)
		}
	}

	return &competitionConfigResponse, nil
}

func (s *CompetitionConfigService) GetCompetitionConfigById(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) (internal_types.CompetitionConfigResponse, error) {
	var users []types.UserSearchResultDangerous
	var competitionConfigResponse internal_types.CompetitionConfigResponse
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	roleClaims := []helpers.RoleClaim{}
	if _, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = ctx.Value("roleClaims").([]helpers.RoleClaim)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(competitionConfigTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"id": &dynamodb_types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := dynamodbClient.GetItem(ctx, input)
	if err != nil {
		return competitionConfigResponse, err
	}

	var competition internal_types.CompetitionConfig
	err = attributevalue.UnmarshalMap(result.Item, &competition)
	if err != nil {
		return competitionConfigResponse, err
	}

	allOwners := []string{competition.PrimaryOwner}
	allOwners = append(allOwners, competition.AuxilaryOwners...)

	canEdit := helpers.CanEditCompetition(competition, &userInfo, roleClaims)

	if !canEdit {
		return competitionConfigResponse, fmt.Errorf("You are not an authorized editor of this competition")
	}
	if len(allOwners) > 0 {
		_users, err := helpers.SearchUsersByIDs(allOwners, false)
		if err != nil {
			return competitionConfigResponse, err
		}
		users = _users
	}

	competitionConfigResponse.CompetitionConfig = competition
	competitionConfigResponse.Owners = users

	return competitionConfigResponse, nil
}

func (s *CompetitionConfigService) GetCompetitionConfigsByPrimaryOwner(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, primaryOwner string) (*[]internal_types.CompetitionConfig, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(competitionConfigTableName),
		IndexName:              aws.String("primaryOwner"),
		KeyConditionExpression: aws.String("primaryOwner = :primaryOwner"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":primaryOwner": &dynamodb_types.AttributeValueMemberS{Value: primaryOwner},
		},
	}

	result, err := dynamodbClient.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var competitions []internal_types.CompetitionConfig
	err = attributevalue.UnmarshalListOfMaps(result.Items, &competitions)
	if err != nil {
		return nil, err
	}

	return &competitions, nil
}

func (s *CompetitionConfigService) DeleteCompetitionConfig(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(competitionConfigTableName),
		Key: map[string]dynamodb_types.AttributeValue{
			"id": &dynamodb_types.AttributeValueMemberS{Value: id},
		},
	}

	_, err := dynamodbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	log.Printf("competition config successfully deleted")
	return nil
}

// TODO: Deal with syncing with actual interface
// Mock service for testing
type MockCompetitionConfigService struct {
	GetCompetitionConfigsByPkFunc      func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) (*internal_types.CompetitionConfig, error)
	GetCompetitionConfigsByEventIDFunc func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.CompetitionConfig, error)
	UpdateCompetitionConfigFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string, competition internal_types.CompetitionConfigUpdate) (*internal_types.CompetitionConfig, error)
	DeleteCompetitionConfigFunc        func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, id string) error
}

// Mock service implementation
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
