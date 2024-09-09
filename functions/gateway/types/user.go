package types

import (
	"context"
	"time"
)

// UserInsert represents the data required to insert a new user
type UserInsert struct {
    ID            string `json:"id"` // UUID format validation
    Name          string `json:"name" validate:"required"`
    Email         string `json:"email" validate:"required,email"` // Validate as email
    Address		  string `json:"address"`
    Phone         string `json:"phone"`
    ProfilePictureURL string `json:"profilePictureUrl"`
    CreatedAt     string `json:"created_at"` // Adjust based on your date format
    UpdatedAt     string `json:"updated_at"` // Adjust based on your date format
    Role          string `json:"role" validate:"required"`
    OrganizationUserID string `json:"organizationUserId"`
}


// User represents a user in the system
type User struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	Address     string    `json:"address,omitempty"`
	Phone             string    `json:"phone,omitempty"`
	ProfilePictureURL string    `json:"profile_picture_url,omitempty"`
	Role              string    `json:"role"`
	OrganizationUserID *string  `json:"organization_user_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UserUpdate represents the data required to update a user
type UserUpdate struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	Address     string `json:"address,omitempty"`
	Phone             string `json:"phone,omitempty"`
	ProfilePictureURL string `json:"profile_picture_url,omitempty"`
	Role              string `json:"role,omitempty" validate:"omitempty,oneof=standard_user organization_user suborganization_user"`
	OrganizationUserID *string `json:"organization_user_id,omitempty" validate:"omitempty,uuid4"`
}

// UserServiceInterface defines the methods for user-related operations using the RDSDataAPI
type UserServiceInterface interface {
	InsertUser(ctx context.Context, rdsClient RDSDataAPI, user UserInsert) (*User, error)
	GetUserByID(ctx context.Context, rdsClient RDSDataAPI, id string) (*User, error)
	GetUsers(ctx context.Context, rdsClient RDSDataAPI) ([]User, error)
	UpdateUser(ctx context.Context, rdsClient RDSDataAPI, id string, user UserUpdate) (*User, error)
	DeleteUser(ctx context.Context, rdsClient RDSDataAPI, id string) error
}


