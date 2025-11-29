package dynamodb_handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/handlers/dynamodb_handlers" // Adjust import path
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetPurchasable(t *testing.T) {

	var eventTime time.Time
	eventTimeStr := "2024-09-01T12:00:00Z" // Replace with your string

	var err error
	eventTime, err = time.Parse(time.RFC3339, eventTimeStr)
	if err != nil {
		// Handle the error
		fmt.Println("Error parsing event time:", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, "2024-09-01T12:00:00Z")
	updatedAt, _ := time.Parse(time.RFC3339, "2024-09-01T12:00:00Z")

	mockService := &dynamodb_service.MockPurchasableService{
		GetPurchasablesByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) (*internal_types.Purchasable, error) { // Change to return []*Purchasable
			return &internal_types.Purchasable{ // Return a pointer to Purchasable
				EventId: eventId,
				PurchasableItems: []internal_types.PurchasableItemInsert{ // Corrected initialization of slice
					{
						Name:                          "Sample Item",
						ItemType:                      "Type A",
						Cost:                          100.0,
						Inventory:                     50,
						StartingQuantity:              100,
						ChargeRecurrenceInterval:      "monthly",
						ChargeRecurrenceIntervalCount: 3,
						ChargeRecurrenceEndDate:       eventTime, // Format if necessary
						DonationRatio:                 0.1,
						RegistrationFields:            []string{"field1", "field2"},
						CreatedAt:                     createdAt, // Use time.Time type
						UpdatedAt:                     updatedAt, // Use time.Time type
					},
				},
				CreatedAt: eventTime, // Use time.Time type
				UpdatedAt: eventTime, // Use time.Time type
			}, nil
		},
	}

	handler := dynamodb_handlers.NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/purchasables/"+constants.EVENT_ID_KEY+"/123", nil)
	req = mux.SetURLVars(req, map[string]string{constants.EVENT_ID_KEY: "123"})

	w := httptest.NewRecorder()
	handler.GetPurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}

	var purchasables *internal_types.Purchasable // Change to pointer slice
	if err := json.NewDecoder(res.Body).Decode(&purchasables); err != nil {
		t.Errorf("Failed to decode response body: %v", err)
	}
	// Further assertions can be made on the returned purchasables if needed
}

func TestUpdatePurchasable(t *testing.T) {
	mockService := &dynamodb_service.MockPurchasableService{
		UpdatePurchasableFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, purchasableUpdate internal_types.PurchasableUpdate) (*internal_types.Purchasable, error) {
			return &internal_types.Purchasable{EventId: purchasableUpdate.EventId}, nil // Mock response
		},
	}

	handler := dynamodb_handlers.NewPurchasableHandler(mockService)

	// Constructing a JSON body
	updatePurchasable := map[string]interface{}{
		"event_id": "123e4567-e89b-12d3-a456-426614174000",
		"purchasable_items": []internal_types.PurchasableItemInsert{
			{
				Name:     "Sample Item",
				ItemType: "Type A",
				Cost:     100.0,
			},
		},
	}

	data, err := json.Marshal(updatePurchasable)
	if err != nil {
		t.Fatal(err) // Handle JSON marshaling error
	}

	req := httptest.NewRequest(http.MethodPut, "/purchasables/"+constants.EVENT_ID_KEY+"/123", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{constants.EVENT_ID_KEY: "123"})

	w := httptest.NewRecorder()
	handler.UpdatePurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}

	var updatedPurchasable *internal_types.Purchasable // Change to pointer
	if err := json.NewDecoder(res.Body).Decode(&updatedPurchasable); err != nil {
		t.Errorf("Failed to decode response body: %v", err)
	}
	// Further assertions can be made on the updatedPurchasable if needed
}

func TestDeletePurchasable(t *testing.T) {
	mockService := &dynamodb_service.MockPurchasableService{
		DeletePurchasableFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) error {
			return nil // Mock successful deletion
		},
	}

	handler := dynamodb_handlers.NewPurchasableHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/purchasables/"+constants.EVENT_ID_KEY+"/123", nil)
	req = mux.SetURLVars(req, map[string]string{constants.EVENT_ID_KEY: "123"})

	w := httptest.NewRecorder()
	handler.DeletePurchasable(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
	}

	expectedMessage := "Purchasable successfully deleted"
	if string(body) != expectedMessage {
		t.Errorf("Expected response body %q, got %q", expectedMessage, string(body))
	}
}
