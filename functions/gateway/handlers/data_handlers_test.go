package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/types"
)


func TestCreateEvent(t *testing.T) {
    tests := []struct {
        name string
        requestBody string
        mockInsertFunc    func(ctx context.Context, db types.DynamoDBAPI, event services.EventInsert) (*services.EventSelect, error)
        expectedStatus int
        expectedBodyCheck func(body string) error
    }{
        {
            name:        "Valid event",
            requestBody: `{"name":"Test Event","description":"A test event","datetime":"2099-05-01T12:00:00Z","address":"123 Test St","zip_code":"12345","country":"Test Country","latitude":51.5074,"longitude":-0.1278}`,
            mockInsertFunc: func(ctx context.Context, db types.DynamoDBAPI, event services.EventInsert) (*services.EventSelect, error) {
                return &services.EventSelect{
                    Id:          "mockID",
                    Name:        event.Name,
                    Description: event.Description,
                    StartTime:    event.StartTime,
                    Address:     event.Address,
                    Lat:    event.Lat,
                    Long:   event.Long,
                }, nil
            },
            expectedStatus: http.StatusCreated,
            expectedBodyCheck: func(body string) error {
                var event map[string]interface{}
                if err := json.Unmarshal([]byte(body), &event); err != nil {
                    return fmt.Errorf("failed to unmarshal response body: %v", err)
                }
                if id, ok := event["id"].(string); !ok || id == "" {
                    return fmt.Errorf("expected non-empty id, got '%v'", id)
                }
                return nil
            },
        },
        {
            name:           "Invalid JSON",
            requestBody:    `{"name":"Test Event","description":}`,
            mockInsertFunc: nil,
            expectedStatus: http.StatusUnprocessableEntity,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "Invalid JSON payload") {
                    return fmt.Errorf("expected 'Invalid JSON payload', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:           "Missing required field",
            requestBody:    `{"description":"A test event"}`,
            mockInsertFunc: nil,
            expectedStatus: http.StatusBadRequest,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "Invalid body") {
                    return fmt.Errorf("expected 'Invalid body', got '%s'", body)
                }
                return nil
            },
        },
        {
            name:        "Service error",
            requestBody: `{"name":"Test Event","description":"A test event","datetime":"2023-05-01T12:00:00Z","address":"123 Test St","zip_code":"12345","country":"Test Country","latitude":51.5074,"longitude":-0.1278}`,
            mockInsertFunc: func(ctx context.Context, db types.DynamoDBAPI, event services.EventInsert) (*services.EventSelect, error) {
                return nil, errors.New("service error")
            },
            expectedStatus: http.StatusInternalServerError,
            expectedBodyCheck: func(body string) error {
                if !strings.Contains(body, "Failed to add event") {
                    return fmt.Errorf("expected 'Failed to add event', got '%s'", body)
                }
                return nil
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockService := &services.MockEventService{
                InsertEventFunc: tt.mockInsertFunc,
            }

            req, err := http.NewRequestWithContext(context.Background(), "POST", "/event", bytes.NewBufferString(tt.requestBody))
            if err != nil {
                t.Fatalf("Unexpected error: %v", err)
            }

            rr := httptest.NewRecorder()
            handler := NewEventHandler(mockService)

            handler.CreateEvent(rr, req)

            if status := rr.Code; status != tt.expectedStatus {
                t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
            }

            if err := tt.expectedBodyCheck(rr.Body.String()); err != nil {
                t.Errorf("Body check failed: %v", err)
            }
        })
    }
}
