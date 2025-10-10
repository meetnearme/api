package dynamodb_service

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func setupTestDB(t *testing.T) {
	// Ensure we're in test mode
	os.Setenv("GO_ENV", "test")
}

func TestGetItem(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(t *testing.T) *test_helpers.MockDynamoDBClient
		input       *dynamodb.GetItemInput
		want        *dynamodb.GetItemOutput
		wantErr     bool
		errContains string
	}{
		{
			name: "successful get item",
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						if params == nil {
							return nil, errors.New("nil params")
						}
						if params.TableName == nil {
							return nil, errors.New("nil table name")
						}
						if *params.TableName != "test-table" {
							t.Errorf("expected table name 'test-table', got %s", *params.TableName)
						}
						return &dynamodb.GetItemOutput{
							Item: map[string]types.AttributeValue{
								"id":   &types.AttributeValueMemberS{Value: "test-id"},
								"name": &types.AttributeValueMemberS{Value: "test-name"},
							},
						}, nil
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input: &dynamodb.GetItemInput{
				TableName: aws.String("test-table"),
				Key: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: "test-id"},
				},
			},
			want: &dynamodb.GetItemOutput{
				Item: map[string]types.AttributeValue{
					"id":   &types.AttributeValueMemberS{Value: "test-id"},
					"name": &types.AttributeValueMemberS{Value: "test-name"},
				},
			},
			wantErr: false,
		},
		{
			name: "nil input",
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						if params == nil {
							return nil, errors.New("invalid input: nil")
						}
						return nil, nil
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input:       nil,
			want:        nil,
			wantErr:     true,
			errContains: "invalid input",
		},
		{
			name: "empty table name",
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return nil, errors.New("missing required parameter TableName")
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input: &dynamodb.GetItemInput{
				TableName: aws.String(""),
				Key: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: "test-id"},
				},
			},
			want:        nil,
			wantErr:     true,
			errContains: "missing required parameter",
		},
		{
			name: "item not found",
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return &dynamodb.GetItemOutput{
							Item: nil,
						}, nil
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input: &dynamodb.GetItemInput{
				TableName: aws.String("test-table"),
				Key: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: "non-existent"},
				},
			},
			want: &dynamodb.GetItemOutput{
				Item: nil,
			},
			wantErr: false,
		},
		{
			name: "connection failure",
			setupMock: func(t *testing.T) *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
						return nil, errors.New("connection failed")
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input: &dynamodb.GetItemInput{
				TableName: aws.String("test-table"),
			},
			want:        nil,
			wantErr:     true,
			errContains: "connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDB(t)
			mockDB := tt.setupMock(t)
			if mockDB == nil {
				t.Fatal("mockDB is nil")
			}

			// Get the DB from transport package
			db := transport.GetDB()
			if db == nil {
				t.Fatal("db is nil")
			}

			got, err := db.GetItem(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("GetItem() expected error but got nil")
					return
				}
				if !errorContains(err.Error(), tt.errContains) {
					t.Errorf("GetItem() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutItem(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *test_helpers.MockDynamoDBClient
		input       *dynamodb.PutItemInput
		want        *dynamodb.PutItemOutput
		wantErr     bool
		errContains string
	}{
		{
			name: "successful put item",
			setupMock: func() *test_helpers.MockDynamoDBClient {
				mockDB := &test_helpers.MockDynamoDBClient{
					PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
						if *params.TableName != "test-table" {
							t.Errorf("expected table name 'test-table', got %s", *params.TableName)
						}
						return &dynamodb.PutItemOutput{}, nil
					},
				}
				transport.SetTestDB(mockDB)
				return mockDB
			},
			input: &dynamodb.PutItemInput{
				TableName: aws.String("test-table"),
				Item: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: "test-id"},
				},
			},
			want:    &dynamodb.PutItemOutput{},
			wantErr: false,
		},
		// Add more test cases...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDB(t)
			tt.setupMock()

			// Get the DB from transport package
			db := transport.GetDB()

			got, err := db.PutItem(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if an error message contains a substring
func errorContains(got, want string) bool {
	if want == "" {
		return true
	}
	return strings.Contains(got, want)
}

func TestUpdateItem(t *testing.T) {
	// Test cases for UpdateItem operations
}

func TestDeleteItem(t *testing.T) {
	// Test cases for DeleteItem operations
}

func TestQuery(t *testing.T) {
	// Test cases for Query operations
}

func TestScan(t *testing.T) {
	// Test cases for Scan operations
}

func TestTransactionWriteItems(t *testing.T) {
	// Test cases for transaction operations
}
