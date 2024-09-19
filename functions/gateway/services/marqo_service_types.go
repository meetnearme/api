package services

import (
	"github.com/ganeshdipdumbare/marqo-go"
)

// MarqoServiceInterface defines the methods we need from the services package
type MarqoServiceInterface interface {
    UpsertEventToMarqo(
        client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error)
}

type MarqoService struct{}

func (e *MarqoService) UpsertEventToMarqo(
    client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
    // Implement the method logic here
    return nil, nil
}

func NewMarqoService() MarqoServiceInterface {
    return &MarqoService{}
}

type MockMarqoService struct {
    UpsertEventToMarqoFunc func(
        client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error)
    BulkUpsertEventToMarqoFunc func(
            client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error)
    SearchEventsFunc       func(client *marqo.Client, query string, userLocation []float64, maxDistance float64, startTime int64, endTime int64, ownerIds []string) (EventSearchResponse, error)
    UpdateOneEventFunc     func(client *marqo.Client, eventId string, event Event) (*marqo.UpsertDocumentsResponse, error)
    BulkUpdateEventsFunc   func(client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error)
}

func (m *MockMarqoService) UpsertEventToMarqo(
    client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
    return m.UpsertEventToMarqoFunc(client, event)
}

func (m *MockMarqoService) BulkUpsertEventToMarqo(
    client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error) {
    return m.BulkUpsertEventToMarqoFunc(client, events)
}

func (m *MockMarqoService) SearchEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64, startTime int64, endTime int64, ownerIds []string) (EventSearchResponse, error) {
	return m.SearchEventsFunc(client, query, userLocation, maxDistance, startTime, endTime, ownerIds)
}

func (m *MockMarqoService) UpdateOneEvent(client *marqo.Client, eventId string, event Event) (*marqo.UpsertDocumentsResponse, error) {
	return m.UpdateOneEventFunc(client, eventId, event)
}

func (m *MockMarqoService) BulkUpdateEvents(client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error) {
    return m.BulkUpdateEventsFunc(client, events)
}
