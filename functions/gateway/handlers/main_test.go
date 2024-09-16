package handlers

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func TestMain(m *testing.M) {
	log.Println("Setting up test environment for handlers package")

	// Set GO_ENV to "test" to trigger test-specific behavior
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)

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
							"lat":    &types.AttributeValueMemberN{Value: "51.5074"},
							"long":   &types.AttributeValueMemberN{Value: "-0.1278"},
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

	os.Exit(exitCode)
}
