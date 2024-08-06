package handlers

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func TestMain(m *testing.M) {
	log.Println("Setting up test environment for handlers package")

	// Set GO_ENV to "test" to trigger test-specific behavior
	os.Setenv("GO_ENV", "test")

	mockDB := &test_helpers.MockDynamoDBClient{
		ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{
				Items: []map[string]types.AttributeValue{},
			}, nil
		},
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
		// Add other mock functions as needed
	}

	transport.SetTestDB(mockDB)

	log.Println("Running tests for handlers package")
	exitCode := m.Run()

	log.Println("Cleaning up test environment for handlers package")
	// Perform any necessary cleanup here

	os.Exit(exitCode)
}
