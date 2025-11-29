package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

type MockScrapingService struct {
	GetHTMLFromURLFunc            func(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error)
	GetHTMLFromURLWithRetriesFunc func(unescapedURL string, timeout int, jsRender bool, waitFor string, retries int, validate services.ContentValidationFunc) (string, error)
}

func (m *MockScrapingService) GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error) {
	return m.GetHTMLFromURLFunc(unescapedURL, timeout, jsRender, waitFor)
}

func (m *MockScrapingService) GetHTMLFromURLWithRetries(unescapedURL string, timeout int, jsRender bool, waitFor string, retries int, validate services.ContentValidationFunc) (string, error) {
	if m.GetHTMLFromURLWithRetriesFunc != nil {
		return m.GetHTMLFromURLWithRetriesFunc(unescapedURL, timeout, jsRender, waitFor, retries, validate)
	}
	// Fallback behavior: call GetHTMLFromURL directly (you can enhance this logic)
	return m.GetHTMLFromURL(unescapedURL, timeout, jsRender, waitFor)
}

func TestHandleSeshuSessionSubmit_RandomURL(t *testing.T) {
	// Save original environment variables
	originalScrapingBeeAPIBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	originalOpenAIAPIBaseURL := os.Getenv("OPENAI_API_BASE_URL")
	originalOpenAIAPIKey := os.Getenv("OPENAI_API_KEY")

	// Set up mock ScrapingBee server with proper port rotation
	scrapingBeeHostAndPort := test_helpers.GetNextPort()
	mockScrapingBee := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Mock HTML Content</body></html>"))
	}))

	scrapingBeeListener, err := test_helpers.BindToPort(t, scrapingBeeHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind ScrapingBee server: %v", err)
	}
	mockScrapingBee.Listener = scrapingBeeListener
	mockScrapingBee.Start()
	defer mockScrapingBee.Close()

	// Set up mock OpenAI server with proper port rotation
	openAIHostAndPort := test_helpers.GetNextPort()
	mockOpenAI := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := services.ChatCompletionResponse{
			ID:      "mock-session-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4o-mini",
			Choices: []services.Choice{
				{
					Index: 0,
					Message: services.Message{
						Role:    "assistant",
						Content: `[{"event_title":"Mock Event","event_location":"Mock Location","event_start_time":"2023-05-01T10:00:00Z","event_end_time":"2023-05-01T12:00:00Z","event_url":"https://mock-event.com"}]`,
					},
					FinishReason: "stop",
				},
			},
			Usage: services.Usage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))

	openAIListener, err := test_helpers.BindToPort(t, openAIHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind OpenAI server: %v", err)
	}
	mockOpenAI.Listener = openAIListener
	mockOpenAI.Start()
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

	tests := []struct {
		name           string
		method         string
		action         string
		body           string
		expectedStatus int
		expectBodyText string
	}{
		{
			name:           "Valid POST init",
			method:         "POST",
			action:         "init",
			body:           `{"url":"https://example.com"}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "Mock Event",
		},
		{
			name:           "Valid POST recursive",
			method:         "POST",
			action:         "rs",
			body:           `{"url":"https://child.com","parent_url":"https://parent.com"}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "Mock Event",
		},
		{
			name:           "GET not allowed",
			method:         "GET",
			action:         "init",
			body:           "",
			expectedStatus: http.StatusOK,
			expectBodyText: "Method Not Allowed",
		},
		{
			name:           "Missing action param",
			method:         "POST",
			action:         "",
			body:           `{"url":"https://example.com"}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "invalid action",
		},
		{
			name:           "Invalid action param",
			method:         "POST",
			action:         "invalid",
			body:           `{}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "invalid action",
		},
		{
			name:           "Malformed JSON",
			method:         "POST",
			action:         "init",
			body:           `{"url":}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "unprocessable",
		},
		{
			name:           "Init empty body",
			method:         "POST",
			action:         "init",
			body:           "",
			expectedStatus: http.StatusOK,
			expectBodyText: "unprocessable",
		},
		{
			name:           "Init empty URL",
			method:         "POST",
			action:         "init",
			body:           `{"url":""}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "badrequest",
		},
		{
			name:           "Recursive missing parent_url",
			method:         "POST",
			action:         "rs",
			body:           `{"url":"https://child.com"}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "badrequest",
		},
		{
			name:           "Recursive missing url",
			method:         "POST",
			action:         "rs",
			body:           `{"parent_url":"https://parent.com"}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "badrequest",
		},
		{
			name:           "Recursive malformed JSON",
			method:         "POST",
			action:         "rs",
			body:           `{"url":}`,
			expectedStatus: http.StatusOK,
			expectBodyText: "unprocessable",
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

			// Add AWS Lambda context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})

			// Add auth context (required for authorization)
			mockUserInfo := constants.UserInfo{
				Sub:   "test-user-123",
				Name:  "Test User",
				Email: "test@example.com",
			}
			mockRoleClaims := map[string]interface{}{
				"roles": []string{"user"},
			}
			ctx = context.WithValue(ctx, "userInfo", mockUserInfo)
			ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
			ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "test-user-123"})
			req = req.WithContext(ctx)

			// Call HandleSeshuSessionSubmit and get the resulting handler
			handler := HandleSeshuSessionSubmit(httptest.NewRecorder(), req)

			// Call the returned handler
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if tt.expectBodyText != "" && !strings.Contains(rec.Body.String(), tt.expectBodyText) {
				t.Errorf("Expected response to contain %q, got %q", tt.expectBodyText, rec.Body.String())
			}
		})
	}
}

func setScrapingBeeEnv(baseURL string) func() {
	prevBase := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	prevKey := os.Getenv("SCRAPINGBEE_API_KEY")
	_ = os.Setenv("SCRAPINGBEE_API_URL_BASE", baseURL)
	_ = os.Setenv("SCRAPINGBEE_API_KEY", "test-key")
	return func() {
		if prevBase == "" {
			_ = os.Unsetenv("SCRAPINGBEE_API_URL_BASE")
		} else {
			_ = os.Setenv("SCRAPINGBEE_API_URL_BASE", prevBase)
		}
		if prevKey == "" {
			_ = os.Unsetenv("SCRAPINGBEE_API_KEY")
		} else {
			_ = os.Setenv("SCRAPINGBEE_API_KEY", prevKey)
		}
	}
}

func TestHandleSeshuSessionSubmit_Facebook(t *testing.T) {
	// Save/restore env for OpenAI
	origAI := os.Getenv("OPENAI_API_BASE_URL")
	origAIKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		os.Setenv("OPENAI_API_BASE_URL", origAI)
		os.Setenv("OPENAI_API_KEY", origAIKey)
	}()

	type fbCase struct {
		name              string
		sbHandler         func() http.HandlerFunc
		expectBodyText    string
		expectStatus      int
		expectOpenAICalls int
	}

	cases := []fbCase{
		{
			name: "List success (single-line JSON)",
			sbHandler: func() http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`<html><head><meta property="og:url" content="https://www.facebook.com/events/1719880215557306"></head><body><script data-sjs data-content-len="4096">{"__bbox":{"result":{"data":{"event":{"__typename":"Event","name":"FB Mock Event","url":"https://www.facebook.com/events/1719880215557306","day_time_sentence":"2025-10-24T23:00:00Z","contextual_name":"FB Hall","description":"FB Mock Description","event_creator":{"name":"FB Host"}}}}}}</script></body></html>`))
				}
			},
			expectBodyText:    "FB Mock Event",
			expectStatus:      http.StatusOK,
			expectOpenAICalls: 0,
		},
		{
			name: "List requires retry then success",
			sbHandler: func() http.HandlerFunc {
				var count int
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					if count == 0 {
						count++
						// First attempt: script present but no __typename -> triggers retry
						_, _ = w.Write([]byte(`<html><body><script data-sjs data-content-len="1000">{"data":{"event":{}}}</script></body></html>`))
						return
					}
					// Second attempt: success
					_, _ = w.Write([]byte(`<html><head><meta property="og:url" content="https://www.facebook.com/events/1719880215557306"></head><body><script data-sjs data-content-len="4096">{"__bbox":{"result":{"data":{"event":{"__typename":"Event","name":"Retry Event","url":"https://www.facebook.com/events/1719880215557306","day_time_sentence":"2025-10-24T23:00:00-05:00","contextual_name":"Retry Hall","description":"Desc","event_creator":{"name":"Host"}}}}}}</script></body></html>`))
				}
			},
			expectBodyText:    "Retry Event",
			expectStatus:      http.StatusOK,
			expectOpenAICalls: 0,
		},
		{
			name: "List invalid (missing url)",
			sbHandler: func() http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					// __typename present but missing url -> no valid events
					_, _ = w.Write([]byte(`<html><body><script data-sjs data-content-len="4096">{"data":{"edges":[{"node":{"__typename":"Event","name":"No URL Event","day_time_sentence":"2025-10-24T23:00:00Z","contextual_name":"Hall"}}]}}</script></body></html>`))
				}
			},
			expectBodyText:    "no valid events extracted",
			expectStatus:      http.StatusOK,
			expectOpenAICalls: 0,
		},
		{
			name: "List invalid (missing name)",
			sbHandler: func() http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					// Missing name -> invalid
					_, _ = w.Write([]byte(`<html><body><script data-sjs data-content-len="4096">{"data":{"edges":[{"node":{"__typename":"Event","url":"https://www.facebook.com/events/1719880215557306","day_time_sentence":"2025-10-24T23:00:00Z"}}]}}</script></body></html>`))
				}
			},
			expectBodyText:    "no valid events extracted",
			expectStatus:      http.StatusOK,
			expectOpenAICalls: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Per-test OpenAI mock (to count calls)
			openaiCalls := 0
			aiPort := test_helpers.GetNextPort()
			mockOpenAI := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				openaiCalls++
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(services.ChatCompletionResponse{
					ID: "mock", Object: "chat.completion", Created: time.Now().Unix(), Model: "gpt-4o-mini",
					Choices: []services.Choice{{Index: 0, Message: services.Message{Role: "assistant", Content: "[]"}, FinishReason: "stop"}},
					Usage:   services.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
				})
			}))
			aiListener, err := test_helpers.BindToPort(t, aiPort)
			if err != nil {
				t.Fatalf("Failed to bind OpenAI server: %v", err)
			}
			mockOpenAI.Listener = aiListener
			mockOpenAI.Start()
			defer mockOpenAI.Close()

			// Per-test ScrapingBee mock
			sbPort := test_helpers.GetNextPort()
			mockScrapingBee := httptest.NewUnstartedServer(tc.sbHandler())
			sbListener, err := test_helpers.BindToPort(t, sbPort)
			if err != nil {
				t.Fatalf("Failed to bind ScrapingBee server: %v", err)
			}
			mockScrapingBee.Listener = sbListener
			mockScrapingBee.Start()
			defer mockScrapingBee.Close()

			restoreSB := setScrapingBeeEnv(mockScrapingBee.URL)
			defer restoreSB()
			os.Setenv("OPENAI_API_BASE_URL", mockOpenAI.URL)
			os.Setenv("OPENAI_API_KEY", "mock-api-key")

			// Mock DB insert
			db = &test_helpers.MockDynamoDBClient{
				PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
					return &dynamodb.PutItemOutput{}, nil
				},
			}

			// Build request
			body := `{"url":"https://www.facebook.com/events/1719880215557306"}`
			req := httptest.NewRequest(http.MethodPost, "/?action=init", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")

			// Lambda + auth context
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{RequestID: "fb-" + tc.name},
				PathParameters: map[string]string{},
			})
			ctx = context.WithValue(ctx, "userInfo", constants.UserInfo{Sub: "fb-user-1"})
			ctx = context.WithValue(ctx, "roleClaims", map[string]any{"roles": []string{"user"}})
			ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "fb-user-1"})
			req = req.WithContext(ctx)

			// Execute
			rec := httptest.NewRecorder()
			h := HandleSeshuSessionSubmit(rec, req)
			h.ServeHTTP(rec, req)

			if rec.Code != tc.expectStatus {
				t.Fatalf("expected %d, got %d; body=%s", tc.expectStatus, rec.Code, rec.Body.String())
			}
			if tc.expectBodyText != "" && !strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(tc.expectBodyText)) {
				t.Fatalf("expected response to contain %q; body=%s", tc.expectBodyText, rec.Body.String())
			}
			if openaiCalls != tc.expectOpenAICalls {
				t.Fatalf("expected %d OpenAI calls, got %d", tc.expectOpenAICalls, openaiCalls)
			}
		})
	}
}

func TestParsePayload(t *testing.T) {
	t.Run("init", func(t *testing.T) {
		body := `{"url":"https://example.com"}`
		url, parent, child, err := parsePayload("init", body)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if url != "https://example.com" {
			t.Errorf("expected url https://example.com, got %s", url)
		}
		if parent != "" {
			t.Errorf("expected parent empty, got %s", parent)
		}
		if child != "" {
			t.Errorf("expected child empty, got %s", child)
		}
	})

	t.Run("rs", func(t *testing.T) {
		body := `{"url":"https://child.com", "parent_url":"https://parent.com"}`
		url, parent, child, err := parsePayload("rs", body)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if url != "https://child.com" {
			t.Errorf("expected url https://child.com, got %s", url)
		}
		if parent != "https://parent.com" {
			t.Errorf("expected parent https://parent.com, got %s", parent)
		}
		if child != "" {
			t.Errorf("expected child empty, got %s", child)
		}
	})

	t.Run("invalid action", func(t *testing.T) {
		_, _, _, err := parsePayload("unknown", `{}`)
		if err == nil {
			t.Error("expected error for unknown action, got nil")
		}
	})
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
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid JSON", `{"key": "value"}`, `{"key":"value"}`, false},
		{"Padded JSON", ` { "key" : "value" } `, `{"key":"value"}`, false},
		{"Invalid JSON", `{"key": "value"`, `{"key": "value"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := services.UnpadJSON(tt.input)
			if err != nil && !tt.expectError {
				t.Errorf("UnpadJSON() unexpected error = %v", err)
			}
			if err == nil && result != tt.expected {
				t.Errorf("UnpadJSON() = %v, want %v", result, tt.expected)
			}
			if result != tt.expected {
				t.Errorf("UnpadJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseAndValidatePayload(t *testing.T) {
	tests := []struct {
		name        string
		payloadBody string
		payload     interface{}
		expectError bool
	}{
		{
			name:        "Valid JSON and struct",
			payloadBody: `{"url": "https://example.com"}`,
			payload:     &SeshuInputPayload{},
			expectError: false,
		},
		{
			name:        "Invalid JSON",
			payloadBody: `{"url":}`,
			payload:     &SeshuInputPayload{},
			expectError: true,
		},
		{
			name:        "Invalid struct validation",
			payloadBody: `{"url": ""}`,
			payload:     &SeshuInputPayload{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseAndValidatePayload(tt.payloadBody, tt.payload)
			if tt.expectError && err == nil {
				t.Error("parseAndValidatePayload() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("parseAndValidatePayload() unexpected error = %v", err)
			}
		})
	}
}

// Helper function to generate JSON for many events
func generateManyEventsJSON(count int) string {
	events := make([]string, count)
	for i := 0; i < count; i++ {
		events[i] = fmt.Sprintf(`{"event_title":"Event %d","event_location":"Location %d","event_start_time":"2023-05-01T10:00:00Z","event_end_time":"2023-05-01T12:00:00Z","event_url":"https://event-%d.com"}`, i+1, i+1, i+1)
	}
	return "[" + strings.Join(events, ",") + "]"
}

func TestSeshuSessionSubmitEventTruncation(t *testing.T) {
	// Save original environment variables
	originalScrapingBeeAPIBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	originalOpenAIAPIBaseURL := os.Getenv("OPENAI_API_BASE_URL")
	originalOpenAIAPIKey := os.Getenv("OPENAI_API_KEY")

	// Set up mock ScrapingBee server with proper port rotation
	scrapingBeeHostAndPort := test_helpers.GetNextPort()
	mockScrapingBee := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Mock HTML Content</body></html>"))
	}))

	scrapingBeeListener, err := test_helpers.BindToPort(t, scrapingBeeHostAndPort)
	if err != nil {
		t.Fatalf("Failed to bind ScrapingBee server: %v", err)
	}
	mockScrapingBee.Listener = scrapingBeeListener
	mockScrapingBee.Start()
	defer mockScrapingBee.Close()

	// Override environment variables
	os.Setenv("SCRAPINGBEE_API_URL_BASE", mockScrapingBee.URL)
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

	// Base response template - only the Content field will be overridden
	baseResponse := services.ChatCompletionResponse{
		ID:      "mock-session-id",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4o-mini",
		Choices: []services.Choice{
			{
				Index: 0,
				Message: services.Message{
					Role:    "assistant",
					Content: "", // This will be overridden per test case
				},
				FinishReason: "stop",
			},
		},
		Usage: services.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	tests := []struct {
		name           string
		url            string
		openAIResponse string
		expectedEvents int
		expectBodyText string
	}{
		{
			name:           "Zero events returned",
			url:            "https://zero-events.com",
			openAIResponse: `[]`, // Empty events array
			expectedEvents: 0,
			expectBodyText: "", // Empty response should not contain any event text
		},
		{
			name:           "One event returned",
			url:            "https://one-event.com",
			openAIResponse: `[{"event_title":"Single Event","event_location":"Single Location","event_start_time":"2023-05-01T10:00:00Z","event_end_time":"2023-05-01T12:00:00Z","event_url":"https://single-event.com"}]`,
			expectedEvents: 1,
			expectBodyText: "Single Event",
		},
		{
			name:           "99 events returned (truncated to 3)",
			url:            "https://many-events.com",
			openAIResponse: generateManyEventsJSON(99),
			expectedEvents: 3,         // Should be truncated to 3
			expectBodyText: "Event 1", // Should contain first event
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock OpenAI server for this specific test
			openAIHostAndPort := test_helpers.GetNextPort()
			mockOpenAI := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				// Create response by copying base and overriding only the Content field
				response := baseResponse
				response.Choices[0].Message.Content = tt.openAIResponse

				json.NewEncoder(w).Encode(response)
			}))

			openAIListener, err := test_helpers.BindToPort(t, openAIHostAndPort)
			if err != nil {
				t.Fatalf("Failed to bind OpenAI server: %v", err)
			}
			mockOpenAI.Listener = openAIListener
			mockOpenAI.Start()
			defer mockOpenAI.Close()

			// Override OpenAI URL for this test
			os.Setenv("OPENAI_API_BASE_URL", mockOpenAI.URL)

			// Create request
			body := fmt.Sprintf(`{"url":"%s"}`, tt.url)
			req := httptest.NewRequest(http.MethodPost, "/?action=init", bytes.NewBuffer([]byte(body)))
			req.Header.Set("Content-Type", "application/json")

			// Add AWS Lambda context (required for transport layer)
			ctx := context.WithValue(req.Context(), constants.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
				PathParameters: map[string]string{},
			})

			// Add auth context (required for authorization)
			mockUserInfo := constants.UserInfo{
				Sub:   "test-user-123",
				Name:  "Test User",
				Email: "test@example.com",
			}
			mockRoleClaims := map[string]interface{}{
				"roles": []string{"user"},
			}
			ctx = context.WithValue(ctx, "userInfo", mockUserInfo)
			ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
			ctx = context.WithValue(ctx, constants.MNM_OPTIONS_CTX_KEY, map[string]string{"userId": "test-user-123"})
			req = req.WithContext(ctx)

			// Call HandleSeshuSessionSubmit and get the resulting handler
			handler := HandleSeshuSessionSubmit(httptest.NewRecorder(), req)

			// Call the returned handler
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Check status
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
			}

			// Check response content
			responseBody := rec.Body.String()

			if tt.expectedEvents == 0 {
				// For zero events, should not contain any event text
				if strings.Contains(responseBody, "Event") {
					t.Errorf("Expected empty response for zero events, but found event text: %s", responseBody)
				}
			} else {
				// For non-zero events, should contain expected text
				if tt.expectBodyText != "" && !strings.Contains(responseBody, tt.expectBodyText) {
					t.Errorf("Expected response to contain %q, got %q", tt.expectBodyText, responseBody)
				}
			}

			// Count actual events in response by counting card elements
			eventCount := strings.Count(responseBody, "checkbox-card card card-compact")
			if eventCount != tt.expectedEvents {
				t.Errorf("Expected %d events in response, found %d", tt.expectedEvents, eventCount)
			}
		})
	}
}
