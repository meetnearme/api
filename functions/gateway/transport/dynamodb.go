package transport

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

var (
	db   *dynamodb.Client
	once sync.Once
)

func init() {
	db = CreateDbClient()
}

func CreateDbClient() types.DynamoDBAPI {

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

func GetDB() *dynamodb.Client {
	once.Do(func() {
		db = CreateDbClient()
	})
	return db
}
