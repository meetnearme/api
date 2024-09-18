package rds_handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/services/rds_service"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetEventRsvpsByUserID(t *testing.T) {
    mockService := &rds_service.MockEventRsvpService{
        GetEventRsvpsByUserIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, userID string) ([]internal_types.EventRsvp, error) {
            return []internal_types.EventRsvp{
                {ID: "1", UserID: "user123", EventID: "event123", EventSourceType: "email", Status: "confirmed"},
                {ID: "2", UserID: "user123", EventID: "event124", EventSourceType: "email", Status: "confirmed"},
            }, nil
        },
    }
    handler := NewEventRsvpHandler(mockService)

    req := httptest.NewRequest(http.MethodGet, "/events/rsvps/user/user123", nil)
    req = mux.SetURLVars(req, map[string]string{"user_id": "user123"})
    w := httptest.NewRecorder()

    handler.GetEventRsvpsByUserID(w, req)

    res := w.Result()
    if res.StatusCode != http.StatusOK {
        t.Errorf("Expected status code 200, got %d", res.StatusCode)
    }
}


func TestGetEventRsvp(t *testing.T) {
	mockService := &rds_service.MockEventRsvpService{
		GetEventRsvpByIDFunc: func(ctx context.Context, rdsClient internal_types.RDSDataAPI, id string) (*internal_types.EventRsvp, error) {
			// Mock some response
			return &internal_types.EventRsvp{ID: id, UserID: "user123", EventID: "event123"}, nil
		},
	}
	handler := NewEventRsvpHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/events/rsvp/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handler.GetEventRsvp(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}


