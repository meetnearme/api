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
	return m.GetJobsFunc(ctx)
}
func (m *MockPostgresService) CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	return m.CreateJobFunc(ctx, job)
}
func (m *MockPostgresService) UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	return m.UpdateJobFunc(ctx, job)
}
func (m *MockPostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	return m.DeleteJobFunc(ctx, id)
}
func (m *MockPostgresService) ScanSeshuJobsWithInHour(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
	return m.ScanJobsFunc(ctx, hour)
}

func (m *MockPostgresService) Close() error {
	return nil
}

type MockNatsService struct {
	PeekTopFunc func(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishFunc func(ctx context.Context, job interface{}) error
}

func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	return m.PeekTopFunc(ctx)
}

func (m *MockNatsService) PublishMsg(ctx context.Context, job interface{}) error {
	return m.PublishFunc(ctx, job)
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}

func TestGatherSeshuJobsHandler(t *testing.T) {
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
	ctx = context.WithValue(ctx, helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
		PathParameters: map[string]string{},
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler := handlers.GatherSeshuJobsHandler(w, req)
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Result().StatusCode)
	}
}
