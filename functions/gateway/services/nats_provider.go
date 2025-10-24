package services

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	natsService     interfaces.NatsServiceInterface
	natsServiceOnce sync.Once
)

func GetNatsService(ctx context.Context) (interfaces.NatsServiceInterface, error) {
	// In test mode, check if a mock service is provided in context
	if os.Getenv("GO_ENV") == "test" {
		if mockService, ok := ctx.Value("mockNatsService").(interfaces.NatsServiceInterface); ok {
			return mockService, nil
		}
		// Fall back to singleton mock if no context mock provided
		natsServiceOnce.Do(func() {
			natsService = getMockNatsService()
		})
		return natsService, nil
	}

	// Non-test mode: use singleton pattern
	natsServiceOnce.Do(func() {
		conn, err := GetNatsClient()
		if err != nil {
			log.Printf("Failed to get NATS client: %v", err)
			// Don't panic, return nil service and let caller handle it
			natsService = nil
			return
		}
		natsService, err = NewNatsService(ctx, conn)
		if err != nil {
			log.Printf("Failed to create NATS service: %v", err)
			// Don't panic, return nil service and let caller handle it
			natsService = nil
			return
		}
	})
	return natsService, nil
}

type MockNatsService struct{}

func (m *MockNatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	return nil, nil
}

func (m *MockNatsService) PublishMsg(ctx context.Context, job interface{}) error {
	return nil
}

func (m *MockNatsService) ConsumeMsg(ctx context.Context, workers int) error {
	return nil
}

func (m *MockNatsService) Close() error {
	return nil
}

func getMockNatsService() interfaces.NatsServiceInterface {
	return &MockNatsService{}
}
