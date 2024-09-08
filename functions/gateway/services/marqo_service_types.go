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
}

func (m *MockMarqoService) UpsertEventToMarqo(
    client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
    return m.UpsertEventToMarqoFunc(client, event)
}
