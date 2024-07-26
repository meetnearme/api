package transport

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var (
	db   internal_types.DynamoDBAPI
	once sync.Once
    testDB internal_types.DynamoDBAPI
)

func init() {
	db = CreateDbClient()
}

func CreateDbClient() internal_types.DynamoDBAPI {

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

	cfg, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		fmt.Println("Error loading default Dynamo client config", err)
	}

	if !helpers.IsRemoteDB() {
		optionalCredentials := config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: "test", SecretAccessKey: "test", SessionToken: "test",
				Source: "Hard-coded credentials; values are irrelevant for local dynamo",
			},
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver), optionalCredentials)
		log.Println("Connected to LOCAL DB")
	}

	if err != nil {
		panic(err)
	}

	return dynamodb.NewFromConfig(cfg)
}

func SetTestDB(db internal_types.DynamoDBAPI) {
    testDB = db
}

func GetDB() internal_types.DynamoDBAPI {
    log.Printf("GetDB called. GO_ENV: %s, testDB is nil: %v", os.Getenv("GO_ENV"), testDB == nil)
    if os.Getenv("GO_ENV") == "test" {
        if testDB == nil {
            log.Println("Creating mock DB for testing")
            testDB = &test_helpers.MockDynamoDBClient{
                ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
                    return &dynamodb.ScanOutput{
                        Items: []map[string]types.AttributeValue{},
                    }, nil
                },
            }
        }
        log.Println("Returning mock DB for testing")
        return testDB
    }
	once.Do(func() {
        log.Println("Creating real DB client")
		db = CreateDbClient()
	})
    log.Println("Returning real DB client")
	return db
}
