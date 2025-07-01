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
	"github.com/weaviate/weaviate/entities/models"
)

func TestGetPurchasesByEventID(t *testing.T) {
	// --- Standard Test Setup (same pattern as other tests) ---
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
	defer os.Unsetenv("GO_ENV")

	originalWeaviateHost := os.Getenv("WEAVIATE_HOST")
	originalWeaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	originalWeaviatePort := os.Getenv("WEAVIATE_PORT")
	originalTransport := http.DefaultTransport

	defer func() {
		os.Setenv("WEAVIATE_HOST", originalWeaviateHost)
		os.Setenv("WEAVIATE_SCHEME", originalWeaviateScheme)
		os.Setenv("WEAVIATE_PORT", originalWeaviatePort)
		http.DefaultTransport = originalTransport
	}()

	// Set up logging transport
	http.DefaultTransport = test_helpers.NewLoggingTransport(http.DefaultTransport, t)

	// Mock server setup (same as working pattern)
	hostAndPort := test_helpers.GetNextPort()

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

	// Create mock Weaviate server (following established pattern)
	mockWeaviateServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("🎯 MOCK SERVER HIT: %s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/v1/meta":
			t.Logf("   └─ Handling /v1/meta")
			metaResponse := `{"version":"1.23.4"}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(metaResponse))

		case "/v1/graphql":
			t.Logf("   └─ Handling /v1/graphql (event lookup for authorization)")
			if r.Method != "POST" {
				t.Errorf("expected method POST for /v1/graphql, got %s", r.Method)
			}

			// Return event data for authorization check
			mockResponse := models.GraphQLResponse{
				Data: map[string]models.JSONObject{
					"Get": map[string]interface{}{
						"EventStrict": []interface{}{
							map[string]interface{}{
								"name":           testEventName,
								"description":    testEventDescription,
								"eventOwners":    []interface{}{testEventOwnerID},
								"eventOwnerName": "Event Host Test",
								"startTime":      testEventStartTime,
								"timezone":       "America/New_York",
								"_additional": map[string]interface{}{
									"id": testEventID,
								},
							},
						},
					},
				},
			}

			responseBytes, err := json.Marshal(mockResponse)
			if err != nil {
				t.Fatalf("failed to marshal mock GraphQL response: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(responseBytes)

		default:
			t.Logf("   └─ ⚠️  UNHANDLED PATH: %s", r.URL.Path)
			t.Errorf("mock server received request to unhandled path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))

	// Use the same binding pattern as working tests
	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockWeaviateServer.Listener = listener
	mockWeaviateServer.Start()
	defer mockWeaviateServer.Close()

	// Set environment variables to the actual bound port
	actualAddr := listener.Addr().String()
	actualParts := strings.Split(actualAddr, ":")
	actualHost, actualPort := actualParts[0], actualParts[1]

	os.Setenv("WEAVIATE_HOST", actualHost)
	os.Setenv("WEAVIATE_PORT", actualPort)
	os.Setenv("WEAVIATE_SCHEME", "http")
	os.Setenv("WEAVIATE_API_KEY_ALLOWED_KEYS", "test-weaviate-api-key")

	t.Logf("🔧 PURCHASE TEST SETUP COMPLETE")
	t.Logf("   └─ Mock Server bound to: %s", actualAddr)
	t.Logf("   └─ WEAVIATE_HOST: %s", os.Getenv("WEAVIATE_HOST"))
	t.Logf("   └─ WEAVIATE_PORT: %s", os.Getenv("WEAVIATE_PORT"))

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
				GetPurchasesByEventIDFunc: func(ctx context.Context, dynamodbClient internal_types.DynamoDBAPI, eventId string, limit int32, startKey string) ([]internal_types.PurchaseDangerous, map[string]dynamodb_types.AttributeValue, error) {
					return []internal_types.PurchaseDangerous{{EventID: eventId, UserID: "user1", UserEmail: "user1@example.com", UserDisplayName: "User 1"}}, nil, nil
				},
			}
			handler := NewPurchaseHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/purchases/event_id", nil)
			req = mux.SetURLVars(req, map[string]string{helpers.EVENT_ID_KEY: testEventID})

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
		helpers.EVENT_ID_KEY: testEventID,
		"user_id":            testUserID,
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
