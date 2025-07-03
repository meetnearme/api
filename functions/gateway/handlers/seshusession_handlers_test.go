package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

type MockScrapingService struct {
	GetHTMLFromURLFunc func(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error)
}

func (m *MockScrapingService) GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error) {
	return m.GetHTMLFromURLFunc(unescapedURL, timeout, jsRender, waitFor)
}

func TestNonLambdaRouter(t *testing.T) {
	// Save original environment variables
	originalScrapingBeeAPIBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	originalOpenAIAPIBaseURL := os.Getenv("OPENAI_API_BASE_URL")
	originalOpenAIAPIKey := os.Getenv("OPENAI_API_KEY")

	// Set up mock ScrapingBee server
	mockScrapingBee := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Mock HTML Content</body></html>"))
	}))
	defer mockScrapingBee.Close()

	// Set up mock OpenAI server
	mockOpenAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := ChatCompletionResponse{
			ID:      "mock-session-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: `[{"event_title":"Mock Event","event_location":"Mock Location","event_start_time":"2023-05-01T10:00:00Z","event_end_time":"2023-05-01T12:00:00Z","event_url":"https://mock-event.com"}]`,
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockOpenAI.Close()

	// Override environment variables
	os.Setenv("SCRAPINGBEE_API_URL_BASE", mockScrapingBee.URL)
	os.Setenv("OPENAI_API_BASE_URL", mockOpenAI.URL)
	os.Setenv("OPENAI_API_KEY", "mock-api-key")

	// Restore env after test
	defer func() {
		os.Setenv("SCRAPINGBEE_API_URL_BASE", originalScrapingBeeAPIBaseURL)
		os.Setenv("OPENAI_API_BASE_URL", originalOpenAIAPIBaseURL)
		os.Setenv("OPENAI_API_KEY", originalOpenAIAPIKey)
	}()

	// Mock DynamoDB client
	mockDB := &test_helpers.MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	db = mockDB // Inject mock into global var

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Decode request body into InternalRequest
		var reqBody InternalRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Check for required query param
		if r.URL.Query().Get("action") != "init" && r.URL.Query().Get("action") != "rs" {
			log.Print(r.URL.Query())
			http.Error(w, "Missing or invalid action", http.StatusBadRequest)
			return
		}

		// Call your business logic
		resp, err := HandlePost(r.Context(), reqBody, &services.RealScrapingService{})
		if err != nil {
			log.Printf("handlePost failed: %+v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Write response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp.Body))
	})

	tests := []struct {
		name           string
		method         string
		action         string
		body           string
		expectedStatus int
		expectBodyText string
	}{
		{
			name:           "Valid POST request for init",
			method:         "POST",
			action:         "init",
			body:           `{"method": "POST", "action": "init", "body": "{\"url\":\"https://example.com\"}", "headers": {}}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "Mock Event",
		},
		{
			name:           "Valid POST request for recursive",
			method:         "POST",
			action:         "rs",
			body:           `{"method": "POST", "action": "rs", "body": "{\"parent_url\":\"https://example.com\",\"url\":\"https://example2.com\"}", "headers": {}}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "Mock Event",
		},
		{
			name:           "Unsupported GET request",
			method:         "GET",
			action:         "init",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == "POST" {
				req = httptest.NewRequest(http.MethodPost, "/?action="+tt.action, bytes.NewBuffer([]byte(tt.body)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(http.MethodGet, "/?action="+tt.action, nil)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			if tt.expectBodyText != "" && !strings.Contains(rec.Body.String(), tt.expectBodyText) {
				t.Errorf("Expected body to contain %q, got: %s", tt.expectBodyText, rec.Body.String())
			}
		})
	}
}

// func TestCreateChatSession(t *testing.T) {
// 	// Setup mock server
// 	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.URL.Path != "/chat/completions" {
// 			t.Errorf("Expected path '/chat/completions', got %s", r.URL.Path)
// 		}
// 		if r.Method != "POST" {
// 			t.Errorf("Expected method 'POST', got %s", r.Method)
// 		}
// 		if r.Header.Get("Authorization") != "Bearer test-api-key" {
// 			t.Errorf("Expected Authorization header 'Bearer test-api-key', got %s", r.Header.Get("Authorization"))
// 		}
// 		if r.Header.Get("Content-Type") != "application/json" {
// 			t.Errorf("Expected Content-Type header 'application/json', got %s", r.Header.Get("Content-Type"))
// 		}

// 		// Parse the request body
// 		var payload CreateChatSessionPayload
// 		err := json.NewDecoder(r.Body).Decode(&payload)
// 		if err != nil {
// 			t.Errorf("Error decoding request body: %v", err)
// 		}

// 		// Check payload contents
// 		if payload.Model != "gpt-4o-mini" {
// 			t.Errorf("Expected model 'gpt-4o-mini', got %s", payload.Model)
// 		}
// 		if len(payload.Messages) != 1 {
// 			t.Errorf("Expected 1 message, got %d", len(payload.Messages))
// 		}
// 		if payload.Messages[0].Role != "user" {
// 			t.Errorf("Expected role 'user', got %s", payload.Messages[0].Role)
// 		}
// 		if !strings.Contains(payload.Messages[0].Content, "You are a helpful LLM") {
// 			t.Errorf("Expected content to contain 'You are a helpful LLM', got %s", payload.Messages[0].Content)
// 		}

// 		// Respond with a mock response
// 		mockResponse := ChatCompletionResponse{
// 			ID:      "mock-session-id",
// 			Object:  "chat.completion",
// 			Created: 1234567890,
// 			Model:   "gpt-4o-mini",
// 			Choices: []Choice{
// 				{
// 					Index: 0,
// 					Message: Message{
// 						Role:    "assistant",
// 						Content: `[{"event_title": "Mock Event", "event_location": "Mock Location", "event_start_time": "2023-05-01T10:00:00Z"}]`,
// 					},
// 					FinishReason: "stop",
// 				},
// 			},
// 			Usage: map[string]int{"total_tokens": 100},
// 		}

// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(mockResponse)
// 	}))
// 	defer mockServer.Close()

// 	// Set environment variables
// 	os.Setenv("OPENAI_API_BASE_URL", mockServer.URL)
// 	os.Setenv("OPENAI_API_KEY", "test-api-key")

// 	// Call the function
// 	sessionID, messageContent, err := CreateChatSession("test markdown content")

// 	// Check results
// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}
// 	if sessionID != "mock-session-id" {
// 		t.Errorf("Expected session ID 'mock-session-id', got %s", sessionID)
// 	}
// 	expectedContent := `[{"event_title": "Mock Event", "event_location": "Mock Location", "event_start_time": "2023-05-01T10:00:00Z"}]`
// 	if messageContent != expectedContent {
// 		t.Errorf("Expected message content %s, got %s", expectedContent, messageContent)
// 	}
// }

func TestUnpadJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid JSON", `{"key": "value"}`, `{"key":"value"}`},
		{"Padded JSON", ` { "key" : "value" } `, `{"key":"value"}`},
		{"Invalid JSON", `{"key": "value"`, `{"key": "value"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnpadJSON(tt.input)
			if result != tt.expected {
				t.Errorf("UnpadJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
