package types

import (
	"context"
	"time"
)

// PurchasablesInsert represents the data required to insert a new user
type PurchasableItemInsert struct {
    Name          string `json:"name" validate:"required" dynamodbav:"name"`
    ItemType         string `json:"item_type" validate:"required" dynamodbav:"itemType"` // Validate as email
    Cost		  float64 `json:"cost" validate:"required" dynamodbav:"cost"`
	Inventory int64 `json:"inventory" validate:"required" dynamodbav:"inventory"`
	StartingQuantity int64	`json:"starting_quantity" validate:"required" dynamodbav:"startingQuantity"`
    Currency         string `json:"currency" validate:"required" dynamodbav:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval" validate:"required" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count" validate:"required" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate string `json:"charge_recurrence_end_date" validate:"required" dynamodbav:"chargeRecurrenceEndDate"`
	DonationRatio float64 `json:"donation_ratio" validate:"required" dynamodbav:"donationRatio"`
    CreatedAt     string `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     string `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}


// Purchasables represents a user in the system
type PurchasableItem struct {
    Name          string `json:"name" dynamodbav:"name"`
    ItemType         string `json:"item_type" dynamodbav:"itemType"` // Validate as email
    Cost		  float64 `json:"cost" dynamodbav:"cost"`
	Inventory int64 `json:"inventory" dynamodbav:"inventory"`
	StartingQuantity int64	`json:"starting_quantity" dynamodbav:"startingQuantity"`
    Currency         string `json:"currency" dynamodbav:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate time.Time `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
    DonationRatio float64 `json:"donation_ratio" dynamodbav:"donationRatio"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasablesUpdate represents the data required to update a user
type PurchasableItemUpdate struct {
	Name          string `json:"name" dynamodbav:"name"`
	ItemType         string `json:"item_type" dynamodbav:"itemType"` // Validate as email
    Cost		  float64 `json:"cost" dynamodbav:"cost"`
	Inventory int64 `json:"inventory" dynamodbav:"inventory"`
	StartingQuantity int64	`json:"starting_quantity" dynamodbav:"startingQuantity"`
    Currency         string `json:"currency" dynamodbav:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate string `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
    DonationRatio float64 `json:"donation_ratio" dynamodbav:"donationRatio"`
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

type PurchasableInsert struct {
	EventId string `json:"event_id" validate:"required" dynamodbav:"eventId"`
	RegistrationFieldsName []string `json:"registration_fields" validate:"required" dynamodbav:"registrationFields"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items" validate:"required" dynamodbav:"purchasableItems"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

type Purchasable struct {
	EventId string `json:"event_id" dynamodbav:"eventId"`
	RegistrationFieldsNames []string `json:"registration_fields"  dynamodbav:"registrationFields"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items"  dynamodbav:"purchasableItems"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

type PurchasableUpdate struct {
	EventId string `json:"event_id" validastringte:"required" dynamodbav:"eventId"`
	RegistrationFieldsNames []string `json:"registration_fields"  dynamodbav:"registrationFields"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items" dynamodbav:"purchasableItems"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
    UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasablesServiceInterface defines the methods for user-related operations using the RDSDataAPI
type PurchasableServiceInterface interface {
	InsertPurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, purchasable PurchasableInsert) (*Purchasable, error)
	GetPurchasablesByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) (*Purchasable, error)
	UpdatePurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, purchasable PurchasableUpdate) (*Purchasable, error)
	DeletePurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) error
}



