package transport

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestCreateDbClient(t *testing.T) {
	client := CreateDbClient()
	if client == nil {
		t.Error("CreateDbClient returned nil")
	}
}

func TestGetDB(t *testing.T) {
	// Test with GO_ENV=test
	os.Setenv("GO_ENV", "test")
	defer os.Unsetenv("GO_ENV")

	db1 := GetDB()
	if db1 == nil {
		t.Error("GetDB returned nil for test environment")
	}

	// Ensure we get the same instance when called again
	db2 := GetDB()
	if db1 != db2 {
		t.Error("GetDB returned different instances for test environment")
	}

	// Test with GO_ENV not set (production-like environment)
	os.Unsetenv("GO_ENV")
	prodDB := GetDB()
	if prodDB == nil {
		t.Error("GetDB returned nil for production-like environment")
	}
}

func TestSetTestDB(t *testing.T) {
	mockDB := &test_helpers.MockDynamoDBClient{
		ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{}, nil
		},
	}

	SetTestDB(mockDB)

	os.Setenv("GO_ENV", "test")
	defer os.Unsetenv("GO_ENV")

	retrievedDB := GetDB()
	if retrievedDB != mockDB {
		t.Error("GetDB did not return the mock DB set by SetTestDB")
	}
}
