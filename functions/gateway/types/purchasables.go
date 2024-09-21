package types

import (
	"context"
	"time"
)

// PurchasablesInsert represents the data required to insert a new user
type PurchasableInsert struct {
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


// Purchasables represents a user in the system
type Purchasable struct {
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

// PurchasablesUpdate represents the data required to update a user
type PurchasableUpdate struct {
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

// PurchasablesServiceInterface defines the methods for user-related operations using the RDSDataAPI
type PurchasableServiceInterface interface {
	InsertPurchasable(ctx context.Context, rdsClient RDSDataAPI, user PurchasableInsert) (*Purchasable, error)
	GetPurchasableByID(ctx context.Context, rdsClient RDSDataAPI, id string) (*Purchasable, error)
	GetPurchasablesByUserID(ctx context.Context, rdsClient RDSDataAPI, userId string) ([]Purchasable, error)
	UpdatePurchasable(ctx context.Context, rdsClient RDSDataAPI, id string, user PurchasableUpdate) (*Purchasable, error)
	DeletePurchasable(ctx context.Context, rdsClient RDSDataAPI, id string) error
}



