package types

import (
	"context"
	"time"
)

// RegistrationsInsert represents the data required to insert a new user
type RegistrationInsert struct {
    ID            string `json:"id"` // UUID format validation
	UserID string `json:"user_id" validastringte:"required"`
    Name          string `json:"name" validate:"required"`
    ItemType         string `json:"item_type" validate:"required"` // Validate as email
    Cost		  float64 `json:"cost" validate:"required"`
	Inventory int64 `json:"inventory" validate:"required"`
    Currency         string `json:"currency" validate:"required"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval" validate:"required"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count" validate:"required"`
	ChargeRecurrenceEndDate string `json:"charge_recurrence_end_date" validate:"required"`
	DonationRatio float64 `json:"donation_ratio" validate:"required"`
    CreatedAt     string `json:"created_at"` // Adjust based on your date format
    UpdatedAt     string `json:"updated_at"` // Adjust based on your date format
}


// Registrations represents a user in the system
type Registration struct {
    ID            string `json:"id"` // UUID format validation
	UserID string `json:"userId" `
    Name          string `json:"name"`
    ItemType         string `json:"item_type"` // Validate as email
    Cost		  float64 `json:"cost"`
	Inventory int64 `json:"inventory"`
    Currency         string `json:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count"`
	ChargeRecurrenceEndDate time.Time `json:"charge_recurrence_end_date"`
    DonationRatio float64 `json:"donation_ratio"`
    CreatedAt     time.Time `json:"created_at"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at"` // Adjust based on your date format
}

// RegistrationsUpdate represents the data required to update a user
type RegistrationUpdate struct {
	UserID string `json:"userId"`
    Name          string `json:"name" `
    ItemType         string `json:"item_type" ` // Validate as email
    Cost		  float64 `json:"cost"`
	Inventory int64 `json:"inventory"`
    Currency         string `json:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count"`
	ChargeRecurrenceEndDate string `json:"charge_recurrence_end_date"`
    DonationRatio float64 `json:"donation_ratio"`
}

// RegistrationsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type RegistrationServiceInterface interface {
	InsertRegistration(ctx context.Context, dynamoClient DynamoDBAPI, registration RegistrationInsert) (*Registration, error)
	GetRegistrationByID(ctx context.Context, dynamoClient DynamoDBAPI, id string) (*Registration, error)
	GetRegistrationsByEventID(ctx context.Context, dynamoClient DynamoDBAPI, eventId string) ([]Registration, error)
	UpdateRegistration(ctx context.Context, dynamoClient DynamoDBAPI, id string, registration RegistrationUpdate) (*Registration, error)
	DeleteRegistration(ctx context.Context, dynamoClient DynamoDBAPI, id string) error
}




