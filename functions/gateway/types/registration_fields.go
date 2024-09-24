package types

import (
	"context"
	"time"
)

// RegistrationFieldsInsert represents the data required to insert a new user
type RegistrationFieldItemInsert struct {
    Name        string   `json:"name" dynamodbav:"name" validate:"required"`
    Type        string   `json:"type" dynamodbav:"type" validate:"required"`
    Options     []string `json:"options" dynamodbav:"options"`
    Default     string   `json:"default" dynamodbav:"default"`
    Placeholder string   `json:"placeholder" dynamodbav:"placeholder"`
    Description string   `json:"description" dynamodbav:"description"`
    Required    bool     `json:"required" dynamodbav:"required" validate:"required"`
}

// RegistrationFields represents a user in the system
type RegistrationField struct {
    Name        string   `json:"name" dynamodbav:"name"`
    Type        string   `json:"type" dynamodbav:"type"`
    Options     []string `json:"options" dynamodbav:"options"`
    Default     string   `json:"default" dynamodbav:"default"`
    Placeholder string   `json:"placeholder" dynamodbav:"placeholder"`
    Description string   `json:"description" dynamodbav:"description"`
    Required    bool     `json:"required" dynamodbav:"required"`
}

// RegistrationFieldsUpdate represents the data required to update a user
type RegistrationFieldItemUpdate struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Options []string `json:"options"`
	Default string `json:"default"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
	Required bool `json:"required"`
}

type RegistrationFieldsInsert struct {
	EventId string `json:"user_id" dynamodbav:"eventId" validate:"required"`
	Fields []RegistrationFieldItemInsert `json:"fields" dynamodbav:"fields"`
	CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date formatstors
	UpdatedBy string `json:"updated_by" validate:"required" dynamodbav:"updatedBy"`
}

type RegistrationFields struct {
	EventId string `json:"user_id" dynamodbav:"eventId"`
	Fields []RegistrationField `json:"fields" dynamodbav:"fields"`
	CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
	UpdatedBy string `json:"updated_by" dynamodbav:"updatedBy"`
}

type RegistrationFieldsUpdate struct {
	Fields []RegistrationField `json:"fields" dynamodbav:"fields"`
	UpdatedBy string `json:"updated_by" validate:"required"`
	UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// RegistrationFieldsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type RegistrationFieldsServiceInterface interface {
	InsertRegistrationFields(ctx context.Context, dynamodbClient DynamoDBAPI, registrationFields RegistrationFieldsInsert, eventId string) (*RegistrationFields, error)
	GetRegistrationFieldsByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) (*RegistrationFields, error)
	UpdateRegistrationFields(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string, registrationFields RegistrationFieldsUpdate) (*RegistrationFields, error)
	DeleteRegistrationFields(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) error
}
