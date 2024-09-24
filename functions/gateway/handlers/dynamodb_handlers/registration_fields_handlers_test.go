package dynamodb_handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestGetRegistrationFieldsByEventID(t *testing.T) {
	mockService := &dynamodb_service.MockRegistrationFieldsService{
		GetRegistrationFieldsByEventIDFunc: func(ctx context.Context, dynamodbClient types.DynamoDBAPI, eventId string) (*types.RegistrationFields, error) {
			return &types.RegistrationFields{EventId: eventId}, nil
		},
	}
	handler := NewRegistrationFieldsHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/registration_fields/event_id", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event_id"})

	w := httptest.NewRecorder()
	handler.GetRegistrationFieldsByEventID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

func TestDeleteRegistrationFields(t *testing.T) {
	mockService := &dynamodb_service.MockRegistrationFieldsService{
		DeleteRegistrationFieldsFunc: func(ctx context.Context, dynamodbClient types.DynamoDBAPI, eventId string) error {
			return nil
		},
	}
	handler := NewRegistrationFieldsHandler(mockService)

	req := httptest.NewRequest(http.MethodDelete, "/registration_fields/event_id", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event_id"})

	w := httptest.NewRecorder()
	handler.DeleteRegistrationFields(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

