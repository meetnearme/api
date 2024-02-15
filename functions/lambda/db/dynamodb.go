package db

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/lambda/shared"
)

var TableName = getDbTableName(shared.EventsTablePrefix)

type DynamoDB struct {
    Client *dynamodb.Client
} 

func NewDynamoDB() *DynamoDB {
    db := CreateDBClient()
    return &DynamoDB{Client: db}
} 

func CreateDBClient() *dynamodb.Client {

    // used for local dev via aws sam in docker container
    dbUrl := "http://localhost:8000"

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

    cfg, err :=  config.LoadDefaultConfig(context.TODO())

    if (os.Getenv("SST_STAGE") != "prod") {
        optionalCredentials := config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
            Value: aws.Credentials{
                AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
                Source: "Hard-coded credentials; values are irrelevant for local dynamo",
            },
        })
        cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver), optionalCredentials)
    }

    if err != nil {
        panic(err)
    }
    return dynamodb.NewFromConfig(cfg)
}

func (db *DynamoDB) ListItems(ctx context.Context) ([]shared.Event, error) {
    events := make([]shared.Event, 0)
    var token map[string]types.AttributeValue

    for {
        input := &dynamodb.ScanInput{
            TableName: aws.String(TableName),
            ExclusiveStartKey: token,
        }

        result, err := db.Client.Scan(ctx, input)
        if err != nil {
            return nil, err
        }

        var fetchedEvents []shared.Event
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


func (db *DynamoDB) InsertItem( ctx context.Context, createEvent shared.CreateEvent) (*shared.Event, error) {
    event := shared.Event{
        Name: createEvent.Name,
        Description: createEvent.Description,
        Datetime: createEvent.Datetime,
        Address: createEvent.Address,
        ZipCode: createEvent.ZipCode,
        Id: uuid.NewString(),
    }

    item, err := attributevalue.MarshalMap(event)
    if err != nil {
        return nil, err
    }

    input := &dynamodb.PutItemInput{
        TableName: aws.String(TableName),
        Item: item,
    }

    res, err := db.Client.PutItem(ctx, input)
    if err != nil {
        return nil, err
    }


    err = attributevalue.UnmarshalMap(res.Attributes, &event)
    if err != nil {
        return nil, err
    }
    return &event, nil
}
// Helpers 


func getDbTableName (tableName string) string {
    var SST_Table_tableName_Events = os.Getenv("SST_Table_tableName_Events")
    if (os.Getenv("SST_STAGE") != "prod") {
        return tableName
    }
    return SST_Table_tableName_Events
}

