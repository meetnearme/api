package dynamodb_handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestGetRegistrationsByEventID(t *testing.T) {
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	t.Logf("36 << testMarqoEndpoint: %v", testMarqoEndpoint)
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
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

	t.Log("54 << got to")
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
	t.Log("84 << got to")
	// Set up mock Marqo server
	mockMarqoServer.Listener.Close()
	mockMarqoServer.Listener, _ = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()
	t.Log("90 << got to")
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
			expectedError: "You are not authorized to view this event's registrations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockService := &dynamodb_service.MockRegistrationService{
				GetRegistrationsByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, limit int32, startKey string) ([]internal_types.Registration, map[string]dynamodb_types.AttributeValue, error) {
					return []internal_types.Registration{{EventId: eventId, UserId: "user1"}}, nil, nil
				},
			}
			handler := NewRegistrationHandler(mockService)
			t.Log("120 << got to")
			req := httptest.NewRequest(http.MethodGet, "/registrations/event_id", nil)
			req = mux.SetURLVars(req, map[string]string{"event_id": testEventID})

			// Add authentication context with test user
			userInfo := helpers.UserInfo{
				Sub: tt.userID,
			}
			ctx := context.WithValue(req.Context(), "userInfo", userInfo)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetRegistrationsByEventID(w, req)
			t.Log("133 << got to")
			res := w.Result()
			if res.StatusCode != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, res.StatusCode)
			}
			// If we expect an error, verify the error message
			if tt.expectedError != "" {
				var response map[string]interface{}
				if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if msg, ok := response["error"].(map[string]interface{})["message"].(string); !ok ||
					!strings.Contains(msg, tt.expectedError) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedError, msg)
				}
			}
		})
	}
}

func TestGetRegistrationsByUserID(t *testing.T) {
	mockService := &dynamodb_service.MockRegistrationService{
		GetRegistrationsByUserIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, userId string) ([]internal_types.Registration, error) {
			return []internal_types.Registration{{EventId: "event1", UserId: userId}}, nil
		},
	}
	handler := NewRegistrationHandler(mockService)

	// Create request with user_id in path params
	req := httptest.NewRequest(http.MethodGet, "/registrations/user123", nil)
	req = mux.SetURLVars(req, map[string]string{"user_id": "user123"})

	// Create context with user info
	userInfo := helpers.UserInfo{
		Sub: "user123", // This should match the user_id in path params
		// Add other required UserInfo fields if needed
	}
	ctx := context.WithValue(req.Context(), "userInfo", userInfo)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetRegistrationsByUserID(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}

	// Optionally verify response body
	var response []internal_types.Registration
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(response) != 1 || response[0].UserId != "user123" {
		t.Errorf("Unexpected response content")
	}
}
