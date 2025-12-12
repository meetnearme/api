package test_helpers

import (
	"bytes"
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestMockDynamoDBClient_Scan(t *testing.T) {
	mockClient := &MockDynamoDBClient{
		ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{
				{"ID": &types.AttributeValueMemberS{Value: "item1"}},
			}}, nil
		},
	}

	result, err := mockClient.Scan(context.Background(), &dynamodb.ScanInput{})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestMockDynamoDBClient_PutItem(t *testing.T) {
	mockClient := &MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	_, err := mockClient.PutItem(context.Background(), &dynamodb.PutItemInput{})
	if err != nil {
		t.Fatalf("PutItem failed: %v", err)
	}
}

func TestMockDynamoDBClient_GetItem(t *testing.T) {
	mockClient := &MockDynamoDBClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"ID": &types.AttributeValueMemberS{Value: "item1"},
			}}, nil
		},
	}

	result, err := mockClient.GetItem(context.Background(), &dynamodb.GetItemInput{})
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if result.Item == nil {
		t.Error("expected item to be found, got nil")
	}
}

func TestMockDynamoDBClient_DeleteItem(t *testing.T) {
	mockClient := &MockDynamoDBClient{
		DeleteItemFunc: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}

	_, err := mockClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{})
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
}

func TestMockDynamoDBClient_UpdateItem(t *testing.T) {
	mockClient := &MockDynamoDBClient{
		UpdateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}

	_, err := mockClient.UpdateItem(context.Background(), &dynamodb.UpdateItemInput{})
	if err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}
}

func TestMockGeoService_GetGeo(t *testing.T) {

	mockGeoService := &MockGeoService{}

	lat, long, address, err := mockGeoService.GetGeo("New York", "http://example.com")
	if err != nil {
		t.Fatalf("GetGeo failed: %v", err)
	}

	if lat != "40.7128" || long != "-74.0060" || address != "New York, NY 10001, USA" {
		t.Errorf("unexpected geo values: lat=%s, long=%s, address=%s", lat, long, address)
	}
}

func TestMockSeshuService_UpdateSeshuSession(t *testing.T) {
	mockSeshuService := &MockSeshuService{}

	update := internal_types.SeshuSessionUpdate{Url: "http://example.com"}
	result, err := mockSeshuService.UpdateSeshuSession(context.Background(), nil, update)
	if err != nil {
		t.Fatalf("UpdateSeshuSession failed: %v", err)
	}

	if result.Url != "http://example.com" {
		t.Errorf("expected URL to be %s, got %s", "http://example.com", result.Url)
	}
}

func TestMockSeshuService_GetSeshuSession(t *testing.T) {
	mockSeshuService := &MockSeshuService{}

	payload := internal_types.SeshuSessionGet{Url: "http://example.com"}
	result, err := mockSeshuService.GetSeshuSession(context.Background(), nil, payload)
	if err != nil {
		t.Fatalf("GetSeshuSession failed: %v", err)
	}

	if result.OwnerId != "mockOwner" {
		t.Errorf("expected OwnerId to be %s, got %s", "mockOwner", result.OwnerId)
	}
}

func TestMockTemplateRenderer_Render(t *testing.T) {
	mockRenderer := &MockTemplateRenderer{}
	buf := new(bytes.Buffer)

	err := mockRenderer.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if buf.String() != "<div>Mock rendered template</div>" {
		t.Errorf("unexpected output: %s", buf.String())
	}
}

func TestMockRdsDataClient_ExecStatement(t *testing.T) {
	mockClient := &MockRdsDataClient{
		ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
			return &rdsdata.ExecuteStatementOutput{}, nil
		},
	}

	_, err := mockClient.ExecStatement(context.Background(), "SELECT * FROM test", nil)
	if err != nil {
		t.Fatalf("ExecStatement failed: %v", err)
	}
}

func TestNewMockRdsDataClientWithJSONRecords(t *testing.T) {
	records := []map[string]interface{}{
		{"ID": "item1"},
		{"ID": "item2"},
	}

	mockClient := NewMockRdsDataClientWithJSONRecords(records)
	result, err := mockClient.ExecStatement(context.Background(), "SELECT * FROM test", nil)
	if err != nil {
		t.Fatalf("ExecStatement failed: %v", err)
	}

	if result.FormattedRecords == nil {
		t.Error("expected formatted records to be returned, got nil")
	}
}
