package test_helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/meetnearme/api/functions/gateway/types"
)

type MockDynamoDBClient struct {
	ScanFunc       func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	PutItemFunc    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItemFunc    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItemFunc func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItemFunc func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	QueryFunc      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) // New Query method
}

func (m *MockDynamoDBClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	return m.ScanFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.PutItemFunc != nil {
		return m.PutItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return m.GetItemFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return m.QueryFunc(ctx, params, optFns...)
}

func (m *MockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if m.UpdateItemFunc != nil {
		return m.UpdateItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

func (m *MockDynamoDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.DeleteItemFunc != nil {
		return m.DeleteItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

// MockGeoService
type MockGeoService struct {
	GetGeoFunc func(location, baseUrl string) (string, string, string, error)
}

func (m *MockGeoService) GetGeo(location, baseUrl string) (string, string, string, error) {
	return "40.7128", "-74.0060", "New York, NY 10001, USA", nil
}

// MochSeshuService mocks the UpdateSeshuSession function
type MockSeshuService struct{}

func (m *MockSeshuService) UpdateSeshuSession(ctx context.Context, db types.DynamoDBAPI, update types.SeshuSessionUpdate) (*types.SeshuSessionUpdate, error) {
	return &update, nil
}

func (m *MockSeshuService) GetSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionGet) (*types.SeshuSession, error) {
	// Return mock data
	return &types.SeshuSession{
		OwnerId: "mockOwner",
		Url:     seshuPayload.Url,
		Status:  "draft",
		// Fill in other fields as needed
	}, nil
}

func (m *MockSeshuService) InsertSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionInput) (*types.SeshuSessionInsert, error) {
	// Return mock data
	return &types.SeshuSessionInsert{
		OwnerId: seshuPayload.OwnerId,
		Url:     seshuPayload.Url,
		Status:  "draft",
		// Fill in other fields as needed
	}, nil
}

// MockTemplateRenderer mocks the template rendering process
type MockTemplateRenderer struct{}

func (m *MockTemplateRenderer) Render(ctx context.Context, buf *bytes.Buffer) error {
	// Simulate rendering by writing a mock HTML string to the buffer
	_, err := buf.WriteString("<div>Mock rendered template</div>")
	return err
}

var PortCounter int32 = 8000

// NOTE: this is due to an issue where github auto paralellizes these
// test to run in serial, which causes port collisions
func GetNextPort() string {
	port := atomic.AddInt32(&PortCounter, 1)
	return fmt.Sprintf("localhost:%d", port)
}

// Go tests run in parallel by default, which causes port collisions
// This function binds to a port and returns a listener and adds a
// retry mechanism to rotate and attempt to prevent collision
func BindToPort(t *testing.T, endpoint string) (net.Listener, error) {
	var listener net.Listener
	var err error
	currentEndpoint := endpoint

	for retries := 0; retries < 10; retries++ {
		// Strip any http:// prefix if present
		hostPort := currentEndpoint
		if strings.HasPrefix(hostPort, "http://") {
			hostPort = hostPort[len("http://"):]
		}

		t.Logf("Attempting to bind to: %s", hostPort)

		// Try to connect to the port first to check if it's in use
		client := &http.Client{
			Timeout: 100 * time.Millisecond,
		}

		_, err := client.Get(fmt.Sprintf("http://%s", hostPort))
		if err != nil {
			// If we get a connection refused error, the port is available
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "connect: connection refused") {
				// Try to bind to the port
				listener, err = net.Listen("tcp", hostPort)
				if err == nil {
					t.Logf("Successfully bound to: %s", hostPort)
					return listener, nil
				}
			}
			t.Logf("Port %s has existing server or other issue: %v", hostPort, err)
		} else {
			t.Logf("Port %s is already in use", hostPort)
		}

		currentEndpoint = GetNextPort()
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("failed to bind to port after 10 retries: %v", err)
}

type MockRdsDataClient struct {
	ExecStatementFunc func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error)
}

func (m *MockRdsDataClient) ExecStatement(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
	if m.ExecStatementFunc != nil {
		return m.ExecStatementFunc(ctx, sql, params)
	}
	return nil, nil
}

func NewMockRdsDataClientWithJSONRecords(records []map[string]interface{}) *MockRdsDataClient {
	recordsJSON, _ := json.Marshal(records)
	recordsString := string(recordsJSON)
	return &MockRdsDataClient{
		ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
			// Convert records to AWS SDK's expected format
			return &rdsdata.ExecuteStatementOutput{
				FormattedRecords: &recordsString, // Directly return JSON bytes
			}, nil
		},
	}
}

// Ensure MockRdsDataClient implements the RDSDataAPI interface
var _ types.RDSDataAPI = (*MockRdsDataClient)(nil)
