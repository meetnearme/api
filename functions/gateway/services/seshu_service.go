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
	EventStartTime 	 string `json:"event_start_time"`
	EventEndTime   	 string `json:"event_end_time"`
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
	EventValidations [][]bool `json:"eventValidations" validate:"optional"`
	Status 		string `json:"status" validate:"optional"`
	CreatedAt int64  `json:"createdAt" validate:"required"`
	UpdatedAt int64  `json:"updatedAt" validate:"required"`
	ExpireAt  int64  `json:"expireAt" validate:"required"`
}

type SeshuSessionGet struct {
	OwnerId    string `json:"ownerId" dynamodbav:"ownerId" validate:"required"`
	Url    string `json:"url" dynamodbav:"url" validate:"required"`
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

const FakeCity = "Nowhere City, NM 11111"
const FakeUrl1 = "http://example.com/events/12345"
const FakeUrl2 = "http://example.com/events/98765"
const FakeEventTitle1 = "Fake Event Title 1"
const FakeEventTitle2 = "Fake Event Title 2"
const FakeStartTime1 = "Sep 26, 26:30pm"
const FakeStartTime2 = "Oct 10, 25:00am"
const FakeEndTime1 = "Sep 26, 27:30pm"
const FakeEndTime2 = "Oct 10, 26:00am"

const InitialEmptyLatLong = 9e+10;

func init () {
	seshuSessionsTableName = helpers.GetDbTableName(helpers.SeshuSessionTablePrefix)
}

func GetSeshuSession(ctx context.Context, db *dynamodb.Client, seshuPayload SeshuSessionGet) (*SeshuSession, error) {
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
		// TODO: this needs to become a map to avoid regressions pertaining
		// to key ordering in the future
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

	// TODO: Before this db PUT, check for existing seshu job
  // key via query to "jobs" table... if it exists, then return a
  // 409 error to the client, explaining it can't be added because
	// that URL is already owned by someone else

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

// TODO: `CommitSeshuSession` needs to consider:
// 1. Validate the session
// 2. Update the status to `committed`
// 3. If the "are these in the same location?" checkbox in UI is not checked,
//    then `latitude` `longitude` and `address` should be ignored from the OpenAI
//    structured response
// 4. Iterate over the union array of `EventCandidates` and `EventValidations` to
//    create a new array that removes any `EventCandidates` that lack any of:
//    `event_title`, `event_location`, `event_start_time` which are all required
// 5. Use that reduced array to find the corresponding strings in the stored
//    `SeshuSession.Html` in the db
// 6. Store the deduced DOM query strings in the new "Scraping Jobs" db table we've
//    not created yet
// 7. The input URL should have it's query params algorithmically sorted to prevent
//    db index (the URL itself is the index) collision / duplicates
// 8. Delete the session from the `SeshuSessions` table once it's confirmed to be
//    successfully committed to the "Scraping Jobs" table



