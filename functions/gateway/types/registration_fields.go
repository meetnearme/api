package types

import (
	"context"
	"time"
)

// RegistrationFieldsInsert represents the data required to insert a new user
type RegistrationFieldsInsert struct {
    ID            string `json:"id"` // UUID format validation
	Name string `json:"name" validate:"required"`
	Type string `json:"type" validate:"required"`
	Options string `json:"options"`
	Default string `json:"default"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
	Required bool `json:"required" validate:"required"`
    CreatedAt     string `json:"created_at"` // Adjust based on your date format
    UpdatedAt     string `json:"updated_at"` // Adjust based on your date format
}


// RegistrationFields represents a user in the system
type RegistrationFields struct {
    ID            string `json:"id"` // UUID format validation
	Name string `json:"name"`
	Type string `json:"type"`
	Options string `json:"options"`
	Default string `json:"default"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
	Required bool `json:"required"`
    CreatedAt     time.Time `json:"created_at"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at"` // Adjust based on your date format
}

// RegistrationFieldsUpdate represents the data required to update a user
type RegistrationFieldsUpdate struct {
    ID            string `json:"id"` // UUID format validation
	Name string `json:"name"`
	Type string `json:"type"`
	Options string `json:"options"`
	Default string `json:"default"`
	Placeholder string `json:"placeholder"`
	Description string `json:"description"`
	Required bool `json:"required"`
}

// RegistrationFieldsServiceInterface defines the methods for user-related operations using the RDSDataAPI
type RegistrationFieldsServiceInterface interface {
	InsertRegistrationFields(ctx context.Context, rdsClient RDSDataAPI, user RegistrationFieldsInsert) (*RegistrationFields, error)
	GetRegistrationFieldsByID(ctx context.Context, rdsClient RDSDataAPI, id string) (*RegistrationFields, error)
	UpdateRegistrationFields(ctx context.Context, rdsClient RDSDataAPI, id string, user RegistrationFieldsUpdate) (*RegistrationFields, error)
	DeleteRegistrationFields(ctx context.Context, rdsClient RDSDataAPI, id string) error
}




