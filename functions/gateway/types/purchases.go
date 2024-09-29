package types

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// FlexibleValue can be a string, int32, or bool
type FlexibleValue struct {
	value interface{}
}

// attach behavior the guarantees Unmarshaling JSON works correctly
func (fv *FlexibleValue) UnmarshalJSON(data []byte) error {
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
			fv.value = s
			return nil
	}

	// Try bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
			fv.value = b
			return nil
	}

	// Try int32
	var i int32
	if err := json.Unmarshal(data, &i); err == nil {
			fv.value = i
			return nil
	}

	return fmt.Errorf("unable to unmarshal value")
}

// Handle Marshal default behavior
func (fv FlexibleValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(fv.value)
}

// Add a method that hadles for calling Value on struct
func (fv FlexibleValue) Value() interface{} {
	return fv.value
}

// Purchase represents a purchase in the system
type PurchasedItem struct {
	Name          string `json:"name" dynamodbav:"name"`
	PurchasableIndex int `json:"purchasable_index" dyamodbav:"purchasableIndex"`
	ItemType         string `json:"item_type" dynamodbav:"itemType"` // Validate as email
	Cost		  float64 `json:"cost" dynamodbav:"cost"`
	Quantity  int32 `json:"quantity" dynamodbav:"quantity"`
	Currency         string `json:"currency" dynamodbav:"currency"`
	ChargeRecurrenceInterval string `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval"`
	ChargeRecurrenceIntervalCount int64 `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate time.Time `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
	DonationRatio float64 `json:"donation_ratio" dynamodbav:"donationRatio"`
	RegResponses  []map[string]FlexibleValue `json:"reg_responses" dynamodbav:"regResponses"`
}

// PurchaseInsert represents the data required to insert a new purchase
type PurchaseInsert struct {
	UserID string `json:"user_id" validate:"required" dynamodbav:"userId"`
	EventID string `json:"event_id" validate:"required" dynamodbav:"eventId"`
	EventName string `json:"event_name" validate:"required"  dynamodbav:"eventName"`
  Status         string `json:"status" validate:"required" dynamodbav:"status"`
	PurchasedItems []PurchasedItem `json:"purchased_items" validate:"required" dynamodbav:"purchasedItems"`
	Total          int32 `json:"total" validate:"required" dynamodbav:"total"`
	Currency       string `json:"currency" validate:"required" dynamodbav:"currency"`
	StripeSessionId string `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
  CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
  UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// Purchases represents a purchase in the system
type Purchase struct {
	UserID string `json:"user_id" dynamodbav:"userId"`
	EventID string `json:"event_id" dynamodbav:"eventId"`
	EventName string `json:"event_name" dynamodbav:"eventName"`
  Status         string `json:"status" dynamodbav:"status"`
	PurchasedItems []PurchasedItem `json:"purchased_items" dynamodbav:"purchasedItems"`
	Total          int32 `json:"total" dynamodbav:"total"`
	Currency       string `json:"currency" dynamodbav:"currency"`
	StripeSessionId string `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
  CreatedAt     time.Time `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
  UpdatedAt     time.Time `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasesUpdate represents the data required to update a purchase
type PurchaseUpdate struct {
	UserID string `json:"user_id" dynamodbav:"userId"`
	EventID string `json:"event_id" dynamodbav:"eventId"`
	EventName string `json:"event_name" dynamodbav:"eventName"`
	Status string `json:"status" dynamodbav:"status"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updatedAt"`
}

// PurchasesServiceInterface defines the methods for purchase-related operations using the RDSDataAPI
type PurchaseServiceInterface interface {
	InsertPurchase(ctx context.Context, dynamodbClient DynamoDBAPI, Purchase PurchaseInsert) (*Purchase, error)
	GetPurchaseByPk(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string) (*Purchase, error)
	GetPurchasesByUserID(ctx context.Context, dynamodbClient DynamoDBAPI, userId string) ([]Purchase, error)
	GetPurchasesByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string) ([]Purchase, error)
	UpdatePurchase(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string, Purchase PurchaseUpdate) (*Purchase, error)
	DeletePurchase(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string) error
}



