package dynamodb_service

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type MockDynamoDBClient struct {
	// Mock responses
	GetItemOutput    *dynamodb.GetItemOutput
	PutItemOutput    *dynamodb.PutItemOutput
	UpdateItemOutput *dynamodb.UpdateItemOutput
	DeleteItemOutput *dynamodb.DeleteItemOutput
	QueryOutput      *dynamodb.QueryOutput
	ScanOutput       *dynamodb.ScanOutput

	// Error responses
	GetItemError    error
	PutItemError    error
	UpdateItemError error
	DeleteItemError error
	QueryError      error
	ScanError       error

	// Capture inputs for verification
	LastGetItemInput    *dynamodb.GetItemInput
	LastPutItemInput    *dynamodb.PutItemInput
	LastUpdateItemInput *dynamodb.UpdateItemInput
	LastDeleteItemInput *dynamodb.DeleteItemInput
	LastQueryInput      *dynamodb.QueryInput
	LastScanInput       *dynamodb.ScanInput
}

// Implement the DynamoDB interface methods...
