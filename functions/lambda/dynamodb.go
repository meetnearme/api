package main

import (
	"context"
	"fmt"
	"os"

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
    db = CreateDbClient()
}

func CreateDbClient() *dynamodb.Client {

    fmt.Println("Creating local client ==>", os.Getenv("SST_STAGE"))

    dbUrl := "http://localhost:8000"
    if (os.Getenv("SST_STAGE") == "prod") {
        dbUrl = "http://some-prod-url-from-secrets.com"
    }

    customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
        if service == dynamodb.ServiceID && region == "us-east-1" {
            return aws.Endpoint{
                PartitionID:   "aws",
                URL:           dbUrl,
                SigningRegion: "us-east-1",
            }, nil
        }
        // returning EndpointNotFoundError will allow the service to fallback to it's default resolution
        return aws.Endpoint{}, &aws.EndpointNotFoundError{}
    })

    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver),
    config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
        Value: aws.Credentials{
            AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
            Source: "Hard-coded credentials; values are irrelevant for local dynamo",
        },
    }))

    if err != nil {
        panic(err)
    }
    return dynamodb.NewFromConfig(cfg)
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
