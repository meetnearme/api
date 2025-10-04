package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/nats-io/nats.go/jetstream"
)

type contextKey string

const (
	PostgresKey contextKey = "postgresService"
	NatsKey     contextKey = "natsService"
)

type PostgresService interface {
	GetSeshuJobs(ctx context.Context) ([]internal_types.SeshuJob, error)
	CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error
	UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error
	DeleteSeshuJob(ctx context.Context, id string) error
	ScanSeshuJobsWithInHour(ctx context.Context, hour int) ([]internal_types.SeshuJob, error)
}

type NatsService interface {
	PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishMsg(ctx context.Context, job interface{}) error
}

type MockPostgresService struct {
	GetJobsFunc   func(ctx context.Context) ([]internal_types.SeshuJob, error)
	CreateJobFunc func(ctx context.Context, job internal_types.SeshuJob) error
	UpdateJobFunc func(ctx context.Context, job internal_types.SeshuJob) error
	DeleteJobFunc func(ctx context.Context, id string) error
	ScanJobsFunc  func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error)
}

func (m *MockPostgresService) GetSeshuJobs(ctx context.Context) ([]internal_types.SeshuJob, error) {
	if m == nil || m.GetJobsFunc == nil {
		return nil, nil
	}
	return m.GetJobsFunc(ctx)
}
func (m *MockPostgresService) CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	if m == nil || m.CreateJobFunc == nil {
		return nil
	}
	return m.CreateJobFunc(ctx, job)
}
func (m *MockPostgresService) UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	if m == nil || m.UpdateJobFunc == nil {
		return nil
	}
	return m.UpdateJobFunc(ctx, job)
}
func (m *MockPostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	if m == nil || m.DeleteJobFunc == nil {
		return nil
	}
	return m.DeleteJobFunc(ctx, id)
}
func (m *MockPostgresService) ScanSeshuJobsWithInHour(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
	if m == nil || m.ScanJobsFunc == nil {
		return nil, nil
	}
	return m.ScanJobsFunc(ctx, hour)
}

func (m *MockPostgresService) Close() error {
	return nil
}

type MockNatsService struct {
	PeekTopFunc func(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishFunc func(ctx context.Context, job interface{}) error
	ConsumeFunc func(ctx context.Context, workers int) error
	CloseFunc   func() error
}

func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	if m.PeekTopFunc == nil {
		return nil, nil
	}
	return m.PeekTopFunc(ctx)
}

func (m *MockNatsService) PublishMsg(ctx context.Context, job interface{}) error {
	if m.PublishFunc == nil {
		return nil
	}
	return m.PublishFunc(ctx, job)
}

func (m *MockNatsService) ConsumeMsg(ctx context.Context, workers int) error {
	if m.ConsumeFunc == nil {
		return nil
	}
	return m.ConsumeFunc(ctx, workers)
}

func (m *MockNatsService) Close() error {
	if m.CloseFunc == nil {
		return nil
	}
	return m.CloseFunc()
}

// --- Inject Mock Services into Context ---

func injectMockServices(db PostgresService, nats NatsService) context.Context {
	ctx := context.Background()
	if db != nil {
		ctx = context.WithValue(ctx, PostgresKey, db)
	}
	if nats != nil {
		ctx = context.WithValue(ctx, NatsKey, nats)
	}
	return ctx
}

// --- Tests ---

func newAPIGatewayContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
}

func TestGetSeshuJobs(t *testing.T) {
	mockDB := &MockPostgresService{
		GetJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{{NormalizedUrlKey: "test-key"}}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/seshujob", nil)

	// Inject mock service via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestGetSeshuJobs_DBError(t *testing.T) {
	mockDB := &MockPostgresService{
		GetJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return nil, errors.New("db failure")
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/seshujob", nil)

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Failed to retrieve jobs") {
		t.Errorf("Expected error message in response, got %s", body)
	}
}

func TestCreateSeshuJob(t *testing.T) {
	job := internal_types.SeshuJob{
		NormalizedUrlKey:         "event-123",
		LocationLatitude:         1.3521,
		LocationLongitude:        103.8198,
		LocationAddress:          "Singapore",
		ScheduledHour:            15,
		TargetNameCSSPath:        ".event-title",
		TargetLocationCSSPath:    ".event-location",
		TargetStartTimeCSSPath:   ".event-start",
		TargetEndTimeCSSPath:     ".event-end",
		TargetDescriptionCSSPath: ".event-desc",
		TargetHrefCSSPath:        "a.event-link",
		Status:                   "HEALTHY",
		LastScrapeSuccess:        time.Now().Unix(),
		LastScrapeFailure:        0,
		LastScrapeFailureCount:   0,
		OwnerID:                  "owner-456",
		KnownScrapeSource:        "MEETUP",
	}

	body, _ := json.Marshal(job)

	mockDB := &MockPostgresService{
		CreateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/seshujob", bytes.NewReader(body))

	// Inject mock service via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d", w.Result().StatusCode)
	}
}

func TestCreateSeshuJob_DBError(t *testing.T) {
	job := internal_types.SeshuJob{
		NormalizedUrlKey:         "event-123",
		LocationLatitude:         1.3521,
		LocationLongitude:        103.8198,
		LocationAddress:          "Singapore",
		ScheduledHour:            15,
		TargetNameCSSPath:        ".event-title",
		TargetLocationCSSPath:    ".event-location",
		TargetStartTimeCSSPath:   ".event-start",
		TargetEndTimeCSSPath:     ".event-end",
		TargetDescriptionCSSPath: ".event-desc",
		TargetHrefCSSPath:        "a.event-link",
		Status:                   "HEALTHY",
		LastScrapeSuccess:        time.Now().Unix(),
		LastScrapeFailure:        0,
		LastScrapeFailureCount:   0,
		OwnerID:                  "owner-456",
		KnownScrapeSource:        "MEETUP",
	}

	body, _ := json.Marshal(job)

	mockDB := &MockPostgresService{
		CreateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return errors.New("mock DB error")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/seshujob", bytes.NewReader(body))

	// Inject mock service via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	// Check that the response contains the error message
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Failed to insert job") {
		t.Errorf("Expected response to contain 'Failed to insert job', got %s", responseBody)
	}
}

func TestCreateSeshuJob_InvalidJSON(t *testing.T) {
	mockDB := &MockPostgresService{}
	req := httptest.NewRequest(http.MethodPost, "/api/seshujob", bytes.NewBufferString("not-json"))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Invalid JSON payload") {
		t.Errorf("Expected invalid JSON error, got %s", w.Body.String())
	}
}

func TestCreateSeshuJob_DuplicateKey(t *testing.T) {
	job := internal_types.SeshuJob{
		NormalizedUrlKey: "event-duplicate",
	}
	body, _ := json.Marshal(job)

	mockDB := &MockPostgresService{
		CreateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return errors.New("duplicate key value violates unique constraint \"seshujobs_pkey\"")
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/seshujob", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "That URL is already owned by another user") {
		t.Errorf("Expected duplicate key error message, got %s", w.Body.String())
	}
}
func TestUpdateSeshuJob(t *testing.T) {
	job := internal_types.SeshuJob{
		NormalizedUrlKey:         "event-456",
		LocationLatitude:         1.3,
		LocationLongitude:        103.8,
		LocationAddress:          "Singapore Update",
		ScheduledHour:            10,
		TargetNameCSSPath:        ".name",
		TargetLocationCSSPath:    ".location",
		TargetStartTimeCSSPath:   ".start",
		TargetEndTimeCSSPath:     ".end",
		TargetDescriptionCSSPath: ".desc",
		TargetHrefCSSPath:        ".link",
		Status:                   "HEALTHY",
		LastScrapeSuccess:        time.Now().Unix(),
		LastScrapeFailure:        0,
		LastScrapeFailureCount:   0,
		OwnerID:                  "owner-update",
		KnownScrapeSource:        "EVENTBRITE",
	}

	body, _ := json.Marshal(job)

	mockDB := &MockPostgresService{
		UpdateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPut, "/api/seshujob", bytes.NewReader(body))

	// Inject mock service via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestUpdateSeshuJob_InvalidJSON(t *testing.T) {
	mockDB := &MockPostgresService{}
	req := httptest.NewRequest(http.MethodPut, "/api/seshujob", bytes.NewBufferString("oops"))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Invalid JSON payload") {
		t.Errorf("Expected invalid JSON message, got %s", w.Body.String())
	}
}

func TestUpdateSeshuJob_DBError(t *testing.T) {
	job := internal_types.SeshuJob{NormalizedUrlKey: "event-update"}
	body, _ := json.Marshal(job)

	mockDB := &MockPostgresService{
		UpdateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return errors.New("update failed")
		},
	}
	req := httptest.NewRequest(http.MethodPut, "/api/seshujob", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Failed to update job") {
		t.Errorf("Expected update failure message, got %s", w.Body.String())
	}
}

func TestDeleteSeshuJob(t *testing.T) {
	mockDB := &MockPostgresService{
		DeleteJobFunc: func(ctx context.Context, id string) error {
			if id != "abc123" {
				t.Errorf("Unexpected ID: %s", id)
			}
			return nil
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/seshujob?id=abc123", nil)

	// Inject mock service via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestDeleteSeshuJob_MissingID(t *testing.T) {
	mockDB := &MockPostgresService{}
	req := httptest.NewRequest(http.MethodDelete, "/api/seshujob", nil)

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Missing 'id' query parameter") && !strings.Contains(body, "Missing &#39;id&#39; query parameter") {
		t.Errorf("Expected missing id error, got %s", body)
	}
}

func TestDeleteSeshuJob_DBError(t *testing.T) {
	mockDB := &MockPostgresService{
		DeleteJobFunc: func(ctx context.Context, id string) error {
			return errors.New("delete broke")
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/seshujob?id=abc123", nil)

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Failed to delete job") {
		t.Errorf("Expected delete failure message, got %s", w.Body.String())
	}
}

func TestGatherSeshuJobsHandler(t *testing.T) {
	// Reset global state to avoid interference from other tests
	originalLastExecutionTime := handlers.GetLastExecutionTime()
	defer handlers.SetLastExecutionTime(originalLastExecutionTime)
	handlers.SetLastExecutionTime(0) // Reset to 0 for this test

	mockDB := &MockPostgresService{
		ScanJobsFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{
				{
					NormalizedUrlKey:         "event-gather",
					LocationLatitude:         1.3,
					LocationLongitude:        103.8,
					LocationAddress:          "SG",
					ScheduledHour:            hour,
					TargetNameCSSPath:        ".name",
					TargetLocationCSSPath:    ".loc",
					TargetStartTimeCSSPath:   ".start",
					TargetEndTimeCSSPath:     ".end",
					TargetDescriptionCSSPath: ".desc",
					TargetHrefCSSPath:        ".href",
					Status:                   "HEALTHY",
					LastScrapeSuccess:        time.Now().Unix(),
					LastScrapeFailure:        0,
					LastScrapeFailureCount:   0,
					OwnerID:                  "owner-gather",
					KnownScrapeSource:        "MEETUP",
				},
			}, nil
		},
	}
	mockNats := &MockNatsService{
		PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
			return nil, nil
		},
		PublishFunc: func(ctx context.Context, job interface{}) error {
			return nil
		},
	}

	trigger := handlers.TriggerRequest{Time: time.Now().Unix()}
	body, _ := json.Marshal(trigger)

	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	// Inject mock services via context
	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)

	// Add AWS Lambda context (required for transport layer)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestGatherSeshuJobsHandler_InvalidJSON(t *testing.T) {
	mockDB := &MockPostgresService{}
	mockNats := &MockNatsService{}

	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewBufferString("{"))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Invalid JSON payload") {
		t.Errorf("Expected invalid JSON error, got %s", w.Body.String())
	}
}

func TestGatherSeshuJobsHandler_PeekError(t *testing.T) {
	mockDB := &MockPostgresService{}
	mockNats := &MockNatsService{
		PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
			return nil, errors.New("peek failed")
		},
	}

	trigger := handlers.TriggerRequest{Time: time.Now().Unix()}
	body, _ := json.Marshal(trigger)
	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Failed to get top of NATS queue") {
		t.Errorf("Expected peek error message, got %s", w.Body.String())
	}
}

func TestGatherSeshuJobsHandler_ScanError(t *testing.T) {
	originalLastExecutionTime := handlers.GetLastExecutionTime()
	defer handlers.SetLastExecutionTime(originalLastExecutionTime)
	handlers.SetLastExecutionTime(0)

	mockDB := &MockPostgresService{
		ScanJobsFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			return nil, errors.New("scan blew up")
		},
	}
	mockNats := &MockNatsService{}

	trigger := handlers.TriggerRequest{Time: time.Unix(0, 0).Add(2 * time.Hour).Unix()}
	body, _ := json.Marshal(trigger)
	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Unable to obtain Jobs") {
		t.Errorf("Expected scan error message, got %s", w.Body.String())
	}
}

func TestGatherSeshuJobsHandler_SkipDueToRecentExecution(t *testing.T) {
	originalLastExecutionTime := handlers.GetLastExecutionTime()
	defer handlers.SetLastExecutionTime(originalLastExecutionTime)

	currentTime := time.Now().Unix()
	handlers.SetLastExecutionTime(currentTime - 30)

	var scanCalled bool
	mockDB := &MockPostgresService{
		ScanJobsFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			scanCalled = true
			return nil, nil
		},
	}
	var publishCalled bool
	mockNats := &MockNatsService{
		PublishFunc: func(ctx context.Context, job interface{}) error {
			publishCalled = true
			return nil
		},
	}

	trigger := handlers.TriggerRequest{Time: currentTime}
	body, _ := json.Marshal(trigger)
	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if w.Body.Len() != 0 {
		t.Errorf("Expected empty body for skipped execution, got %s", w.Body.String())
	}

	if scanCalled {
		t.Errorf("Scan should not be called when skipping execution")
	}

	if publishCalled {
		t.Errorf("Publish should not be called when skipping execution")
	}
}

func TestGatherSeshuJobsHandler_InvalidQueuePayload(t *testing.T) {
	originalLastExecutionTime := handlers.GetLastExecutionTime()
	defer handlers.SetLastExecutionTime(originalLastExecutionTime)
	handlers.SetLastExecutionTime(0)

	mockDB := &MockPostgresService{}
	mockNats := &MockNatsService{
		PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
			return &jetstream.RawStreamMsg{Data: []byte("not-json")}, nil
		},
	}

	trigger := handlers.TriggerRequest{Time: time.Now().Add(time.Hour).Unix()}
	body, _ := json.Marshal(trigger)
	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if !strings.Contains(w.Body.String(), "Invalid JSON payload") {
		t.Errorf("Expected invalid queue payload message, got %s", w.Body.String())
	}
}

func TestGatherSeshuJobsHandler_PublishError(t *testing.T) {
	originalLastExecutionTime := handlers.GetLastExecutionTime()
	defer handlers.SetLastExecutionTime(originalLastExecutionTime)
	handlers.SetLastExecutionTime(0)

	mockDB := &MockPostgresService{
		ScanJobsFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{{NormalizedUrlKey: "event1"}}, nil
		},
	}
	publishAttempts := 0
	mockNats := &MockNatsService{
		PublishFunc: func(ctx context.Context, job interface{}) error {
			publishAttempts++
			return errors.New("publish fail")
		},
	}

	trigger := handlers.TriggerRequest{Time: time.Now().Add(time.Hour).Unix()}
	body, _ := json.Marshal(trigger)
	req := httptest.NewRequest(http.MethodPost, "/api/gather-seshu-jobs", bytes.NewReader(body))

	ctx := context.WithValue(req.Context(), "mockPostgresService", mockDB)
	ctx = context.WithValue(ctx, "mockNatsService", mockNats)
	ctx = newAPIGatewayContext(ctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Result().StatusCode)
	}

	if publishAttempts != 1 {
		t.Fatalf("Expected publish attempt once, got %d", publishAttempts)
	}

	if !strings.Contains(w.Body.String(), "successful") {
		t.Errorf("Expected successful response despite publish errors, got %s", w.Body.String())
	}
}
