package services

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/meetnearme/api/functions/lambda/helpers"
	"github.com/meetnearme/api/functions/lambda/indexing"
)

type EventSelect struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Datetime    string `json:"datetime" dynamodbav:"datetime"`
	Address     string `json:"address" dynamodbav:"address"`
	ZipCode     string `json:"zip_code" dynamodbav:"zip_code"`
	Country     string `json:"country" dynamodbav:"country"`
    Latitude float64 `json:"latitude" dynamodbav:"latitude,N"`
    Longitude float64 `json:"longitude" dynamodbav:"longitude,N"`
    ZOrderIndex []byte `json:"zOrderIndex" dynamodbav:"z_order_index,B"`
}

type EventInsert struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Datetime    string `json:"datetime" validate:"required"`
	Address     string `json:"address" validate:"required"`
	ZipCode     string `json:"zip_code" validate:"required"`
	Country     string `json:"country" validate:"required"`
    Latitude float64 `json:"latitude" validate:"required"`
    Longitude float64 `json:"longitude" validate:"required"`
    ZOrderIndex []byte `json:"zOrderIndex"`
}

var TableName = helpers.GetDbTableName(helpers.EventsTablePrefix)

func GetEvents(ctx context.Context, db *dynamodb.Client, startTime, endTime time.Time, lat, lon, radius float64) ([]EventSelect, error) {
	// events := make([]EventSelect, 0)

    // Calc z order index range for given time and location
    minZOrderIndex, err := indexing.CalculateZOrderIndex(startTime, lat, lon, "min")
    if err != nil {
        return nil, fmt.Errorf("error calculating min z-order index: %v", err)
    } 

    maxZOrderIndex, err := indexing.CalculateZOrderIndex(startTime, lat, lon, "max")
    if err != nil {
        return nil, fmt.Errorf("error calculating max z-order index: %v", err)
    } 


    input := &dynamodb.QueryInput{
        TableName: aws.String(TableName),
        IndexName: aws.String("zorder_index"),
        KeyConditionExpression: aws.String(
            "z_order_index BETWEEN :min AND :max AND #datetime BETWEEN :startTime AND :endTime",
        ),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":min": &types.AttributeValueMemberB{Value: minZOrderIndex},
            ":max": &types.AttributeValueMemberB{Value: maxZOrderIndex},
            ":startTime": &types.AttributeValueMemberS{Value: startTime.Format(time.RFC3339)},
            ":endTime": &types.AttributeValueMemberS{Value: endTime.Format(time.RFC3339)},
        },
    }

    result, err := db.Query(ctx, input)
    if err != nil {
        return nil, err
    } 

    var fetchedEvents []EventSelect
    err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEvents)
    if err != nil {
        return nil, err
    } 

    return fetchedEvents, nil

	// var token map[string]types.AttributeValue

	// for {
	// 	input := &dynamodb.ScanInput{
	// 		TableName:         aws.String(TableName),
	// 		ExclusiveStartKey: token,
	// 	}

	// 	result, err := db.Scan(ctx, input)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	var fetchedEvents []EventSelect
	// 	err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEvents)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	events = append(events, fetchedEvents...)
	// 	token = result.LastEvaluatedKey
	// 	if token == nil {
	// 		break
	// 	}
	// }
	// return events, nil
}

func InsertEvent(ctx context.Context, db *dynamodb.Client, createEvent EventInsert) (*EventSelect, error) {

	newEvent := EventSelect{
		Name:        createEvent.Name,
		Description: createEvent.Description,
		Datetime:    createEvent.Datetime,
		Address:     createEvent.Address,
		ZipCode:     createEvent.ZipCode,
        Latitude: createEvent.Latitude,
        Longitude: createEvent.Longitude,
		Id:          uuid.NewString(),
	}

    // Calculate Z order index val
    startTime, err := time.Parse(time.RFC3339, createEvent.Datetime)
    if err != nil {
        return nil, err
    } 

    newEvent.ZOrderIndex, err = indexing.CalculateZOrderIndex(startTime, createEvent.Latitude, createEvent.Longitude, "actual")
    if err != nil {
        return nil, err
    } 

    eventCreationTime := time.Now()

    var eventCreationTimeBinary [8]byte
    binary.BigEndian.PutUint64(eventCreationTimeBinary[:], uint64(eventCreationTime.UnixNano()))

    newEvent.ZOrderIndex = append(newEvent.ZOrderIndex, eventCreationTimeBinary[:]...)

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
