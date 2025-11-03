package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/handlers"
	"github.com/meetnearme/api/functions/gateway/interfaces"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/nats-io/nats.go/jetstream"
)

// --- Mock Services ---

type MockPostgresService struct {
	GetSeshuJobsFunc       func(ctx context.Context) ([]internal_types.SeshuJob, error)
	CreateJobFunc          func(ctx context.Context, job internal_types.SeshuJob) error
	UpdateJobFunc          func(ctx context.Context, job internal_types.SeshuJob) error
	DeleteJobFunc          func(ctx context.Context, id string) error
	ScanJobsWithInHourFunc func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error)
}

func (m *MockPostgresService) GetSeshuJobs(ctx context.Context) ([]internal_types.SeshuJob, error) {
	if m.GetSeshuJobsFunc != nil {
		return m.GetSeshuJobsFunc(ctx)
	}
	return []internal_types.SeshuJob{}, nil
}

func (m *MockPostgresService) CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	if m.CreateJobFunc != nil {
		return m.CreateJobFunc(ctx, job)
	}
	return nil
}

func (m *MockPostgresService) UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	if m.UpdateJobFunc != nil {
		return m.UpdateJobFunc(ctx, job)
	}
	return nil
}

func (m *MockPostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	if m.DeleteJobFunc != nil {
		return m.DeleteJobFunc(ctx, id)
	}
	return nil
}

func (m *MockPostgresService) ScanSeshuJobsWithInHour(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
	if m.ScanJobsWithInHourFunc != nil {
		return m.ScanJobsWithInHourFunc(ctx, hour)
	}
	return nil, nil
}

func (m *MockPostgresService) Close() error {
	return nil
}

// Ensure it satisfies the PostgresServiceInterface
var _ interfaces.PostgresServiceInterface = (*MockPostgresService)(nil)

type MockNatsService struct {
	PeekTopFunc func(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishFunc func(ctx context.Context, job interface{}) error
	ConsumeFunc func(ctx context.Context, workers int) error
	CloseFunc   func() error
}

func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	if m.PeekTopFunc != nil {
		return m.PeekTopFunc(ctx)
	}
	return nil, nil
}

func (m *MockNatsService) PublishMsg(ctx context.Context, job interface{}) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, job)
	}
	return nil
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

func setupMockServices(pg *MockPostgresService, nats *MockNatsService) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "mockPostgresService", pg)
	ctx = context.WithValue(ctx, "mockNatsService", nats)
	return ctx
}

// --- Tests ---
func TestGetSeshuJobs_Success(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	// Mock implementation for GetSeshuJobs
	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{
				{
					NormalizedUrlKey: "test-key",
					Status:           "active",
					OwnerID:          "user123",
				},
			}, nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))

	req := httptest.NewRequest(http.MethodGet, "/seshu-jobs", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	t.Logf("Response Body:\n%s", bodyString)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(bodyString, "test-key") {
		t.Errorf("expected response body to contain job key 'test-key'")
	}
}

func TestGetSeshuJobs_DBError(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return nil, errors.New("DB connection failed")
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodGet, "/seshu-jobs", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	t.Logf("Error Response Body:\n%s", bodyString)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected http 200 for partial error handler, got %d", resp.StatusCode)
	}

	if !strings.Contains(bodyString, "Failed to retrieve jobs") {
		t.Errorf("expected error message in response body, got: %s", bodyString)
	}
}

func TestGetSeshuJobs_EmptyList(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{}, nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodGet, "/seshu-jobs", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.GetSeshuJobs(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	t.Logf("Empty Response Body:\n%s", bodyString)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	// No jobs, so it shouldn't contain any "Key:" lines
	if strings.Contains(bodyString, "<strong>Key:</strong>") {
		t.Errorf("expected no job HTML cards, but found one")
	}
}

func TestCreateSeshuJob_Success(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		CreateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			if job.NormalizedUrlKey != "new-key" {
				t.Errorf("expected NormalizedUrlKey=new-key, got %s", job.NormalizedUrlKey)
			}
			return nil
		},
	}

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "new-key",
		Status:           "active",
		OwnerID:          "user123",
	}
	body, _ := json.Marshal(payload)

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPost, "/seshu-jobs", bytes.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("Response: %s", bodyStr)

	// ✅ Expect 201, not 200
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Job created successfully") {
		t.Errorf("expected success message, got: %s", bodyStr)
	}
}

func TestCreateSeshuJob_InvalidJSON(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPost, "/seshu-jobs", strings.NewReader("{invalid-json")).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("Response: %s", bodyStr)

	// ✅ The handler returns 200 OK even for errors
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (error wrapped in HTML), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Invalid JSON payload") {
		t.Errorf("expected 'Invalid JSON payload' in response body")
	}
}

func TestCreateSeshuJob_DBError(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		CreateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return errors.New("DB insert failed")
		},
	}

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "fail-key",
		Status:           "inactive",
		OwnerID:          "user456",
	}
	body, _ := json.Marshal(payload)

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPost, "/seshu-jobs", bytes.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.CreateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("DB Error Response: %s", bodyStr)

	// ✅ The handler uses 200 OK even on failure
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Failed to insert job") {
		t.Errorf("expected DB error message, got: %s", bodyStr)
	}
}

func TestUpdateSeshuJob_Success(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		UpdateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			if job.NormalizedUrlKey != "update-key" {
				t.Errorf("expected NormalizedUrlKey=update-key, got %s", job.NormalizedUrlKey)
			}
			return nil
		},
	}

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "update-key",
		Status:           "done",
	}
	body, _ := json.Marshal(payload)

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPut, "/seshu-jobs/update", bytes.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Job updated successfully") {
		t.Errorf("expected success message, got: %s", bodyStr)
	}
}

func TestUpdateSeshuJob_InvalidJSON(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPut, "/seshu-jobs/update", strings.NewReader("{invalid-json")).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("Invalid JSON Response: %s", bodyStr)

	// Handler likely returns 200 with HTML error
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Invalid JSON payload") {
		t.Errorf("expected 'Invalid JSON payload' in response, got: %s", bodyStr)
	}
}

func TestUpdateSeshuJob_DBError(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{
		UpdateJobFunc: func(ctx context.Context, job internal_types.SeshuJob) error {
			return errors.New("update failed")
		},
	}

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "fail-key",
		Status:           "error",
	}
	body, _ := json.Marshal(payload)

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodPut, "/seshu-jobs/update", bytes.NewReader(body)).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.UpdateSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	bodyStr := string(b)

	t.Logf("DB Error Response: %s", bodyStr)

	// Handler likely returns 200 even on failure
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Failed to update job") {
		t.Errorf("expected DB error message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_Success(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	userId := "user123"
	targetUrl := "https://example.com/events"
	mockJob := internal_types.SeshuJob{
		NormalizedUrlKey: targetUrl,
		OwnerID:          userId,
	}

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			// Verify that targetUrl is in the context
			ctxTargetUrl, ok := ctx.Value("targetUrl").(string)
			if !ok || ctxTargetUrl != targetUrl {
				t.Errorf("expected targetUrl=%s in context, got %v", targetUrl, ctxTargetUrl)
			}
			return []internal_types.SeshuJob{mockJob}, nil
		},
		DeleteJobFunc: func(ctx context.Context, id string) error {
			if id != targetUrl {
				t.Errorf("expected ID=%s, got %s", targetUrl, id)
			}
			return nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: userId})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Job deleted successfully") {
		t.Errorf("expected success message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_InvalidID(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: "user123"})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Invalid ID Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Missing &#39;id&#39; query parameter") {
		t.Errorf("expected 'Missing &#39;id&#39; query parameter' message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_DBError(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	userId := "user123"
	targetUrl := "https://example.com/events/456"
	mockJob := internal_types.SeshuJob{
		NormalizedUrlKey: targetUrl,
		OwnerID:          userId,
	}

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{mockJob}, nil
		},
		DeleteJobFunc: func(ctx context.Context, id string) error {
			return errors.New("delete failed")
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: userId})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("DB Error Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Failed to delete event source URL") {
		t.Errorf("expected DB error message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_MissingUserID(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	mockService := &MockPostgresService{}

	targetUrl := "https://example.com/events/123"
	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Missing User ID Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Missing user ID") {
		t.Errorf("expected 'Missing user ID' message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_JobNotFound(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	targetUrl := "https://example.com/events/999"
	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{}, nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: "user123"})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Job Not Found Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Event source URL not found") {
		t.Errorf("expected 'Event source URL not found' message, got: %s", bodyStr)
	}
	// Verify the ID is not exposed in the error message
	if strings.Contains(bodyStr, targetUrl) {
		t.Errorf("expected ID to not be exposed in error message, but found: %s", targetUrl)
	}
}

func TestDeleteSeshuJob_GetSeshuJobsError(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	targetUrl := "https://example.com/events/999"
	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return nil, errors.New("failed to get jobs")
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: "user123"})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Get Jobs Error Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Internal server error") {
		t.Errorf("expected 'Internal server error' message, got: %s", bodyStr)
	}
	// Verify the ID and error details are not exposed in the error message
	if strings.Contains(bodyStr, targetUrl) {
		t.Errorf("expected ID to not be exposed in error message, but found: %s", targetUrl)
	}
	if strings.Contains(bodyStr, "failed to get jobs") {
		t.Errorf("expected error details to not be exposed in error message")
	}
}

func TestDeleteSeshuJob_Unauthorized(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	userId := "user123"
	otherUserId := "other-user"
	targetUrl := "https://example.com/events/456"
	mockJob := internal_types.SeshuJob{
		NormalizedUrlKey: targetUrl,
		OwnerID:          otherUserId, // Different owner
	}

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{mockJob}, nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: userId})
	ctx = context.WithValue(ctx, "roleClaims", []constants.RoleClaim{}) // No super admin role
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Unauthorized Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 (HTML error), got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "You are not the owner of this event source URL") {
		t.Errorf("expected 'You are not the owner of this event source URL' message, got: %s", bodyStr)
	}
}

func TestDeleteSeshuJob_SuperAdmin(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	userId := "super-admin-user"
	otherUserId := "other-user"
	targetUrl := "https://example.com/events/789"
	mockJob := internal_types.SeshuJob{
		NormalizedUrlKey: targetUrl,
		OwnerID:          otherUserId, // Different owner, but super admin can delete
	}

	mockService := &MockPostgresService{
		GetSeshuJobsFunc: func(ctx context.Context) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{mockJob}, nil
		},
		DeleteJobFunc: func(ctx context.Context, id string) error {
			if id != targetUrl {
				t.Errorf("expected ID=%s, got %s", targetUrl, id)
			}
			return nil
		},
	}

	ctx := context.WithValue(context.Background(), "mockPostgresService", interfaces.PostgresServiceInterface(mockService))
	ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: userId})
	ctx = context.WithValue(ctx, "roleClaims", []constants.RoleClaim{
		{Role: constants.Roles[constants.SuperAdmin]},
	})
	req := httptest.NewRequest(http.MethodDelete, "/seshu-jobs/delete?id="+url.QueryEscape(targetUrl), nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler := handlers.DeleteSeshuJob(w, req)
	handler(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	t.Logf("Super Admin Response: %s", bodyStr)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(bodyStr, "Job deleted successfully") {
		t.Errorf("expected success message, got: %s", bodyStr)
	}
}

func TestProcessGatherSeshuJobs_Success_EmptyQueue(t *testing.T) {
	mockPg := &MockPostgresService{
		ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{
				{NormalizedUrlKey: "job1", ScheduledHour: hour},
				{NormalizedUrlKey: "job2", ScheduledHour: hour},
			}, nil
		},
	}

	mockNats := &MockNatsService{
		PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
			return nil, nil // empty queue
		},
		PublishFunc: func(ctx context.Context, job interface{}) error {
			// Assert that job is the correct type
			if _, ok := job.(internal_types.SeshuJob); !ok {
				t.Errorf("expected SeshuJob type, got %T", job)
			}
			return nil
		},
	}

	ctx := setupMockServices(mockPg, mockNats)
	trigger := time.Now().Unix()

	count, skip, status, err := handlers.ProcessGatherSeshuJobs(ctx, trigger)
	if err != nil || status != http.StatusOK {
		t.Fatalf("expected success, got status %d err %v", status, err)
	}
	if count != 2 || skip {
		t.Errorf("expected count=2 skip=false, got count=%d skip=%v", count, skip)
	}
}

func TestProcessGatherSeshuJobs_PublishFailure(t *testing.T) {
	handlers.SetLastExecutionTime(0)

	mockDB := &MockPostgresService{
		ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
			return []internal_types.SeshuJob{{NormalizedUrlKey: "job1"}}, nil
		},
	}
	mockNats := &MockNatsService{
		PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
			return nil, nil
		},
		PublishFunc: func(ctx context.Context, job interface{}) error {
			return errors.New("failed to publish")
		},
	}

	ctx := setupMockServices(mockDB, mockNats)

	published, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, time.Now().Unix())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if published != 0 {
		t.Errorf("Expected 0 published, got %d", published)
	}
	if skipped {
		t.Errorf("Expected skipped=false, got true")
	}
	if status != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", status)
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

// Additional coverage for ProcessGatherSeshuJobs edge cases and error paths
func TestProcessGatherSeshuJobs_Variants(t *testing.T) {
	os.Setenv("GO_ENV", "test")

	// Use a fixed trigger time to make Hour deterministic (UTC noon)
	fixedTrigger := int64(12 * 3600) // 12:00:00 UTC on epoch day
	currentHour := 12

	t.Run("NonEmptyQueue_CurrentHour_NoScan_NoPublish", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)

		// Head of queue is for the current hour, so we should not scan DB
		head := internal_types.SeshuJob{NormalizedUrlKey: "queued-job", ScheduledHour: currentHour}
		headBytes, _ := json.Marshal(head)

		mockPg := &MockPostgresService{ // if called, fail the test
			ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
				t.Fatalf("ScanSeshuJobsWithInHour should NOT be called when queue head is current hour")
				return nil, nil
			},
		}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
				return &jetstream.RawStreamMsg{Data: headBytes}, nil
			},
			PublishFunc: func(ctx context.Context, job interface{}) error {
				t.Fatalf("PublishMsg should NOT be called when no scan occurs")
				return nil
			},
		}

		ctx := setupMockServices(mockPg, mockNats)
		published, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err != nil || status != http.StatusOK {
			t.Fatalf("expected OK with no publish, got status=%d err=%v", status, err)
		}
		if skipped {
			t.Errorf("did not expect skipped due to cooldown")
		}
		if published != 0 {
			t.Errorf("expected 0 published, got %d", published)
		}
	})

	t.Run("NonEmptyQueue_OlderHour_ScansAndPublishes", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)

		head := internal_types.SeshuJob{NormalizedUrlKey: "old-queued-job", ScheduledHour: currentHour - 1}
		headBytes, _ := json.Marshal(head)

		jobs := []internal_types.SeshuJob{{NormalizedUrlKey: "jobA"}, {NormalizedUrlKey: "jobB"}}
		publishCount := 0

		mockPg := &MockPostgresService{
			ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
				if hour != currentHour {
					t.Errorf("expected scan hour=%d, got %d", currentHour, hour)
				}
				return jobs, nil
			},
		}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
				return &jetstream.RawStreamMsg{Data: headBytes}, nil
			},
			PublishFunc: func(ctx context.Context, job interface{}) error {
				publishCount++
				return nil
			},
		}

		ctx := setupMockServices(mockPg, mockNats)
		published, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err != nil || status != http.StatusOK {
			t.Fatalf("expected OK, got status=%d err=%v", status, err)
		}
		if skipped {
			t.Errorf("did not expect skipped")
		}
		if published != len(jobs) || publishCount != len(jobs) {
			t.Errorf("expected %d published, got %d (publishCount=%d)", len(jobs), published, publishCount)
		}
	})

	t.Run("PeekTopOfQueue_Error", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)
		mockPg := &MockPostgresService{}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
				return nil, errors.New("peek error")
			},
		}
		ctx := setupMockServices(mockPg, mockNats)
		_, _, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err == nil || status != http.StatusBadRequest {
			t.Fatalf("expected 400 with error, got status=%d err=%v", status, err)
		}
	})

	t.Run("TopOfQueue_InvalidJSON", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)
		mockPg := &MockPostgresService{}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
				return &jetstream.RawStreamMsg{Data: []byte("{bad json}")}, nil
			},
		}
		ctx := setupMockServices(mockPg, mockNats)
		_, _, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err == nil || status != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid JSON, got status=%d err=%v", status, err)
		}
	})

	t.Run("ScanError_EmptyQueue", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)
		mockPg := &MockPostgresService{
			ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
				return nil, errors.New("scan error")
			},
		}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) { return nil, nil },
		}
		ctx := setupMockServices(mockPg, mockNats)
		_, _, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err == nil || status != http.StatusBadRequest {
			t.Fatalf("expected 400 for scan error, got status=%d err=%v", status, err)
		}
	})

	t.Run("ScanError_WithOlderQueueHead", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)
		head := internal_types.SeshuJob{NormalizedUrlKey: "old", ScheduledHour: currentHour - 1}
		headBytes, _ := json.Marshal(head)
		mockPg := &MockPostgresService{
			ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
				return nil, errors.New("scan error")
			},
		}
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
				return &jetstream.RawStreamMsg{Data: headBytes}, nil
			},
		}
		ctx := setupMockServices(mockPg, mockNats)
		_, _, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err == nil || status != http.StatusBadRequest {
			t.Fatalf("expected 400 for scan error with older head, got status=%d err=%v", status, err)
		}
	})

	t.Run("PartialPublishFailures_CountOnlySuccesses", func(t *testing.T) {
		handlers.SetLastExecutionTime(0)
		jobs := []internal_types.SeshuJob{{NormalizedUrlKey: "ok"}, {NormalizedUrlKey: "fail"}}
		mockPg := &MockPostgresService{
			ScanJobsWithInHourFunc: func(ctx context.Context, hour int) ([]internal_types.SeshuJob, error) {
				return jobs, nil
			},
		}
		publishCalls := 0
		mockNats := &MockNatsService{
			PeekTopFunc: func(ctx context.Context) (*jetstream.RawStreamMsg, error) { return nil, nil },
			PublishFunc: func(ctx context.Context, job interface{}) error {
				publishCalls++
				j := job.(internal_types.SeshuJob)
				if j.NormalizedUrlKey == "fail" {
					return errors.New("nats publish failed")
				}
				return nil
			},
		}
		ctx := setupMockServices(mockPg, mockNats)
		published, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err != nil || status != http.StatusOK {
			t.Fatalf("expected OK despite partial failures, got status=%d err=%v", status, err)
		}
		if skipped {
			t.Errorf("did not expect skipped")
		}
		if published != 1 || publishCalls != 2 {
			t.Errorf("expected published=1, calls=2; got published=%d calls=%d", published, publishCalls)
		}
	})

	t.Run("LastExecutionTime_UpdatesOnlyWhenNotSkipped", func(t *testing.T) {
		// Ensure update on non-skipped path
		handlers.SetLastExecutionTime(0)
		mockPg := &MockPostgresService{}
		mockNats := &MockNatsService{}
		ctx := setupMockServices(mockPg, mockNats)
		_, skipped, status, err := handlers.ProcessGatherSeshuJobs(ctx, fixedTrigger)
		if err != nil || status != http.StatusOK || skipped {
			t.Fatalf("unexpected result err=%v status=%d skipped=%v", err, status, skipped)
		}
		if handlers.GetLastExecutionTime() != fixedTrigger {
			t.Errorf("expected lastExecutionTime=%d, got %d", fixedTrigger, handlers.GetLastExecutionTime())
		}

		// Cooldown path should NOT update lastExecutionTime
		later := fixedTrigger + 30 // within 60s
		_, skipped2, status2, err2 := handlers.ProcessGatherSeshuJobs(ctx, later)
		if err2 != nil || status2 != http.StatusOK || !skipped2 {
			t.Fatalf("expected cooldown skip, got err=%v status=%d skipped=%v", err2, status2, skipped2)
		}
		if handlers.GetLastExecutionTime() != fixedTrigger {
			t.Errorf("expected lastExecutionTime unchanged=%d, got %d", fixedTrigger, handlers.GetLastExecutionTime())
		}
	})
}

func TestSeshuJobScheduledHourValidation(t *testing.T) {
	// Create a new validator instance for testing
	validate := validator.New()

	// Helper function to create a valid SeshuJob with a specific ScheduledHour
	createValidSeshuJob := func(scheduledHour int) internal_types.SeshuJob {
		return internal_types.SeshuJob{
			NormalizedUrlKey:         "test-url-key",
			LocationLatitude:         1.3521,
			LocationLongitude:        103.8198,
			LocationAddress:          "Test Location",
			ScheduledHour:            scheduledHour,
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
			OwnerID:                  "test-owner",
			KnownScrapeSource:        "TEST",
		}
	}

	t.Run("ValidBounds", func(t *testing.T) {
		testCases := []struct {
			name          string
			scheduledHour int
		}{
			{"LowerBound", 0},   // Midnight (12:00 AM)
			{"UpperBound", 23},  // 11:00 PM
			{"MiddleRange", 12}, // Noon (12:00 PM)
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				job := createValidSeshuJob(tc.scheduledHour)
				err := validate.Struct(job)
				if err != nil {
					t.Errorf("Expected valid ScheduledHour %d, but got validation error: %v", tc.scheduledHour, err)
				}
			})
		}
	})

	t.Run("InvalidBounds", func(t *testing.T) {
		testCases := []struct {
			name          string
			scheduledHour int
			expectedError string
		}{
			{"BelowLowerBound", -1, "min"},
			{"AboveUpperBound", 24, "max"},
			{"WayBelowLowerBound", -10, "min"},
			{"WayAboveUpperBound", 100, "max"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				job := createValidSeshuJob(tc.scheduledHour)
				err := validate.Struct(job)
				if err == nil {
					t.Errorf("Expected validation error for ScheduledHour %d, but got no error", tc.scheduledHour)
				} else if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error to contain '%s' for ScheduledHour %d, but got: %v", tc.expectedError, tc.scheduledHour, err)
				}
			})
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		// Test that 0 (midnight) is specifically allowed - this was the original issue
		job := createValidSeshuJob(0)
		err := validate.Struct(job)
		if err != nil {
			t.Errorf("ScheduledHour 0 (midnight) should be valid, but got error: %v", err)
		}

		// Test that 23 (11 PM) is specifically allowed
		job = createValidSeshuJob(23)
		err = validate.Struct(job)
		if err != nil {
			t.Errorf("ScheduledHour 23 (11 PM) should be valid, but got error: %v", err)
		}
	})
}
