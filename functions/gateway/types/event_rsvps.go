package types

import (
	"context"
	"time"
)

// EventRsvpsInsert represents the data required to insert a new user
type EventRsvpInsert struct {
	UserID string `json:"user_id" validate:"required" dynamodbav:"userId"`
	EventID string `json:"event_id" validate:"required" dynamodbav:"eventId"`
    EventSourceType         string `json:"event_source_type" validate:"required" dynamodbav:"eventSourceType"` // Validate as email
    EventSourceID         string `json:"event_source_id" validate:"required" dynamodbav:"eventSourceId"` // Validate as email
    Status         string `json:"status" validate:"required" dynamodbav:"status"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}


// EventRsvps represents a user in the system
type EventRsvp struct {
	UserID string `json:"user_id" dynamodbav:"userId"`
	EventID string `json:"event_id" dynamodbav:"eventId"`
    EventSourceType         string `json:"event_source_type" dynamodbav:"eventSourceType"` // Validate as email
    EventSourceID         string `json:"event_source_id" dynamodbav:"eventSourceId"` // Validate as email
	Status string `json:"status" dynamodbav:"status"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// EventRsvpsUpdate represents the data required to update a user
type EventRsvpUpdate struct {
	UserID string `json:"user_id" dynamodbav:"userId"`
	EventID string `json:"event_id" dynamodbav:"eventId"`
    EventSourceID         string `json:"event_source_id" dynamodbav:"eventSourceId"` // Validate as email
	EventSourceType         string `json:"event_source_type" dynamodbav:"eventSourceType"` // Validate as email
	Status string `json:"status" dynamodbav:"status"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updatedAt"`
}

// EventRsvpsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type EventRsvpServiceInterface interface {
	InsertEventRsvp(ctx context.Context, dynamodbClient DynamoDBAPI, eventRsvp EventRsvpInsert) (*EventRsvp, error)
	GetEventRsvpByPk(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string) (*EventRsvp, error)
	GetEventRsvpsByUserID(ctx context.Context, dynamodbClient DynamoDBAPI, userId string) ([]EventRsvp, error)
	GetEventRsvpsByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) ([]EventRsvp, error)
	UpdateEventRsvp(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string, eventRsvp EventRsvpUpdate) (*EventRsvp, error)
	DeleteEventRsvp(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string) error
}



