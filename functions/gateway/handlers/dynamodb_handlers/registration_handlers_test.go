package dynamodb_handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetRegistrationsByEventID(t *testing.T) {
	mockService := &dynamodb_service.MockRegistrationService{
		GetRegistrationsByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, limit int32, startKey string) ([]internal_types.Registration, map[string]dynamodb_types.AttributeValue, error) {
			return []internal_types.Registration{{EventId: eventId, UserId: "user1"}}, nil, nil
		},
	}
	handler := NewRegistrationHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/registrations/event_id", nil)
	req = mux.SetURLVars(req, map[string]string{"event_id": "event_id"})

	w := httptest.NewRecorder()
	handler.GetRegistrationsByEventID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}

func TestGetRegistrationsByUserID(t *testing.T) {
	mockService := &dynamodb_service.MockRegistrationService{
		GetRegistrationsByUserIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Registration, error) {
			return []internal_types.Registration{{EventId: "event1", UserId: userId}}, nil
		},
	}
	handler := NewRegistrationHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/registrations/user_id", nil)
	req = mux.SetURLVars(req, map[string]string{"user_id": "user_id"})

	w := httptest.NewRecorder()
	handler.GetRegistrationsByUserID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}
