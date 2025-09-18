package test_helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/playwright-community/playwright-go"
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
		EventValidations: []types.EventBoolValid{
			{
				EventValidateTitle:       true,
				EventValidateLocation:    true,
				EventValidateStartTime:   true,
				EventValidateEndTime:     false,
				EventValidateURL:         true,
				EventValidateDescription: false,
			},
		},
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

var (
	PortCounter int32 = 8001
	portMutex   sync.Mutex
	usedPorts   = make(map[string]bool)
)

// GetNextPort returns a unique port that is guaranteed to be available
// Uses a mutex to ensure thread-safe port allocation and prevent collisions
func GetNextPort() string {
	portMutex.Lock()
	defer portMutex.Unlock()

	// Find the next available port
	for {
		port := atomic.AddInt32(&PortCounter, 1)
		portStr := fmt.Sprintf("localhost:%d", port)

		// Check if this port is already marked as used
		if !usedPorts[portStr] {
			// Mark this port as used
			usedPorts[portStr] = true
			return portStr
		}
	}
}

// ReleasePort marks a port as available again
func ReleasePort(port string) {
	portMutex.Lock()
	defer portMutex.Unlock()
	delete(usedPorts, port)
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
				// Release the port since we couldn't bind to it
				ReleasePort(hostPort)
			} else {
				t.Logf("Port %s has existing server or other issue: %v", hostPort, err)
			}
		} else {
			t.Logf("Port %s is already in use (received HTTP response)", hostPort)
			// Release the port since it's already in use
			ReleasePort(hostPort)
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

func SetupStaticTestRouter(t *testing.T, staticDir string) *mux.Router {
	router := mux.NewRouter()

	// Add static file handling
	router.PathPrefix("/assets/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "../../../static/assets"

		// Log the absolute path and directory contents
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Printf("Error getting absolute path: %v", err)
		}
		log.Printf("Serving assets from directory: %s", absPath)

		fileServer := http.FileServer(http.Dir(path))
		http.StripPrefix("/assets/", fileServer).ServeHTTP(w, r)
	})

	return router
}

func ScreenshotToStandardDir(t *testing.T, page playwright.Page, screenshotName string) {
	// Get the path to the project root (where go.mod is)
	moduleRoot, err := os.Getwd()
	for !fileExists(filepath.Join(moduleRoot, "go.mod")) && moduleRoot != "/" {
		moduleRoot = filepath.Dir(moduleRoot)
	}
	if moduleRoot == "/" {
		t.Fatal("Could not find project root (go.mod)")
	}

	// Create the screenshots directory if it doesn't exist
	screenshotsDir := filepath.Join(moduleRoot, "screenshots")
	if _, err := os.Stat(screenshotsDir); os.IsNotExist(err) {
		err = os.MkdirAll(screenshotsDir, 0755)
		if err != nil {
			t.Fatalf("could not create screenshots directory: %v\n", err)
		}
	}

	screenshotPath := filepath.Join(moduleRoot, "screenshots", screenshotName)
	_, err = page.Screenshot(
		playwright.PageScreenshotOptions{
			Path: &screenshotPath,
		},
	)
	if err != nil {
		t.Fatalf("could not screenshot: %v\n", err)
	}

	t.Logf("Screenshot saved to %s", screenshotPath)
}

func GetPlaywrightBrowser() (*playwright.Browser, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	var launchOptions playwright.BrowserTypeLaunchOptions
	// Assuming GitHub Actions, CI is always `true`
	if os.Getenv("CI") == "true" {
		launchOptions = playwright.BrowserTypeLaunchOptions{
			Args: []string{"--no-sandbox"},
		}
	}

	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		return nil, err
	}

	// TODO: we can't stop this because we haven't run tests yet,
	// but we should probably return `pw` and let the caller stop it
	// or have some implicit way for something like this `defer` to happen

	// defer browser.Close()
	// defer pw.Stop()

	return &browser, nil
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Custom HTTP transport that logs all requests
type LoggingTransport struct {
	base http.RoundTripper
	t    *testing.T
}

// NewLoggingTransport creates a new LoggingTransport
func NewLoggingTransport(base http.RoundTripper, t *testing.T) *LoggingTransport {
	return &LoggingTransport{
		base: base,
		t:    t,
	}
}

func (lt *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the outgoing request
	lt.t.Logf("🌐 HTTP REQUEST: %s %s", req.Method, req.URL.String())
	lt.t.Logf("   └─ Host: %s", req.URL.Host)
	lt.t.Logf("   └─ Path: %s", req.URL.Path)
	if req.Body != nil {
		// Read body for logging (need to restore it after)
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			// Log first 200 chars of body
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 200 {
				bodyStr = bodyStr[:200] + "..."
			}
			lt.t.Logf("   └─ Body: %s", bodyStr)
		}
	}

	// Make the actual request
	resp, err := lt.base.RoundTrip(req)

	// Log the response
	if err != nil {
		lt.t.Logf("❌ HTTP ERROR: %v", err)
	} else {
		lt.t.Logf("✅ HTTP RESPONSE: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, err
}

type MockPostgresService struct {
	GetSeshuJobsFunc            func(ctx context.Context) ([]types.SeshuJob, error)
	CreateSeshuJobFunc          func(ctx context.Context, job types.SeshuJob) error
	UpdateSeshuJobFunc          func(ctx context.Context, job types.SeshuJob) error
	DeleteSeshuJobFunc          func(ctx context.Context, id string) error
	ScanSeshuJobsWithInHourFunc func(ctx context.Context, hours int) ([]types.SeshuJob, error)
}

func (m *MockPostgresService) GetSeshuJobs(ctx context.Context) ([]types.SeshuJob, error) {
	if m.GetSeshuJobsFunc != nil {
		return m.GetSeshuJobsFunc(ctx)
	}
	return []types.SeshuJob{}, nil
}

func (m *MockPostgresService) CreateSeshuJob(ctx context.Context, job types.SeshuJob) error {
	if m.CreateSeshuJobFunc != nil {
		return m.CreateSeshuJobFunc(ctx, job)
	}
	return nil
}

func (m *MockPostgresService) UpdateSeshuJob(ctx context.Context, job types.SeshuJob) error {
	if m.UpdateSeshuJobFunc != nil {
		return m.UpdateSeshuJobFunc(ctx, job)
	}
	return nil
}

func (m *MockPostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	if m.DeleteSeshuJobFunc != nil {
		return m.DeleteSeshuJobFunc(ctx, id)
	}
	return nil
}

func (m *MockPostgresService) ScanSeshuJobsWithInHour(ctx context.Context, hours int) ([]types.SeshuJob, error) {
	if m.ScanSeshuJobsWithInHourFunc != nil {
		return m.ScanSeshuJobsWithInHourFunc(ctx, hours)
	}
	return []types.SeshuJob{}, nil
}

func (m *MockPostgresService) Close() error {
	return nil
}

type MockNatsService struct {
	mu             sync.Mutex
	PublishedMsgs  [][]byte // all published messages
	SimulatedQueue [][]byte // messages waiting to be consumed
}

// NewMockNatsService initializes a new mock
func NewMockNatsService() *MockNatsService {
	return &MockNatsService{
		PublishedMsgs:  make([][]byte, 0),
		SimulatedQueue: make([][]byte, 0),
	}
}

// PublishMsg simulates pushing a message to the queue
func (m *MockNatsService) PublishMsg(ctx context.Context, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	m.PublishedMsgs = append(m.PublishedMsgs, data)
	m.SimulatedQueue = append(m.SimulatedQueue, data)
	return nil
}

// PeekTopOfQueue returns the first message without removing it
func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.SimulatedQueue) == 0 {
		return nil, nil
	}
	return m.SimulatedQueue[0], nil
}

// ConsumeMsg processes each message one by one (no concurrency)
func (m *MockNatsService) ConsumeMsg(ctx context.Context) error {
	for {
		m.mu.Lock()
		if len(m.SimulatedQueue) == 0 {
			m.mu.Unlock()
			break
		}
		msg := m.SimulatedQueue[0]
		m.SimulatedQueue = m.SimulatedQueue[1:]
		m.mu.Unlock()

		fmt.Printf("Processing: %s\n", string(msg))
	}
	return nil
}
