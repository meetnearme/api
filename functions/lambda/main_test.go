package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/meetnearme/api/functions/lambda/test_helpers"
	"github.com/meetnearme/api/functions/lambda/transport"
)

func TestRouter(t *testing.T) {
	// Override the global db with a mock
    mockDB := &test_helpers.MockDynamoDBClient{
        ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
            return &dynamodb.ScanOutput{}, nil
        },
        PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
            return &dynamodb.PutItemOutput{}, nil
        },
        GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
            return &dynamodb.GetItemOutput{}, nil
        },
    }

	// Initialize the router
	InitializeApp(mockDB)

	testCases := []struct {
		name           string
		request        transport.Request
		expectedStatus int
		checkBody   func(body string) bool
	}{
		{
			name: "GET /",
			request: transport.Request{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/",
					},
				},
			},
			expectedStatus: 200,
			checkBody: func(body string) bool {
                return strings.Contains(body, "Meet Near Me - Home") && strings.Contains(body, "header-hero")
            },
		},
		{
			name: "GET /login",
			request: transport.Request{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/login",
					},
				},
			},
			expectedStatus: 200,
            checkBody: func(body string) bool {
                return strings.Contains(body, "Meet Near Me - Login") && strings.Contains(body, "Login Page")
            },
		},
		{
			name: "GET /events/123",
			request: transport.Request{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/events/123",
					},
				},
			},
			expectedStatus: 200,
            checkBody: func(body string) bool {
                return strings.Contains(body, "Event Details") && strings.Contains(body, "Event Id: 123")
            },
		},
		{
			name: "POST /api/event",
			request: transport.Request{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "POST",
						Path:   "/api/event",
					},
				},
				Body: `{"name":"Test Event","description":"A test event","datetime":"2023-05-01T12:00:00Z","address":"123 Test St","zip_code":"12345","country":"Test Country","latitude":51.5074,"longitude":-0.1278}`,
			},
			expectedStatus: 201,
            checkBody: func(body string) bool {
                var response map[string]interface{}
                err := json.Unmarshal([]byte(body), &response)
                return err == nil && response["id"] != nil
            },
		},
		{
			name: "GET /nonexistent",
			request: transport.Request{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
						Method: "GET",
						Path:   "/nonexistent",
					},
				},
			},
			expectedStatus: 404,
            checkBody: func(body string) bool {
                // The body should be empty for a non-existent route
                return body == ""
            },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := Router(context.Background(), tc.request, mockDB)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
            if !tc.checkBody(resp.Body) {
                t.Errorf("Body check failed for %s. Expected an empty body, got: %s", tc.name, resp.Body)
            }
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
