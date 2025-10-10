package services

import (
	"context"
	"os"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/types"
)

var (
	postgresService     interfaces.PostgresServiceInterface
	postgresServiceOnce sync.Once
)

func GetPostgresService(ctx context.Context) (interfaces.PostgresServiceInterface, error) {
	// In test mode, check if a mock service is provided in context
	if os.Getenv("GO_ENV") == "test" {
		if mockService, ok := ctx.Value("mockPostgresService").(interfaces.PostgresServiceInterface); ok {
			return mockService, nil
		}
		// Fall back to singleton mock if no context mock provided
		postgresServiceOnce.Do(func() {
			postgresService = getMockPostgresService()
		})
		return postgresService, nil
	}

	// Non-test mode: use singleton pattern
	postgresServiceOnce.Do(func() {
		db, err := GetPostgresClient(ctx)
		if err != nil {
			panic(err) // Or handle initialization error gracefully
		}
		postgresService = NewPostgresService(db)
	})
	return postgresService, nil
}

type MockPostgresService struct{}

func (m *MockPostgresService) GetSeshuJobs(ctx context.Context) ([]types.SeshuJob, error) {
	return []types.SeshuJob{}, nil
}

func (m *MockPostgresService) CreateSeshuJob(ctx context.Context, job types.SeshuJob) error {
	return nil
}

func (m *MockPostgresService) UpdateSeshuJob(ctx context.Context, job types.SeshuJob) error {
	return nil
}

func (m *MockPostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	return nil
}

func (m *MockPostgresService) ScanSeshuJobsWithInHour(ctx context.Context, hours int) ([]types.SeshuJob, error) {
	return []types.SeshuJob{}, nil
}

func (m *MockPostgresService) Close() error {
	return nil
}

func getMockPostgresService() interfaces.PostgresServiceInterface {
	return &MockPostgresService{}
}
