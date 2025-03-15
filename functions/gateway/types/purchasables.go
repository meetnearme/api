package types

import (
	"context"
	"time"
)

// PurchasablesInsert represents the data required to insert a new user
type PurchasableItemInsert struct {
	PurchasableIndex              int        `json:"purchasable_index" dynamodbav:"purchasableIndex"`
	Name                          string     `json:"name" validate:"required" dynamodbav:"name"`
	ItemType                      string     `json:"item_type" validate:"required" dynamodbav:"itemType"` // Validate as email
	Cost                          float64    `json:"cost" validate:"required" dynamodbav:"cost"`
	Inventory                     int32      `json:"inventory" validate:"required" dynamodbav:"inventory"`
	StartingQuantity              int32      `json:"starting_quantity" validate:"required" dynamodbav:"startingQuantity"`
	ChargeRecurrenceInterval      string     `json:"charge_recurrence_interval" validate:"required" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int32      `json:"charge_recurrence_interval_count" validate:"required" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate       time.Time  `json:"charge_recurrence_end_date" validate:"required" dynamodbav:"chargeRecurrenceEndDate"`
	DonationRatio                 float64    `json:"donation_ratio" validate:"required" dynamodbav:"donationRatio"`
	ProximityRequirement          float64    `json:"proximity_requirement" validate:"required" dynamodbav:"proximityRequirement"`
	RegistrationFields            []string   `json:"registration_fields" dynamodbav:"registrationFields"`
	ExpiresOn                     *time.Time `json:"expires_on,omitempty" dynamodbav:"expiresOn,omitempty"`
	CreatedAt                     time.Time  `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt                     time.Time  `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasableInventoryUpdate represents the data required to update a purchasable item's inventory
type PurchasableInventoryUpdate struct {
	Name             string
	Quantity         int32
	PurchasableIndex int
}

// Purchasables represents a user in the system
type PurchasableItem struct {
	Name                          string     `json:"name" dynamodbav:"name"`
	ItemType                      string     `json:"item_type" dynamodbav:"itemType"` // Validate as email
	Cost                          float64    `json:"cost" dynamodbav:"cost"`
	Inventory                     int32      `json:"inventory" dynamodbav:"inventory"`
	StartingQuantity              int32      `json:"starting_quantity" dynamodbav:"startingQuantity"`
	ChargeRecurrenceInterval      string     `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int32      `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate       time.Time  `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
	DonationRatio                 float64    `json:"donation_ratio" dynamodbav:"donationRatio"`
	ProximityRequirement          float64    `json:"proximity_requirement" dynamodbav:"proximityRequirement"`
	RegistrationFields            []string   `json:"registration_fields" dynamodbav:"registrationFields"`
	ExpiresOn                     *time.Time `json:"expires_on,omitempty" dynamodbav:"expiresOn,omitempty"`
	CreatedAt                     time.Time  `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	UpdatedAt                     time.Time  `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasablesUpdate represents the data required to update a user
type PurchasableItemUpdate struct {
	Name                          string     `json:"name" dynamodbav:"name"`
	ItemType                      string     `json:"item_type" dynamodbav:"itemType"` // Validate as email
	Cost                          float64    `json:"cost" dynamodbav:"cost"`
	Inventory                     int32      `json:"inventory" dynamodbav:"inventory"`
	StartingQuantity              int32      `json:"starting_quantity" dynamodbav:"startingQuantity"`
	ChargeRecurrenceInterval      string     `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int32      `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate       time.Time  `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
	ProximityRequirement          float64    `json:"proximity_requirement" dynamodbav:"proximityRequirement"`
	RegistrationFields            []string   `json:"registration_fields" dynamodbav:"registrationFields"`
	DonationRatio                 float64    `json:"donation_ratio" dynamodbav:"donationRatio"`
	ExpiresOn                     *time.Time `json:"expires_on,omitempty" dynamodbav:"expiresOn,omitempty"`
	UpdatedAt                     time.Time  `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

type PurchasableInsert struct {
	EventId          string                  `json:"event_id" validate:"required" dynamodbav:"eventId"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items" validate:"required" dynamodbav:"purchasableItems"`
	CreatedAt        time.Time               `json:"created_at" dynamodbav:"createdAt"`
	UpdatedAt        time.Time               `json:"updated_at" dynamodbav:"updatedAt"`
}

type Purchasable struct {
	EventId          string                  `json:"event_id" dynamodbav:"eventId"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items"  dynamodbav:"purchasableItems"`
	CreatedAt        time.Time               `json:"created_at" dynamodbav:"createdAt"`
	UpdatedAt        time.Time               `json:"updated_at" dynamodbav:"updatedAt"`
}

type PurchasableUpdate struct {
	EventId          string                  `json:"event_id" dynamodbav:"eventId"`
	PurchasableItems []PurchasableItemInsert `json:"purchasable_items" validate:"required" dynamodbav:"purchasableItems"`
	CreatedAt        time.Time               `json:"created_at" dynamodbav:"createdAt"`
	UpdatedAt        time.Time               `json:"updated_at" dynamodbav:"updatedAt"`
}

// PurchasablesServiceInterface defines the methods for user-related operations using the RDSDataAPI
type PurchasableServiceInterface interface {
	InsertPurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, purchasable PurchasableInsert) (*Purchasable, error)
	GetPurchasablesByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) (*Purchasable, error)
	UpdatePurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, purchasable PurchasableUpdate) (*Purchasable, error)
	UpdatePurchasableInventory(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string, updates []PurchasableInventoryUpdate, purchasableMap map[string]PurchasableItemInsert) error
	DeletePurchasable(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) error
}
