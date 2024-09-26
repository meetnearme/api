package dynamodb_handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bytes"
	"encoding/json"

	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

// other imports remain unchanged

// Modify your TestInsertEventRsvp function to include a request body
func TestInsertEventRsvp(t *testing.T) {
    mockService := &dynamodb_service.MockEventRsvpService{
        InsertEventRsvpFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventRsvp internal_types.EventRsvpInsert) (*internal_types.EventRsvp, error) {
            return &internal_types.EventRsvp{EventID: eventRsvp.EventID, UserID: eventRsvp.UserID}, nil
        },
    }

    handler := NewEventRsvpHandler(mockService)

    // Constructing a JSON body
    body := `{
        "event_id": "event123",
        "user_id": "user123",
        "event_source_type": "someType",
        "event_source_id": "someSourceID",
        "status": "someStatus"
    }`

    req := httptest.NewRequest(http.MethodPost, "/rsvp/event123/user123", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req = mux.SetURLVars(req, map[string]string{"event_id": "event123", "user_id": "user123"})

    w := httptest.NewRecorder()
    handler.CreateEventRsvp(w, req)

    res := w.Result()
    if res.StatusCode != http.StatusCreated {
        t.Errorf("Expected status code 200, got %d", res.StatusCode)
    }
}

// TestUpdateEventRsvp tests updating an RSVP using a mock service.
func TestUpdateEventRsvp(t *testing.T) {
	mockService := &dynamodb_service.MockEventRsvpService{
		UpdateEventRsvpFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string, eventRsvp internal_types.EventRsvpUpdate) (*internal_types.EventRsvp, error) {
			return &internal_types.EventRsvp{EventID: eventId, UserID: userId}, nil
		},
	}

	handler := NewEventRsvpHandler(mockService)

	// Create a valid JSON payload
	eventRsvp := internal_types.EventRsvpUpdate{
		// Populate this struct as needed for your test
	}
	payload, _ := json.Marshal(eventRsvp) // Handle the error properly in production code

	req := httptest.NewRequest(http.MethodPut, "/rsvp/event123/user123", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json") // Set the content type header
	req = mux.SetURLVars(req, map[string]string{"event_id": "event123", "user_id": "user123"})

	w := httptest.NewRecorder()
	handler.UpdateEventRsvp(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}


// TestGetEventRsvpByPk tests fetching RSVP by primary key using a mock service.
func TestGetEventRsvpByPk(t *testing.T) {
	mockService := &dynamodb_service.MockEventRsvpService{
		GetEventRsvpByPkFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) (*internal_types.EventRsvp, error) {
			return &internal_types.EventRsvp{EventID: eventId, UserID: userId}, nil
		},
	}

	handler := NewEventRsvpHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/rsvp/event123/user123", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event123", "user_id": "user123"})

	w := httptest.NewRecorder()
	handler.GetEventRsvpByPk(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

// TestGetEventRsvpsByUserID tests fetching RSVPs by user ID using a mock service.
func TestGetEventRsvpsByUserID(t *testing.T) {
	mockService := &dynamodb_service.MockEventRsvpService{
		GetEventRsvpsByUserIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.EventRsvp, error) {
			return []internal_types.EventRsvp{
				{EventID: "event123", UserID: userId},
				{EventID: "event456", UserID: userId},
			}, nil
		},
	}

	handler := NewEventRsvpHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/rsvp/user_id", nil)
	req = mux.SetURLVars(req, map[string]string{"user_id": "user123"})

	w := httptest.NewRecorder()
	handler.GetEventRsvpsByUserID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

// TestGetEventRsvpsByEventID tests fetching RSVPs by event ID using a mock service.
func TestGetEventRsvpsByEventID(t *testing.T) {
	mockService := &dynamodb_service.MockEventRsvpService{
		GetEventRsvpsByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string) ([]internal_types.EventRsvp, error) {
			return []internal_types.EventRsvp{
				{EventID: eventId, UserID: "user123"},
				{EventID: eventId, UserID: "user456"},
			}, nil
		},
	}

	handler := NewEventRsvpHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/rsvp/event_id", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event123"})

	w := httptest.NewRecorder()
	handler.GetEventRsvpsByEventID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

// TestDeleteEventRsvp tests deleting an RSVP using a mock service.
func TestDeleteEventRsvp(t *testing.T) {
	mockService := &dynamodb_service.MockEventRsvpService{
		DeleteEventRsvpFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId, userId string) error {
			return nil
		},
	}

	handler := NewEventRsvpHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/rsvp/event123/user123", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event123", "user_id": "user123"})

	w := httptest.NewRecorder()
	handler.DeleteEventRsvp(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

