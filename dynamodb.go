package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)


const TableName = "Events"

var db *dynamodb.Client

type Event struct {
    Id string `json:"id" dynamodbav:"id"`
    Name string `json:"name" dynamodbav:"name"`
    Description string  `json:"description" dynamodbav:"description"`
    Datetime string  `json:"datetime" dynamodbav:"datetime"`
    Address string  `json:"address" dynamodbav:"address"`
    ZipCode string  `json:"zip_code" dynamodbav:"zip_code"`
    Country string  `json:"country" dynamodbav:"country"`
} 

func init() {
    db = CreateLocalClient()
}

func CreateLocalClient() *dynamodb.Client {
    sdkConfig, err := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithEndpointResolver(aws.EndpointResolverFunc(
            func(service, region string) (aws.Endpoint, error) {
                return aws.Endpoint{URL: "http://dynamodb-local:8000"}, nil 
            })),
        config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
            Value: aws.Credentials{
                AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
                Source: "Hard-coded credentials; values are irrelevant for local dynamo",
            },
        }),
    )
    if err != nil {
        panic(err)
    } 
    return dynamodb.NewFromConfig(sdkConfig)
} 

func listItems(ctx context.Context) ([]Event, error) {
    events := make([]Event, 0)
    var token map[string]types.AttributeValue

    for {
        input := &dynamodb.ScanInput{
            TableName: aws.String(TableName),
            ExclusiveStartKey: token,
        }

        result, err := db.Scan(ctx, input)
        if err != nil {
            return nil, err
        } 

        var fetchedEvents []Event
        err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEvents)
        if err != nil {
            return nil, err
        } 

        events = append(events , fetchedEvents...)
        token = result.LastEvaluatedKey
        if token == nil {
            break
        }
    }
    return events, nil 
} 

func insertItem( ctx context.Context, createEvent CreateEvent) (*Event, error) {
    event := Event{
        Name: createEvent.Name,
        Description: createEvent.Description,
        Datetime: createEvent.Datetime,
        Address: createEvent.Address,
        ZipCode: createEvent.ZipCode,
        Id: uuid.NewString(),
    }

    item, err := attributevalue.MarshalMap(event)
    if err != nil {
        fmt.Println("Hitting after marshal map")
        return nil, err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String(TableName),
        Item: item,
    }

    res, err := db.PutItem(ctx, input)
    if err != nil {
        fmt.Println("Hitting after put item")
        return nil, err
    }


    err = attributevalue.UnmarshalMap(res.Attributes, &event)
    if err != nil {
        return nil, err
    }
    return &event, nil 
} 
