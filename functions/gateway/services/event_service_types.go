package services

import (
	"context"

	"github.com/meetnearme/api/functions/gateway/types"
)

// EventServiceInterface defines the methods we need from the services package
type EventServiceInterface interface {
    InsertEvent(ctx context.Context, db types.DynamoDBAPI, createEvent EventInsert) (*EventSelect, error)
}

type EventService struct{}

func NewEventService() EventServiceInterface {
    return &EventService{}
}

func (s *EventService) InsertEvent(ctx context.Context, db types.DynamoDBAPI, createEvent EventInsert) (*EventSelect, error) {
    // This method should contain the actual implementation of InsertEvent
    // For now, we'll just call the existing function
    return InsertEvent(ctx, db, createEvent)
}

type MockEventService struct {
    InsertEventFunc func(ctx context.Context, db types.DynamoDBAPI, createEvent EventInsert) (*EventSelect, error)
}

func (m *MockEventService) InsertEvent(ctx context.Context, db types.DynamoDBAPI, createEvent EventInsert) (*EventSelect, error) {
    return m.InsertEventFunc(ctx, db, createEvent)
}
