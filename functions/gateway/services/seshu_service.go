package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
)

type EventInfo struct {
	EventTitle    	 string `json:"event_title"`
	EventLocation 	 string `json:"event_location"`
	EventDate     	 string `json:"event_date"`
	EventURL      	 string `json:"event_url"`
	EventDescription string `json:"event_description"`
}

type SeshuSession struct {
	OwnerId    string `json:"ownerId" validate:"required"`
	Url        string `json:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" validate:"required"`
	UrlPath        string `json:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" validate:"optional"`
	Html      string `json:"html" validate:"required"`
	EventCandidates	 []EventInfo `json:"eventCandidates" validate:"optional"`
	Status 		string `json:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" validate:"required"`
	UpdatedAt int64  `json:"updatedAt" validate:"required"`
	ExpireAt  int64  `json:"expireAt" validate:"required"`
}

type SeshuSessionInput struct {
	SeshuSession
	CreatedAt struct{} `json:"createdAt,omitempty"`
	UpdatedAt struct{} `json:"updatedAt,omitempty"`
	ExpireAt struct{} `json:"expireAt,omitempty"`
}

type SeshuSessionInsert struct {
	OwnerId    string `json:"ownerId" dynamodbav:"ownerId" validate:"required"`
	Url        string `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" dynamodbav:"urlDomain" validate:"required"`
	UrlPath        string `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html      string `json:"html" dynamodbav:"html" validate:"required"`
	EventCandidates	 []EventInfo `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations [][]bool `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status 		string `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" dynamodbav:"createdAt" validate:"required"`
	UpdatedAt int64  `json:"updatedAt" dynamodbav:"updatedAt" validate:"required"`
	ExpireAt  int64  `json:"expireAt" dynamodbav:"expireAt" validate:"required"`
}
type SeshuSessionUpdate struct {
	OwnerId    string `json:"ownerId" dynamodbav:"ownerId" validate:"optional"`
	Url        string `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain      string `json:"urlDomain" dynamodbav:"urlDomain" validate:"optional"`
	UrlPath        string `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams url.Values `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64 `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64 `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string  `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html      string `json:"html" dynamodbav:"html" validate:"optional"`
	EventCandidates   []EventInfo `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations [][]bool `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status 		string `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" dynamodbav:"createdAt" validate:"optional"`
	UpdatedAt int64  `json:"updatedAt" dynamodbav:"updatedAt" validate:"optional"`
	ExpireAt  int64  `json:"expireAt" dynamodbav:"expireAt" validate:"optional"`
}

var seshuSessionsTableName = helpers.GetDbTableName(helpers.SeshuSessionTablePrefix)

func init () {
	seshuSessionsTableName = helpers.GetDbTableName(helpers.SeshuSessionTablePrefix)
}

func GetSeshuSession(ctx context.Context, db *dynamodb.Client, seshuPayload SeshuSession) (*SeshuSession, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Key: map[string]types.AttributeValue{
			"url": &types.AttributeValueMemberS{Value: seshuPayload.Url},
		},
	}

	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var seshuSession SeshuSession
	err = attributevalue.UnmarshalMap(result.Item, &seshuSession)
	if err != nil {
		return nil, err
	}

	return &seshuSession, nil
}

// curl -X POST -H "Content-Type: application/json" -d '{
//   "name": "New York Event",
//   "description": "An event in New York",
//   "datetime": "2023-06-01T10:00:00Z",
//   "locationAddress": "123 Main Street",
//   "zip_code": "10001",
//   "country": "USA",
//   "locationLatitude": 40.7128,
//   "locationLongitude": -74.0060
// }' https://w65hlwklek.execute-api.us-east-1.amazonaws.com/api/event

// curl -X POST -H 'Content-Type: application/json' -d '{"ownerId": "123", "url": "http://example.com/path?key=value", "urlProperties": { "urlDomain": "example.com", "urlPath": "/path", "urlQueryParams": { "key": "value"} }, "location": {"locationLatitude": 35.6869752, "locationLongitude": 105.937799, "locationAddress": "Santa Fe, NM, USA" }, "html": "<html><body>Test HTML</body></html>"}' https://t3tgatdysl.execute-api.us-east-1.amazonaws.com/api/seshu/session


func InsertSeshuSession(ctx context.Context, db *dynamodb.Client, seshuPayload SeshuSessionInput) (*SeshuSessionInsert, error) {
	currentTime := time.Now().Unix()
	if len(seshuPayload.EventCandidates) < 1 {
		seshuPayload.EventCandidates = []EventInfo{}
	}
	newSeshuSession := SeshuSessionInsert{
		OwnerId:    seshuPayload.OwnerId,
		Url:  seshuPayload.Url,
		UrlDomain: seshuPayload.UrlDomain,
		UrlPath: seshuPayload.UrlPath,
		UrlQueryParams: seshuPayload.UrlQueryParams,
		LocationLatitude:  seshuPayload.LocationLatitude,
		LocationLongitude: seshuPayload.LocationLongitude,
		LocationAddress:   seshuPayload.LocationAddress,
		Html:       seshuPayload.Html,
		EventCandidates: seshuPayload.EventCandidates,
		EventValidations: [][]bool{},
		Status:		 "draft",
		ExpireAt:   currentTime + 3600*24, // 24 hrs expiration
		CreatedAt:  currentTime,
		UpdatedAt:  currentTime,
	}

	item, err := attributevalue.MarshalMap(newSeshuSession)
	if err != nil {
		return nil, err
	}

	if (seshuSessionsTableName == "") {
		return nil, fmt.Errorf("ERR: seshuSessionsTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Item: item,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &newSeshuSession)
	if err != nil {
		return nil, err
	}

	// TODO: omit newSeshuSession.Html from response, it's too large
	return &newSeshuSession, nil
}

func UpdateSeshuSession(ctx context.Context, db *dynamodb.Client, seshuPayload SeshuSessionUpdate) (*SeshuSessionUpdate, error) {

	// TODO: DB call to check if it exists first, and the the owner is the same as the one updating

	if (seshuSessionsTableName == "") {
		return nil, fmt.Errorf("ERR: seshuSessionsTableName is empty")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Key: map[string]types.AttributeValue{
			"url": &types.AttributeValueMemberS{Value: seshuPayload.Url},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
	}

	if seshuPayload.OwnerId != "" {
		input.ExpressionAttributeNames["#ownerId"] = "ownerId"
		input.ExpressionAttributeValues[":ownerId"] = &types.AttributeValueMemberS{Value: seshuPayload.OwnerId}
		*input.UpdateExpression += " #ownerId = :ownerId,"
	}

	if seshuPayload.UrlDomain != "" {
		input.ExpressionAttributeNames["#urlDomain"] = "urlDomain"
		input.ExpressionAttributeValues[":urlDomain"] = &types.AttributeValueMemberS{Value: seshuPayload.UrlDomain}
		*input.UpdateExpression += " #urlDomain = :urlDomain,"
	}

	if seshuPayload.UrlPath != "" {
		input.ExpressionAttributeNames["#urlPath"] = "urlPath"
		input.ExpressionAttributeValues[":urlPath"] = &types.AttributeValueMemberS{Value: seshuPayload.UrlPath}
		*input.UpdateExpression += " #urlPath = :urlPath,"
	}

	if len(seshuPayload.UrlQueryParams) > 0 {
		input.ExpressionAttributeNames["#urlQueryParams"] = "urlQueryParams"
		queryParams, err := attributevalue.MarshalMap(seshuPayload.UrlQueryParams)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":urlQueryParams"] = &types.AttributeValueMemberM{Value: queryParams}
		*input.UpdateExpression += " #urlQueryParams = :urlQueryParams,"
	}

	if seshuPayload.LocationLatitude != 0 {
		input.ExpressionAttributeNames["#locationLatitude"] = "locationLatitude"
		input.ExpressionAttributeValues[":locationLatitude"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(seshuPayload.LocationLatitude, 'f', -1, 64)}
		*input.UpdateExpression += " #locationLatitude = :locationLatitude,"
	}

	if seshuPayload.LocationLongitude != 0 {
		input.ExpressionAttributeNames["#locationLongitude"] = "locationLongitude"
		input.ExpressionAttributeValues[":locationLongitude"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(seshuPayload.LocationLongitude, 'f', -1, 64)}
		*input.UpdateExpression += " #locationLongitude = :locationLongitude,"
	}

	if seshuPayload.LocationAddress != "" {
		input.ExpressionAttributeNames["#locationAddress"] = "locationAddress"
		input.ExpressionAttributeValues[":locationAddress"] = &types.AttributeValueMemberS{Value: seshuPayload.LocationAddress}
		*input.UpdateExpression += " #locationAddress = :locationAddress,"
	}

	if seshuPayload.Html != "" {
		input.ExpressionAttributeNames["#html"] = "html"
		input.ExpressionAttributeValues[":html"] = &types.AttributeValueMemberS{Value: seshuPayload.Html}
		*input.UpdateExpression += " #html = :html,"
	}

	if seshuPayload.EventCandidates != nil {
		input.ExpressionAttributeNames["#eventCandidates"] = "eventCandidates"
		eventCandidates, err := attributevalue.MarshalList(seshuPayload.EventCandidates)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":eventCandidates"] = &types.AttributeValueMemberL{Value: eventCandidates}
		*input.UpdateExpression += " #eventCandidates = :eventCandidates,"
	}

	if seshuPayload.EventValidations != nil {
		input.ExpressionAttributeNames["#eventValidations"] = "eventValidations"
		eventValidations, err := attributevalue.MarshalList(seshuPayload.EventValidations)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":eventValidations"] = &types.AttributeValueMemberL{Value: eventValidations}
		*input.UpdateExpression += " #eventValidations = :eventValidations,"
	}

	if seshuPayload.Status != "" {
		input.ExpressionAttributeNames["#status"] = "status"
		input.ExpressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: seshuPayload.Status}
		*input.UpdateExpression += " #status = :status,"
	}

	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(float64(currentTime), 'f', -1, 64)}
	*input.UpdateExpression += " #updatedAt = :updatedAt"

	_, err := db.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	log.Printf("Updated seshu session: %+v", seshuPayload.Url)

	return nil, nil
}


