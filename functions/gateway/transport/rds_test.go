// transport_test.go
package transport

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

// MockRDSDataClient is a mock implementation of the RDSDataAPI interface for testing
type MockRDSDataClient struct {
	ExecStatementFunc func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error)
}

func (m *MockRDSDataClient) ExecStatement(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
	if m.ExecStatementFunc != nil {
		return m.ExecStatementFunc(ctx, sql, params)
	}
	return nil, nil
}

func TestCreateRDSClient(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("AWS_REGION", "us-west-2")
	os.Setenv("GO_ENV", "test")
	os.Setenv("RDS_CLUSTER_ARN", "test-cluster-arn")
	os.Setenv("RDS_SECRET_ARN", "test-secret-arn")
	os.Setenv("DATABASE_NAME", "test-db")

	client := CreateRDSClient()
	if client == nil {
		t.Fatal("Expected client to be initialized, got nil")
	}
	if _, ok := client.(*RDSDataClient); !ok {
		t.Fatalf("Expected client to be of type *RDSDataClient, got %T", client)
	}
}

func TestExecStatement(t *testing.T) {
	// Create a mock RDS client
	mockClient := &MockRDSDataClient{
		ExecStatementFunc: func(ctx context.Context, sql string, params []rds_types.SqlParameter) (*rdsdata.ExecuteStatementOutput, error) {
			if sql != "SELECT * FROM test_table" {
				return nil, nil
			}
			return &rdsdata.ExecuteStatementOutput{}, nil
		},
	}
	SetTestRdsDB(mockClient)

	// Call the ExecStatement method
	output, err := GetRdsDB().ExecStatement(context.Background(), "SELECT * FROM test_table", nil)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if output == nil {
		t.Fatal("Expected output to be non-nil")
	}
}

func TestGetRdsDB_ReturnsMockClient(t *testing.T) {
	os.Setenv("GO_ENV", "test")
	mockClient := &MockRDSDataClient{}
	SetTestRdsDB(mockClient)

	client := GetRdsDB()
	if client == nil {
		t.Fatal("Expected client to be initialized, got nil")
	}
	if client != mockClient {
		t.Fatalf("Expected client to be the mock client, got %v", client)
	}
}

func TestGetRdsDB_ReturnsRealClient(t *testing.T) {
	// Set environment variables for real client creation
	os.Setenv("AWS_REGION", "us-west-2")
	os.Setenv("GO_ENV", "production")
	os.Setenv("RDS_CLUSTER_ARN", "test-cluster-arn")
	os.Setenv("RDS_SECRET_ARN", "test-secret-arn")
	os.Setenv("DATABASE_NAME", "test-db")

	client := GetRdsDB()
	if client == nil {
		t.Fatal("Expected client to be initialized, got nil")
	}
	if _, ok := client.(*RDSDataClient); !ok {
		t.Fatalf("Expected client to be of type *RDSDataClient, got %T", client)
	}
}


