package services

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/lambda/helpers"
)

type EventSelect struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Datetime    string `json:"datetime" dynamodbav:"datetime"`
	Address     string `json:"address" dynamodbav:"address"`
	ZipCode     string `json:"zip_code" dynamodbav:"zip_code"`
	Country     string `json:"country" dynamodbav:"country"`
}

type EventInsert struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Datetime    string `json:"datetime" validate:"required"`
	Address     string `json:"address" validate:"required"`
	ZipCode     string `json:"zip_code" validate:"required"`
	Country     string `json:"country" validate:"required"`
}

const EventsTablePrefix = "Events"

var TableName = helpers.GetDbTableName(EventsTablePrefix)

func GetEvents(ctx context.Context, db *dynamodb.Client) ([]EventSelect, error) {

	events := make([]EventSelect, 0)
	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			return nil, err
		}

		var fetchedEvents []EventSelect
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEvents)
		if err != nil {
			return nil, err
		}

		events = append(events, fetchedEvents...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}
	}
	return events, nil
}

func InsertEvent(ctx context.Context, db *dynamodb.Client, createEvent EventInsert) (*EventSelect, error) {
	newEvent := EventSelect{
		Name:        createEvent.Name,
		Description: createEvent.Description,
		Datetime:    createEvent.Datetime,
		Address:     createEvent.Address,
		ZipCode:     createEvent.ZipCode,
		Id:          uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(newEvent)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &newEvent)
	if err != nil {
		return nil, err
	}
	return &newEvent, nil
}
