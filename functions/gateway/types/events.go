package types

import (
	"context"
	"time"
)

type Event struct {
	Id                    string        `json:"id,omitempty"`
	EventOwners           []string      `json:"eventOwners" validate:"required,min=1"`
	EventOwnerName        string        `json:"eventOwnerName" validate:"required"`
	EventSourceType       string        `json:"eventSourceType" validate:"required"`
	Name                  string        `json:"name" validate:"required"`
	Description           string        `json:"description" validate:"required"`
	StartTime             int64         `json:"startTime" validate:"required"`
	EndTime               int64         `json:"endTime,omitempty"`
	Address               string        `json:"address" validate:"required"`
	Lat                   float64       `json:"lat" validate:"required"`
	Long                  float64       `json:"long" validate:"required"`
	EventSourceId         string        `json:"eventSourceId"`
	StartingPrice         int32         `json:"startingPrice,omitempty"`
	Currency              string        `json:"currency,omitempty"`
	PayeeId               string        `json:"payeeId,omitempty"`
	SourceUrl             string        `json:"sourceUrl,omitempty"`
	HasRegistrationFields bool          `json:"hasRegistrationFields,omitempty"`
	HasPurchasable        bool          `json:"hasPurchasable,omitempty"`
	ImageUrl              string        `json:"imageUrl,omitempty"`
	Timezone              time.Location `json:"timezone" validate:"required"`
	Categories            []string      `json:"categories,omitempty"`
	Tags                  []string      `json:"tags,omitempty"`
	CreatedAt             int64         `json:"createdAt,omitempty"`
	UpdatedAt             int64         `json:"updatedAt,omitempty"`
	UpdatedBy             string        `json:"updatedBy,omitempty"`
	RefUrl                string        `json:"refUrl,omitempty"`
	HideCrossPromo        bool          `json:"hideCrossPromo,omitempty"`
	CompetitionConfigId   string        `json:"competitionConfigId,omitempty"`
	ShadowOwners          []string      `json:"shadowOwners,omitempty"`

	// New fields for UI use only
	LocalizedStartDate string `json:"localStartDate,omitempty"`
	LocalizedStartTime string `json:"localStartTime,omitempty"`
}

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
