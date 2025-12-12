package types

import (
	"context"

	"github.com/meetnearme/api/functions/gateway/constants"
)

type Event = constants.Event

type EventSearchResponse struct {
	Events []Event `json:"events"`
	Filter string  `json:"filter,omitempty"`
	Query  string  `json:"query,omitempty"`
}

type EventService interface {
	BulkUpsertEvent(ctx context.Context, events []Event) error
	SearchEvents(ctx context.Context, query string, userLocation []float64, maxDistance float64, startTime, endTime int64, ownerIds []string, categories string, address string, parseDates string, eventSourceTypes []string, eventSourceIds []string) (EventSearchResponse, error)
	BulkGetEventByID(ctx context.Context, docIds []string, parseDates string) ([]*Event, error)
	GetEventByID(ctx context.Context, docId string, parseDates string) (*Event, error)
	BulkDeleteEvents(ctx context.Context, docIds []string) error
}
