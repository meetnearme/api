package types

import (
	"context"
	"time"
)

// EventRsvpsInsert represents the data required to insert a new user
type EventRsvpInsert struct {
    ID            string `json:"id"` // UUID format validation
	UserID string `json:"user_id" validate:"required"`
	EventID string `json:"event_id" validate:"required"`
    EventSourceType         string `json:"event_source_type" validate:"required"` // Validate as email
    Status         string `json:"status" validate:"required"`
    CreatedAt     string `json:"created_at"` // Adjust based on your date format
    UpdatedAt     string `json:"updated_at"` // Adjust based on your date format
}


// EventRsvps represents a user in the system
type EventRsvp struct {
    ID            string `json:"id"` // UUID format validation
	UserID string `json:"user_id" `
	EventID string `json:"event_id" `
    EventSourceType         string `json:"event_source_type"` // Validate as email
	Status string `json:"status"`
    CreatedAt     time.Time `json:"created_at"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at"` // Adjust based on your date format
}

// EventRsvpsUpdate represents the data required to update a user
type EventRsvpUpdate struct {
    ID            string `json:"id"` // UUID format validation
	UserID string `json:"user_id" `
	EventID string `json:"event_id" `
    EventSourceType         string `json:"event_source_type" ` // Validate as email
	Status string `json:"status"`
}

// EventRsvpsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type EventRsvpServiceInterface interface {
	InsertEventRsvp(ctx context.Context, rdsClient RDSDataAPI, user EventRsvpInsert) (*EventRsvp, error)
	GetEventRsvpByID(ctx context.Context, rdsClient RDSDataAPI, id string) (*EventRsvp, error)
	GetEventRsvpsByUserID(ctx context.Context, rdsClient RDSDataAPI, userId string) ([]EventRsvp, error)
	GetEventRsvpsByEventID(ctx context.Context, rdsClient RDSDataAPI, eventId string) ([]EventRsvp, error)
	UpdateEventRsvp(ctx context.Context, rdsClient RDSDataAPI, id string, user EventRsvpUpdate) (*EventRsvp, error)
	DeleteEventRsvp(ctx context.Context, rdsClient RDSDataAPI, id string) error
}



