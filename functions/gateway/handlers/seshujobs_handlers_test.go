package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/handlers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/nats-io/nats.go/jetstream"
)

// --- Mock Services ---

type MockPostgresService struct {
	GetJobsFunc   func(ctx context.Context) ([]internal_types.SeshuJob, error)
	CreateJobFunc func(ctx context.Context, job internal_types.SeshuJob) error
	UpdateJobFunc func(ctx context.Context, job internal_types.SeshuJob) error
	DeleteJobFunc func(ctx context.Context, id string) error
	ScanJobsFunc  func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error)
}

type MockNatsService struct {
	PeekTopFunc func(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishFunc func(ctx context.Context, job interface{}) error
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

func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	return m.PeekTopFunc(ctx)
}
func (m *MockNatsService) PublishMsg(ctx context.Context, job interface{}) error {
	return m.PublishFunc(ctx, job)
}

// --- Tests ---

func TestGetSeshuJobs_Success(t *testing.T) {
	mockDB := &MockPostgresService{
		GetJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{{NormalizedUrlKey: "www.example.com"}}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/seshujob", nil)
	rr := httptest.NewRecorder()

	// Mock service injection
	t.Setenv("MOCK_DB", "1")
	ctx := context.WithValue(req.Context(), "postgresService", mockDB)
	req = req.WithContext(ctx)

	handler := handlers.GetSeshuJobs(rr, req)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "job1") {
		t.Errorf("expected HTML to contain job1, got %s", rr.Body.String())
	}
}

func TestProcessGatherSeshuJobs_SkipDueToCooldown(t *testing.T) {
	now := time.Now().Unix()
	handlers.SetLastExecutionTime(now)

	published, skipped, status, err := handlers.ProcessGatherSeshuJobs(context.Background(), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Errorf("expected skipped=true")
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200")
	}
	if published != 0 {
		t.Errorf("expected 0 published")
	}
}

func TestProcessGatherSeshuJobs_DBError(t *testing.T) {
	ctx := context.Background()
	published, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, time.Now().Unix())
	if err == nil {
		t.Fatalf("expected error due to missing DB")
	}
	if skipped {
		t.Errorf("expected skipped=false")
	}
	if status == http.StatusOK {
		t.Errorf("expected non-200 status")
	}
	if published != 0 {
		t.Errorf("expected 0 published")
	}
}

func TestSetAndGetLastExecutionTime(t *testing.T) {
	before := handlers.GetLastExecutionTime()
	now := time.Now().Unix()
	handlers.SetLastExecutionTime(now)
	after := handlers.GetLastExecutionTime()

	if before == after {
		t.Errorf("expected time to change")
	}
	if after != now {
		t.Errorf("expected %d, got %d", now, after)
	}
}
