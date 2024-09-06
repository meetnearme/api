package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/indexing"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type EventSelect struct {
	Id          string `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	StartTime    string `json:"start_time" validate:"required"`
	Address     string `json:"address"`
	Lat			 		float64 `json:"lat" validate:"required"`
	Long				float64 `json:"long" validate:"required"`
	// ZOrderIndex []byte `json:"z_order_index" dynamodbav:"zOrderIndex,B"`
}

type EventInsert struct {
	Id          string `json:"id"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	StartTime   string `json:"startTime" validate:"required"`
	Address     string `json:"address" validate:"required"`
	Lat    			float64 `json:"lat" validate:"required"`
	Long    		float64 `json:"long" validate:"required"`
	// ZOrderIndex []byte `json:"z_order_index"`
}

var eventsTableName = helpers.GetDbTableName(helpers.EventsTablePrefix)

func GetEvents(ctx context.Context, db *dynamodb.Client) ([]EventSelect, error) {

	events := make([]EventSelect, 0)
	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(eventsTableName),
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

func GetEventsZOrder(ctx context.Context, db internal_types.DynamoDBAPI, startTime, endTime time.Time, lat, lon, radius float32) ([]EventSelect, error) {
    minZOrderIndex, err := indexing.CalculateZOrderIndex(startTime, lat, lon, "min")
    if err != nil {
        return nil, fmt.Errorf("error calculating min z-order index: %v", err)
    }

    maxZOrderIndex, err := indexing.CalculateZOrderIndex(endTime, lat, lon, "max")
    if err != nil {
        return nil, fmt.Errorf("error calculating max z-order index: %v", err)
    }

    scanInput := &dynamodb.ScanInput{
        TableName: aws.String(eventsTableName),
        FilterExpression: aws.String(
            "#zOrderIndex BETWEEN :min AND :max AND #datetime BETWEEN :startTime AND :endTime",
        ),
        ExpressionAttributeNames: map[string]string{
            "#zOrderIndex": "zOrderIndex",
            "#datetime": "datetime",
        },
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":min":       &types.AttributeValueMemberB{Value: minZOrderIndex},
            ":max":       &types.AttributeValueMemberB{Value: maxZOrderIndex},
            ":startTime": &types.AttributeValueMemberS{Value: startTime.Format(time.RFC3339)},
            ":endTime":   &types.AttributeValueMemberS{Value: endTime.Format(time.RFC3339)},
        },
    }


    scanResult, err := db.Scan(ctx, scanInput)
    if err != nil {
        return nil, err
    }

    var events []EventSelect
    err = attributevalue.UnmarshalListOfMaps(scanResult.Items, &events)
    if err != nil {
        return nil, err
    }

    return events, nil
}


func GetEventById(ctx context.Context, db internal_types.DynamoDBAPI, eventId string) (*EventSelect, error) {
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(eventsTableName),
		FilterExpression: aws.String(
			"#id = :id",
		),
		ExpressionAttributeNames: map[string]string{
			"#id": "id",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberS{Value: eventId},
		},
	}

	scanResult, err := db.Scan(ctx, scanInput)
	if err != nil {
			log.Println("error scanning for event", err)
			return nil, err
	}

	if len(scanResult.Items) == 0 {
		return nil, fmt.Errorf("event not found")
	}

	var event EventSelect
	err = attributevalue.UnmarshalMap(scanResult.Items[0], &event)
	if err != nil {
			log.Println("error unmarshalling event", err)
			return nil, err
	}

	return &event, nil
}


func InsertEvent(ctx context.Context, db internal_types.DynamoDBAPI, createEvent EventInsert) (*EventSelect, error) {
    // startTime, err := time.Parse(time.RFC3339, createEvent.StartTime)
    // if err != nil {
    //     return nil, fmt.Errorf("invalid datetime format: %v", err)
    // }

    // zOrderIndex, err := indexing.CalculateZOrderIndex(startTime, createEvent.Lat, createEvent.Long, "default")
    // if err != nil {
    //     return nil, fmt.Errorf("failed to calculate Z Order index: %v", err)
    // }

	newEvent := EventSelect{
		Name:        createEvent.Name,
		Description: createEvent.Description,
		StartTime:    createEvent.StartTime,
		Address:     createEvent.Address,
        Lat: createEvent.Lat,
        Long: createEvent.Long,

		Id:          uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(newEvent)
	log.Println("item after marshalMap", item)
	log.Println("item.zOrderIndex after marshalMap:", item["zOrderIndex"])
	log.Println("item.name after marshalMap:", item["name"])
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(eventsTableName),
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

func InsertEventToMarqo(eventToCreate EventInsert) (*marqo.UpsertDocumentsResponse, error) {
	newEvent := EventInsert{
		Name:        eventToCreate.Name,
		Description: eventToCreate.Description,
		StartTime:    eventToCreate.StartTime,
		Address:     eventToCreate.Address,
		Lat: eventToCreate.Lat,
		Long: eventToCreate.Long,
		Id:          uuid.NewString(),
	}

	marqoClient, err := GetMarqoClient()
	if err != nil {
		return nil, err
	}

	item, err := UpsertEventToMarqo(marqoClient, newEvent)
	if err != nil {
		log.Println(">>> error upserting event to marqo", err)
		return nil, err
	}

	log.Println("item after upsert", item)

	return item, nil
}
