package types

import (
	"context"
	"time"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Purchase represents a purchase in the system
type PurchasedItem struct {
	Name                          string                   `json:"name" dynamodbav:"name" validate:"required"`
	PurchasableIndex              int                      `json:"purchasable_index" dynamodbav:"purchasableIndex" validate:"required,gte=0"`
	ItemType                      string                   `json:"item_type" dynamodbav:"itemType" validate:"required"`
	Cost                          float64                  `json:"cost" dynamodbav:"cost" validate:"required,gte=0"`
	Quantity                      int32                    `json:"quantity" dynamodbav:"quantity" validate:"required,gt=0"`
	Currency                      string                   `json:"currency" dynamodbav:"currency" validate:"required,iso4217"`
	ChargeRecurrenceInterval      string                   `json:"charge_recurrence_interval" dynamodbav:"chargeRecurrenceInterval" validate:"omitempty,oneof=daily weekly monthly yearly"`
	ChargeRecurrenceIntervalCount int                      `json:"charge_recurrence_interval_count" dynamodbav:"chargeRecurrenceIntervalCount"`
	ChargeRecurrenceEndDate       time.Time                `json:"charge_recurrence_end_date" dynamodbav:"chargeRecurrenceEndDate"`
	DonationRatio                 float64                  `json:"donation_ratio" dynamodbav:"donationRatio" validate:"gte=0,lte=1"`
	RegResponses                  []map[string]interface{} `json:"reg_responses" dynamodbav:"regResponses"`
}

// PurchaseInsert represents the data required to insert a new purchase
type PurchaseInsert struct {
	UserID              string          `json:"user_id" validate:"required" dynamodbav:"userId"`
	EventID             string          `json:"event_id" validate:"required" dynamodbav:"eventId"`
	CompositeKey        string          `json:"composite_key" validate:"required" dynamodbav:"compositeKey"`
	EventName           string          `json:"event_name" validate:"required"  dynamodbav:"eventName"`
	Status              string          `json:"status" validate:"required" dynamodbav:"status"`
	PurchasedItems      []PurchasedItem `json:"purchased_items" validate:"required" dynamodbav:"purchasedItems"`
	Total               int32           `json:"total" dynamodbav:"total"`
	Currency            string          `json:"currency" dynamodbav:"currency"`
	StripeSessionId     string          `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
	StripeTransactionId string          `json:"stripe_transaction_id" dynamodbav:"stripeTransactionId"`
	CreatedAt           int64           `json:"created_at" validate:"required" dynamodbav:"createdAt"` // Adjust based on your date format
	CreatedAtString     string          `json:"created_at_string" validate:"required" dynamodbav:"createdAtString"`
	UpdatedAt           int64           `json:"updated_at" validate:"required" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// Purchases represents a purchase in the system
type Purchase struct {
	UserID              string          `json:"user_id" dynamodbav:"userId"`
	EventID             string          `json:"event_id" dynamodbav:"eventId"`
	CompositeKey        string          `json:"composite_key" dynamodbav:"compositeKey"`
	EventName           string          `json:"event_name" dynamodbav:"eventName"`
	Status              string          `json:"status" dynamodbav:"status"`
	PurchasedItems      []PurchasedItem `json:"purchased_items" dynamodbav:"purchasedItems"`
	Total               int32           `json:"total" dynamodbav:"total"`
	Currency            string          `json:"currency" dynamodbav:"currency"`
	StripeSessionId     string          `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
	StripeTransactionId string          `json:"stripe_transaction_id" dynamodbav:"stripeTransactionId"`
	CreatedAt           int64           `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	CreatedAtString     string          `json:"created_at_string" dynamodbav:"createdAtString"`
	UpdatedAt           int64           `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// This type exposes sensitive data to the client
// it should not be used except when aggressively gated
type PurchaseDangerous struct {
	UserID              string          `json:"user_id" dynamodbav:"userId"`
	UserEmail           string          `json:"user_email" dynamodbav:"email"`
	UserDisplayName     string          `json:"user_display_name" dynamodbav:"displayName"`
	EventID             string          `json:"event_id" dynamodbav:"eventId"`
	CompositeKey        string          `json:"composite_key" dynamodbav:"compositeKey"`
	EventName           string          `json:"event_name" dynamodbav:"eventName"`
	Status              string          `json:"status" dynamodbav:"status"`
	PurchasedItems      []PurchasedItem `json:"purchased_items" dynamodbav:"purchasedItems"`
	Total               int32           `json:"total" dynamodbav:"total"`
	Currency            string          `json:"currency" dynamodbav:"currency"`
	StripeSessionId     string          `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
	StripeTransactionId string          `json:"stripe_transaction_id" dynamodbav:"stripeTransactionId"`
	CreatedAt           int64           `json:"created_at" dynamodbav:"createdAt"` // Adjust based on your date format
	CreatedAtString     string          `json:"created_at_string" dynamodbav:"createdAtString"`
	UpdatedAt           int64           `json:"updated_at" dynamodbav:"updatedAt"` // Adjust based on your date format
}

// PurchasesUpdate represents the data required to update a purchase
type PurchaseUpdate struct {
	UserID              string `json:"user_id" dynamodbav:"userId"`
	EventID             string `json:"event_id" dynamodbav:"eventId"`
	CompositeKey        string `json:"composite_key" dynamodbav:"compositeKey"`
	EventName           string `json:"event_name" dynamodbav:"eventName"`
	Status              string `json:"status" dynamodbav:"status"`
	UpdatedAt           int64  `json:"updated_at" dynamodbav:"updatedAt"`
	StripeSessionId     string `json:"stripe_session_id" dynamodbav:"stripeSessionId"`
	StripeTransactionId string `json:"stripe_transaction_id" dynamodbav:"stripeTransactionId"`
}

type HasPurchaseForEventPayload struct {
	ChildEventId  string `json:"childEventId" validate:"required" dynamodbav:"childEventId"`
	ParentEventId string `json:"parentEventId" validate:"required" dynamodbav:"parentEventId"`
}

// PurchasesServiceInterface defines the methods for purchase-related operations using the RDSDataAPI
type PurchaseServiceInterface interface {
	InsertPurchase(ctx context.Context, dynamodbClient DynamoDBAPI, Purchase PurchaseInsert) (*Purchase, error)
	GetPurchaseByPk(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId, createdAt string) (*Purchase, error)
	GetPurchasesByUserID(ctx context.Context, dynamodbClient DynamoDBAPI, userId string, limit int32, startKey string) ([]Purchase, map[string]dynamodb_types.AttributeValue, error)
	GetPurchasesByEventID(ctx context.Context, dynamodbClient DynamoDBAPI, eventId string, limit int32, startKey string) ([]PurchaseDangerous, map[string]dynamodb_types.AttributeValue, error)
	UpdatePurchase(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId, createdAtString string, Purchase PurchaseUpdate) (*Purchase, error)
	DeletePurchase(ctx context.Context, dynamodbClient DynamoDBAPI, eventId, userId string) error
	HasPurchaseForEvent(ctx context.Context, dynamodbClient DynamoDBAPI, childEventId, parentEventId, userId string) (bool, error)
}
