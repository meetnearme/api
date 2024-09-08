package services

import (
	"github.com/ganeshdipdumbare/marqo-go"
)

// EventServiceInterface defines the methods we need from the services package
type EventServiceInterface interface {
    UpsertEventToMarqo(
        client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error)
}

type EventService struct{}

func (e *EventService) UpsertEventToMarqo(
    client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
    // Implement the method logic here
    return nil, nil
}

func NewEventService() EventServiceInterface {
    return &EventService{}
}

type MockEventService struct {
    UpsertEventToMarqoFunc func(
        client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error)
}

func (m *MockEventService) UpsertEventToMarqo(
    client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
    return m.UpsertEventToMarqoFunc(client, event)
}
