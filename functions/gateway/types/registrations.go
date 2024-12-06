package types

import (
	"context"
	"time"
)

// RegistrationsInsert represents the data required to insert a new user
type RegistrationInsert struct {
	EventId   string                   `json:"event_id" validate:"required" dynamodbav:"eventId"` // UUID format validation
	UserId    string                   `json:"user_id" validate:"required" dynamodbav:"userId"`
	Responses []map[string]interface{} `json:"responses" dynamodbav:"responses"`
	CreatedAt time.Time                `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt time.Time                `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// Registrations represents a user in the system
type Registration struct {
	EventId   string                   `json:"event_id" validate:"required" dynamodbav:"eventId` // UUID format validation
	UserId    string                   `json:"user_id" validate:"required" dynamodbav:"userId"`
	Responses []map[string]interface{} `json:"responses" dynamodbav:"responses"`
	CreatedAt time.Time                `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt time.Time                `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// RegistrationsUpdate represents the data required to update a user
type RegistrationUpdate struct {
	EventId   string                   `json:"event_id" validate:"required" dynamodbav:"eventId` // UUID format validation
	UserId    string                   `json:"user_id" validate:"required" dynamodbav:"userId"`
	Responses []map[string]interface{} `json:"responses" dynamodbav:"responses"`
	UpdatedAt time.Time                `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// RegistrationsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type RegistrationServiceInterface interface {
	InsertRegistration(ctx context.Context, dynamoClient DynamoDBAPI, registration RegistrationInsert, eventId, userId string) (*Registration, error)
	GetRegistrationByPk(ctx context.Context, dynamoClient DynamoDBAPI, eventId, userId string) (*Registration, error)
	GetRegistrationsByUserID(ctx context.Context, dynamoClient DynamoDBAPI, userId string) ([]Registration, error)
	GetRegistrationsByEventID(ctx context.Context, dynamoClient DynamoDBAPI, eventId string) ([]Registration, error)
	UpdateRegistration(ctx context.Context, dynamoClient DynamoDBAPI, eventId, userId string, registration RegistrationUpdate) (*Registration, error)
	DeleteRegistration(ctx context.Context, dynamoClient DynamoDBAPI, eventId, userId string) error
}
