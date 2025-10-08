package handlers

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	// rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

var testClient *weaviate.Client

func TestMain(m *testing.M) {
	log.Println("Setting up test environment for handlers package")

	originalEnv := map[string]string{
		"APEX_URL":              os.Getenv("APEX_URL"),
		"ZITADEL_INSTANCE_HOST": os.Getenv("ZITADEL_INSTANCE_HOST"),
		"ZITADEL_CLIENT_ID":     os.Getenv("ZITADEL_CLIENT_ID"),
		"ZITADEL_CLIENT_SECRET": os.Getenv("ZITADEL_CLIENT_SECRET"),
	}

	originalFlags := map[string]string{}
	for _, name := range []string{"authorizeURI", "tokenURI", "jwksURI", "redirectURI", "loginPageURI", "endSessionURI", "clientID", "clientSecret"} {
		if f := flag.Lookup(name); f != nil {
			originalFlags[name] = f.Value.String()
		}
	}

	// This connects to the real Weaviate container from your docker-compose.test.yml
	// os.Setenv("WEAVIATE_HOST", "localhost")
	// os.Setenv("WEAVIATE_PORT", "8080")
	// os.Setenv("WEAVIATE_SCHEME", "http")

	var err error
	testClient, err = services.GetWeaviateClient()
	if err != nil {
		log.Fatalf("FATAL: Could not create Weaviate client for handler tests: %v", err)
	}

	os.Setenv("GO_ENV", constants.GO_TEST_ENV)
	helpers.InitDefaultProtocol() // Re-initialize protocol after setting GO_ENV
	os.Setenv("APEX_URL", "https://test.example.com")
	os.Setenv("ZITADEL_INSTANCE_HOST", "test.zitadel.cloud")
	os.Setenv("ZITADEL_CLIENT_ID", "test-client-id")
	os.Setenv("ZITADEL_CLIENT_SECRET", "test-client-secret")

	flag.Set("authorizeURI", "https://test.zitadel.cloud/oauth/v2/authorize")
	flag.Set("tokenURI", "https://test.zitadel.cloud/oauth/v2/token")
	flag.Set("jwksURI", "https://test.zitadel.cloud/oauth/v2/keys")
	flag.Set("endSessionURI", "https://test.zitadel.cloud/oauth/v2/end_session")
	flag.Set("redirectURI", "https://test.example.com/auth/callback")
	flag.Set("loginPageURI", "https://test.example.com")
	flag.Set("clientID", "test-client-id")
	flag.Set("clientSecret", "test-client-secret")

	mockDB := &test_helpers.MockDynamoDBClient{
		ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			if params.FilterExpression != nil && *params.FilterExpression == "#id = :id" {
				// This is for GetEvent
				return &dynamodb.ScanOutput{
					Items: []map[string]types.AttributeValue{
						{
							"id":          &types.AttributeValueMemberS{Value: "123"},
							"name":        &types.AttributeValueMemberS{Value: "Test Event (single GetEvent by #id)"},
							"description": &types.AttributeValueMemberS{Value: "This is a test event"},
							"datetime":    &types.AttributeValueMemberS{Value: "2023-05-01T12:00:00Z"},
							"address":     &types.AttributeValueMemberS{Value: "123 Test St"},
							"zip_code":    &types.AttributeValueMemberS{Value: "12345"},
							"country":     &types.AttributeValueMemberS{Value: "Test Country"},
							"lat":         &types.AttributeValueMemberN{Value: "51.5074"},
							"long":        &types.AttributeValueMemberN{Value: "-0.1278"},
						},
					},
				}, nil
			} else {
				// catch for anything un-implemented
				return &dynamodb.ScanOutput{}, nil
			}
		},
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	transport.SetTestDB(mockDB)

	log.Println("Running tests for handlers package")
	exitCode := m.Run()

	log.Println("Cleaning up test environment for handlers package")
	// Perform any necessary cleanup here
	for key, value := range originalEnv {
		os.Setenv(key, value)
	}
	for name, value := range originalFlags {
		flag.Set(name, value)
	}

	os.Exit(exitCode)
}
