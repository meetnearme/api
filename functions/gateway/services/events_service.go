package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/indexing"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

const (
	earthRadiusKm = 6378.0
	milesPerKm    = 0.621371
)

type EventSelect struct {
	Id          string `json:"id" dynamodbav:"id"`
	Name        string `json:"name" dynamodbav:"name"`
	Description string `json:"description" dynamodbav:"description"`
	Datetime    string `json:"datetime" dynamodbav:"datetime"`
	Address     string `json:"address" dynamodbav:"address"`
	ZipCode     string `json:"zip_code" dynamodbav:"zip_code"`
	Country     string `json:"country" validate:"required"`
	Latitude 		float32 `json:"latitude" validate:"required"`
	Longitude		float32 `json:"longitude" validate:"required"`
	ZOrderIndex []byte `json:"z_order_index" dynamodbav:"zOrderIndex,B"`
}

type EventInsert struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	Datetime    string `json:"datetime" validate:"required"`
	Address     string `json:"address" validate:"required"`
	ZipCode     string `json:"zip_code" validate:"required"`
	Country     string `json:"country" validate:"required"`
	Latitude 		float32 `json:"latitude" validate:"required"`
	Longitude		float32 `json:"longitude" validate:"required"`
	ZOrderIndex []byte `json:"z_order_index" dynamodbav:"zOrderIndex,B"`
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


func offsetLatLon(radius float32, lat, lon float32, corner string) (newLat, newLon float32) {
	radiusKm := float64(radius) / milesPerKm
	halfRadius := radiusKm / 2
	latOffset := (halfRadius / earthRadiusKm) * (180 / math.Pi)
	lonOffset := (halfRadius / earthRadiusKm) * (180 / math.Pi) / math.Cos(float64(lat)*math.Pi/180)

	switch corner {
	case "upper left":
		newLat = lat + float32(latOffset)
		newLon = lon - float32(lonOffset)
	case "lower right":
		newLat = lat - float32(latOffset)
		newLon = lon + float32(lonOffset)
	default:
		return lat, lon // Return original coordinates if corner is invalid
	}

	return newLat, newLon
}

func GetEventsZOrder(ctx context.Context, db internal_types.DynamoDBAPI, startTime, endTime time.Time, lat, lon float32, radius float32) ([]EventSelect, error) {
    // Calculate the bounding box coordinates
		maxLat, minLon := offsetLatLon(radius, lat, lon, "upper left")
		minLat, maxLon := offsetLatLon(radius, lat, lon, "lower right")

    // Calculate Z-order indices for the corners of the bounding box
		log.Println("minLat: ", minLat)
		log.Println("maxLat: ", maxLat)
		log.Println("minLon: ", minLon)
		log.Println("maxLon: ", maxLon)

		// TODO: this is temporary, need to decide how to properlhandle radius offset
    // minZOrderIndex, err := indexing.CalculateZOrderIndex(startTime, minLat, minLon, "min")
		minZOrderIndex, err := indexing.CalculateZOrderIndex(startTime, minLat, minLon, "min")
    if err != nil {
        return nil, fmt.Errorf("error calculating min z-order index: %v", err)
    }

		// TODO: this is temporary, need to decide how to properlhandle radius offset
    // maxZOrderIndex, err := indexing.CalculateZOrderIndex(endTime, maxLat, maxLon, "max")
		maxZOrderIndex, err := indexing.CalculateZOrderIndex(endTime, maxLat, maxLon, "max")
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
    startTime, err := time.Parse(time.RFC3339, createEvent.Datetime)
    if err != nil {
        return nil, fmt.Errorf("invalid datetime format: %v", err)
    }

    zOrderIndex, err := indexing.CalculateZOrderIndex(startTime, createEvent.Latitude, createEvent.Longitude, "default")
    if err != nil {
        return nil, fmt.Errorf("failed to calculate Z Order index: %v", err)
    }

	newEvent := EventSelect{
		Name:        createEvent.Name,
		Description: createEvent.Description,
		Datetime:    createEvent.Datetime,
		Address:     createEvent.Address,
		ZipCode:     createEvent.ZipCode,
        Latitude: createEvent.Latitude,
        Longitude: createEvent.Longitude,
        ZOrderIndex: zOrderIndex,
		Id:          uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(newEvent)
	log.Println("newEvent.ZorderIndex", newEvent.ZOrderIndex)
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
