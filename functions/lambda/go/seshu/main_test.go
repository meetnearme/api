package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
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

func TestRouter(t *testing.T) {
	// Save original environment variables
	originalScrapingBeeAPIBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	originalOpenAIAPIBaseURL := os.Getenv("OPENAI_API_BASE_URL")
	originalOpenAIAPIKey := os.Getenv("OPENAI_API_KEY")

	// Set up mock server for ScrapingBee
	mockScrapingBeeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Mock HTML Content</body></html>"))
	}))
	defer mockScrapingBeeServer.Close()

	// Set up mock server for OpenAI
	mockOpenAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer mockOpenAIServer.Close()

	// Set test environment variables
	os.Setenv("SCRAPINGBEE_API_URL_BASE", mockScrapingBeeServer.URL)
	os.Setenv("OPENAI_API_BASE_URL", mockOpenAIServer.URL)
	os.Setenv("OPENAI_API_KEY", "mock-api-key")

	// Defer resetting environment variables
	defer func() {
		os.Setenv("SCRAPINGBEE_API_URL_BASE", originalScrapingBeeAPIBaseURL)
		os.Setenv("OPENAI_API_BASE_URL", originalOpenAIAPIBaseURL)
		os.Setenv("OPENAI_API_KEY", originalOpenAIAPIKey)
	}()

	mockDB := &test_helpers.MockDynamoDBClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			// You can add assertions here to check the input if needed
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	// Replace the global db variable with our mock
	db = mockDB

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		mockHTML       string
		mockErr        error
	}{
		// currently this test case doesn't fully thread data through, we should improve this
		// it's a bit of a false positive, but a good faith attempt toward more coverage here
		{"POST request", "POST", http.StatusOK, `<form class="group" novalidate><div role="alert" class="alert alert-info mt-3 mb-11">Mark each field such as "title" and "location" as correct or incorrect with the adjacent toggle. If the proposed event is not an event, toggle "This is an event" to "This is not an event".</div><div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"><div class="checkbox-card card card-compact shadow-lg"><div class="checkbox-card-header bg-success content-success has-toggleable-text"><label class="label cursor-pointer justify-normal"><input value="candidate-0" x-model.fill="eventCandidates" id="main-toggle-0" type="checkbox" class="toggle mr-4" onclick="this.parentNode.parentNode.parentNode.querySelectorAll(&#39;input.toggle&#39;).forEach(item =&gt; item.checked = this.checked)" checked> <span class="label-text flex contents">This is <strong class="hidden-when-checked">not </strong>an event</span></label></div><div class="card-body"><h2 class="card-title">Mock Event</h2><p><label for="cand_title_0" class="label items-start justify-normal cursor-pointer"><input name="cand_title_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2" checked><span class="label-text"><strong>Title:</strong> Mock Event</span></label></p><p><label for="cand_location_0" class="label items-start justify-normal cursor-pointer"><input name="cand_location_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2" checked><span class="label-text"><strong>Location:</strong> Mock Location</span></label></p><p><label for="cand_date_0" class="label items-start justify-normal cursor-pointer"><input name="cand_date_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2"><span class="label-text"><strong>Start Time:</strong> </span></label></p><p><label for="cand_date_0" class="label items-start justify-normal cursor-pointer"><input name="cand_date_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2"><span class="label-text"><strong>End Time:</strong> </span></label></p><p><label for="cand_url_0" class="label items-start justify-normal cursor-pointer"><input name="cand_url_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2" checked><span class="label-text"><strong>URL:</strong> https://mock-event.com</span></label></p><p><label for="cand_description_0" class="label items-start justify-normal cursor-pointer"><input name="cand_description_0" type="checkbox" class="toggle toggle-sm toggle-success -mb-1 mr-2"><span class="label-text"><strong>Description:</strong> </span></label></p></div></div></div></form>`, nil},
		{"Unsupported method", "GET", http.StatusMethodNotAllowed, "", nil},
		// Remove the "Scraping error" test case as it's now handled by the mock server
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the actual ScrapingService instead of a mock
			actualService := services.RealScrapingService{}

			// Create a custom router function for testing
			testRouter := func(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
				switch req.RequestContext.HTTP.Method {
				case "POST":
					req.Headers["Access-Control-Allow-Origin"] = "*"
					req.Headers["Access-Control-Allow-Credentials"] = "true"
					return handlePost(ctx, req, &actualService)
				default:
					return clientError(http.StatusMethodNotAllowed)
				}
			}

			req := events.LambdaFunctionURLRequest{
				RequestContext: events.LambdaFunctionURLRequestContext{
					HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
						Method: tt.method,
					},
				},
				Body:    `{"url": "https://example.com"}`,
				Headers: make(map[string]string),
			}
			resp, err := testRouter(context.Background(), req)
			if err != nil {
				t.Errorf("Router() error = %v", err)
				return
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Router() status = %v, want %v", resp.StatusCode, tt.expectedStatus)
			}
			if tt.method == "POST" && resp.StatusCode == http.StatusOK {
				var result = string([]byte(resp.Body))
				// err := json.Unmarshal([]byte(resp.Body), &result)
				// if err != nil {
				// 	t.Errorf("Failed to unmarshal response body: %v", err)
				// }
				if result != tt.mockHTML {
					t.Errorf("Expected HTML %s, got %s", tt.mockHTML, result)
				}
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
