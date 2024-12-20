package dynamodb_handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	dynamodb_service "github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetPurchasesByEventID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	// Get port and create full URL
	port := test_helpers.GetNextPort()
	testMarqoEndpoint := fmt.Sprintf("http://%s", port)
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	// os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	const (
		testEventID          = "123"
		testEventOwnerID     = "789"
		testEventName        = "Test Event"
		testEventDescription = "This is a test event"
	)

	loc, _ := time.LoadLocation("America/New_York")
	testEventStartTime, tmErr := helpers.UtcToUnix64("2099-05-01T12:00:00Z", loc)
	if tmErr != nil || testEventStartTime == 0 {
		t.Logf("Error converting tm UTC to unix: %v", tmErr)
	}

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":            testEventID,
					"startTime":      testEventStartTime,
					"eventOwners":    []interface{}{testEventOwnerID},
					"eventOwnerName": "Event Host Test",
					"name":           testEventName,
					"description":    testEventDescription,
				},
			},
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))
	// Set up mock Marqo server
	mockMarqoServer.Listener.Close()
	listener, err := test_helpers.BindToPort(t, testMarqoEndpoint)
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server after retries: %v", err)
	}
	mockMarqoServer.Listener = listener
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Update the environment variable with the actual bound address
	boundAddress := mockMarqoServer.Listener.Addr().String()
	os.Setenv("DEV_MARQO_API_BASE_URL", fmt.Sprintf("http://%s", boundAddress))

	tests := []struct {
		name          string
		userID        string
		eventOwners   []interface{}
		expectedCode  int
		expectedError string
	}{
		{
			name:         "authorized event owner",
			userID:       testEventOwnerID,
			expectedCode: http.StatusOK,
		},
		{
			name:          "unauthorized user",
			userID:        "unauthorized_user",
			expectedCode:  http.StatusForbidden,
			expectedError: "You are not authorized to view this event's purchases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &dynamodb_service.MockPurchaseService{
				GetPurchasesByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, limit int32, startKey string) ([]internal_types.Purchase, map[string]dynamodb_types.AttributeValue, error) {
					return []internal_types.Purchase{{EventID: eventId, UserID: "user1"}}, nil, nil
				},
			}
			handler := NewPurchaseHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/purchases/event_id", nil)
			req = mux.SetURLVars(req, map[string]string{"event_id": testEventID})

			// Add authentication context with test user
			userInfo := helpers.UserInfo{
				Sub: tt.userID,
			}
			ctx := context.WithValue(req.Context(), "userInfo", userInfo)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetPurchasesByEventID(w, req)
			res := w.Result()
			if res.StatusCode != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, res.StatusCode)
			}
			responseBody, _ := io.ReadAll(res.Body)
			// If we expect an error, verify the error message
			if tt.expectedError != "" {
				if !strings.Contains(string(responseBody), tt.expectedError) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedError, string(responseBody))
				}
			}
		})
	}
}

func TestGetPurchasesByUserID(t *testing.T) {
	tests := []struct {
		name          string
		requestUserID string // user ID in the request path
		contextUserID string // user ID in the context
		expectedCode  int
		expectedError string
	}{
		{
			name:          "authorized user",
			requestUserID: "test_user_123",
			contextUserID: "test_user_123",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "unauthorized user",
			requestUserID: "test_user_123",
			contextUserID: "different_user",
			expectedCode:  http.StatusForbidden,
			expectedError: "You are not authorized to view this user's purchases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &dynamodb_service.MockPurchaseService{
				GetPurchasesByUserIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string, limit int32, startKey string) ([]internal_types.Purchase, map[string]dynamodb_types.AttributeValue, error) {
					return []internal_types.Purchase{{UserID: userId}}, nil, nil
				},
			}
			handler := NewPurchaseHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/purchases/user_id", nil)
			req = mux.SetURLVars(req, map[string]string{"user_id": tt.requestUserID})

			// Add authentication context with test user
			userInfo := helpers.UserInfo{
				Sub: tt.contextUserID,
			}
			ctx := context.WithValue(req.Context(), "userInfo", userInfo)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetPurchasesByUserID(w, req)
			res := w.Result()
			if res.StatusCode != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, res.StatusCode)
			}
			responseBody, _ := io.ReadAll(res.Body)
			// If we expect an error, verify the error message
			if tt.expectedError != "" {
				if !strings.Contains(string(responseBody), tt.expectedError) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedError, string(responseBody))
				}
			}
		})
	}
}

func TestDeletePurchase(t *testing.T) {
	mockService := &dynamodb_service.MockPurchaseService{
		DeletePurchaseFunc: func(ctx context.Context, dynamodbClient types.DynamoDBAPI, eventId string, userId string) error {
			return nil
		},
	}
	handler := NewPurchaseHandler(mockService)

	// Add user context
	const (
		testEventID = "event-123"
		testUserID  = "user-456"
	)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/purchases/%s/%s", testEventID, testUserID), nil)
	req = mux.SetURLVars(req, map[string]string{
		"event_id": testEventID,
		"user_id":  testUserID,
	})

	// Add user context
	userInfo := helpers.UserInfo{
		Sub: testUserID, // Using the same user ID to test authorized deletion
	}
	ctx := context.WithValue(req.Context(), "userInfo", userInfo)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.DeletePurchase(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}
}
