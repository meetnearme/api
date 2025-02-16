package test_helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	ScanFunc             func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	PutItemFunc          func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItemFunc          func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItemFunc       func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItemFunc       func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	QueryFunc            func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)                   // New Query method
	BatchWriteItemFunc   func(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) // New Query method
	ExecuteStatementFunc func(ctx context.Context, params *dynamodb.ExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error)
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

func (m *MockDynamoDBClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	if m.BatchWriteItemFunc != nil {
		return m.BatchWriteItemFunc(ctx, params, optFns...)
	}
	// Return empty success response if no mock function is provided
	return &dynamodb.BatchWriteItemOutput{}, nil
}

func (m *MockDynamoDBClient) ExecuteStatement(ctx context.Context, params *dynamodb.ExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error) {
	if m.ExecuteStatementFunc != nil {
		return m.ExecuteStatementFunc(ctx, params, optFns...)
	}
	return &dynamodb.ExecuteStatementOutput{}, nil
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

		// Check for various error conditions that indicate the port is available
		if err != nil {
			isPortAvailable := strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "connect: connection refused") ||
				strings.Contains(err.Error(), "EOF") || // Add EOF as an indicator of available port
				strings.Contains(err.Error(), "no response")

			if isPortAvailable {
				// Try to bind to the port
				listener, err = net.Listen("tcp", hostPort)
				if err == nil {
					t.Logf("Successfully bound to: %s", hostPort)
					return listener, nil
				}
				t.Logf("Failed to bind despite port appearing available: %v", err)
			} else {
				t.Logf("Port %s has existing server or other issue: %v", hostPort, err)
			}
		} else {
			t.Logf("Port %s is already in use (received HTTP response)", hostPort)
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

// PlaywrightTestConfig holds configuration for running Playwright tests
type PlaywrightTestConfig struct {
	TestName   string // Base name of the test (e.g., "home_templ")
	ConfigPath string // Path to playwright.config.js
	TestPath   string // Path to the test file
	TestURL    string // The test server instance
}

// RunPlaywrightTest sets up and runs a Playwright test with the given configuration
func RunPlaywrightTest(t *testing.T, config PlaywrightTestConfig) error {
	// Create temporary directory for test execution
	testDir, err := os.MkdirTemp("", fmt.Sprintf("%s-test", config.TestName))
	if err != nil {
		return fmt.Errorf("failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Copy the test file to temp dir
	testContent, err := os.ReadFile(config.TestPath)
	if err != nil {
		return fmt.Errorf("failed to read test file: %v", err)
	}
	tempTestPath := filepath.Join(testDir, fmt.Sprintf("%s.test.js", config.TestName))
	if err := os.WriteFile(tempTestPath, testContent, 0644); err != nil {
		return fmt.Errorf("failed to write test file: %v", err)
	}

	// Copy the config file to temp dir
	configContent, err := os.ReadFile(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	tempConfigPath := filepath.Join(testDir, "playwright.config.js")
	if err := os.WriteFile(tempConfigPath, configContent, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Create package.json with ES modules support
	packageJSON := `{
		"type": "module",
		"private": true
	}`
	if err := os.WriteFile(filepath.Join(testDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		return fmt.Errorf("failed to create package.json: %v", err)
	}

	// Install dependencies
	installCmd := exec.Command("npm", "install", "@playwright/test", "playwright")
	installCmd.Dir = testDir
	if output, err := installCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install dependencies: %v\n%s", err, output)
	}

	// Execute Playwright tests
	cmd := exec.Command("npx", "playwright", "test",
		tempTestPath,
		"--config", tempConfigPath,
		"--project", "chromium",
		"--reporter", "list",
		"--ignore-snapshots")
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_SERVER_URL=%s", config.TestURL),
		"PLAYWRIGHT_JSON_OUTPUT=true")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("playwright tests failed: %v\n%s", err, output)
	}

	return nil
}

// Helper function to find test files and configs
func GetTestPaths(t *testing.T, filename string) (string, string, error) {
	// Find project root by looking for package.json
	dir := filepath.Dir(filename)
	rootDir := dir
	for {
		if _, err := os.Stat(filepath.Join(rootDir, "package.json")); err == nil {
			break
		}
		parentDir := filepath.Dir(rootDir)
		if parentDir == rootDir {
			return "", "", fmt.Errorf("could not find project root (package.json)")
		}
		rootDir = parentDir
	}

	// Config path (now from project root)
	configPath := filepath.Join(rootDir, "playwright.config.js")
	t.Logf("configPath: %s", configPath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("playwright config not found at %s", configPath)
	}

	// Test file path (relative to current directory)
	baseFileName := strings.TrimSuffix(filepath.Base(filename), "_test.go")
	testPath := filepath.Join(dir, baseFileName+".test.js")
	t.Logf("testPath: %s", testPath)
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("interactive test not found at %s", testPath)
	}

	return configPath, testPath, nil
}
