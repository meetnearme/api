package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

// mockScraper implements ScrapingService for tests
type mockScraper struct {
	html        string
	htmlRetries string
}

func (m *mockScraper) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return m.html, nil
}
func (m *mockScraper) GetHTMLFromURLWithRetries(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, retries int, validate ContentValidationFunc) (string, error) {
	return m.htmlRetries, nil
}

const (
	facebookListHTMLTemplate  = `<html><head></head><body><script data-sjs data-content-len>{"data":{"edges":[{"node":{"__typename":"Event","name":"List Event","url":"https://www.facebook.com/events/123","day_time_sentence":"2025-09-11T10:00:00Z","contextual_name":"","description":"","event_creator":{"name":"List Host"}}}]}}</script></body></html>`
	facebookChildHTMLTemplate = `<html><head><meta property="og:title" content="Child Title" /></head><body><script data-sjs data-content-len>{"__typename":"Event","url":"https://www.facebook.com/events/123","day_time_sentence":"2025-09-11T12:00:00Z","event_description":{"text":"Child Description"},"one_line_address":"Child Venue","event_creator":{"__typename":"User","name":"Child Host"}}</script></body></html>`
)

type mockScrapingService struct {
	responses map[string]string
	calls     []mockScrapeCall
}

type mockScrapeCall struct {
	job        types.SeshuJob
	waitMs     int
	jsRender   bool
	waitFor    string
	maxRetries int
}

func TestGetHTMLFromURL(t *testing.T) {
	testCases := []struct {
		name         string
		value        string
		expectedHTML string
		expectedErr  error
	}{
		{
			name:         "Pre-escaped URL",
			value:        "https%3A%2F%example.com%2Fpath%3Fquery%3Dvalue",
			expectedHTML: "",
			expectedErr:  fmt.Errorf(URLEscapedErrorMsg),
		},
		{
			name:         "Correctly escaped URL value",
			value:        "https://example.com/path?query=value",
			expectedHTML: basicHTMLresp,
			expectedErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock server with proper port rotation
			hostAndPort := test_helpers.GetNextPort()
			mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(basicHTMLresp))
			}))

			listener, err := test_helpers.BindToPort(t, hostAndPort)
			if err != nil {
				t.Fatalf("BindToPort failed: %v", err)
			}
			mockServer.Listener = listener
			mockServer.Start()
			defer mockServer.Close()

			// Use the mock server URL for testing
			baseURL := mockServer.URL

			html, err := GetHTMLFromURLWithBase(baseURL, tc.value, 10, true, "", 1, nil)

			if html != tc.expectedHTML {
				t.Fatalf("Expected %v, got %v", tc.expectedHTML, html)
			}

			// if we expect `nil` and get `nil`, return early, we want to avoid
			// calling `err.Error()` on a `nil` value below
			if tc.expectedErr == nil && err == nil {
				return
			}

			if err.Error() != tc.expectedErr.Error() {
				t.Fatalf("Expected %v, got %v", tc.expectedErr, err)
			}

		})
	}
}

func TestGetHTMLFromURLWithBase_Non200Response(t *testing.T) {
	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "bad gateway")
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/failure", 0, false, "", 1, nil)

	if html != "" {
		t.Fatalf("Expected empty HTML on non-200 response, got %v", html)
	}

	if err == nil {
		t.Fatalf("Expected error for non-200 response, got nil")
	}

	expectedErr := fmt.Sprintf("ERR: %v from scraping service for URL %s", http.StatusBadGateway, baseURL)
	if err.Error() != expectedErr {
		t.Fatalf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestGetHTMLFromURLWithBase_ContentValidationRetries(t *testing.T) {
	var attemptCount int32
	finalHTML := "<html><body>READY</body></html>"

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentAttempt := atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusOK)
		if currentAttempt == 1 {
			fmt.Fprint(w, basicHTMLresp)
			return
		}
		fmt.Fprint(w, finalHTML)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	validationFunc := func(html string) bool {
		return strings.Contains(html, "READY")
	}

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/retry", 0, false, "", 3, validationFunc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if html != finalHTML {
		t.Fatalf("Expected final HTML %q, got %q", finalHTML, html)
	}

	if atomic.LoadInt32(&attemptCount) < 2 {
		t.Fatalf("Expected at least two attempts, got %d", attemptCount)
	}
}

func TestGetHTMLFromURLWithBase_RequestError(t *testing.T) {
	hostAndPort := test_helpers.GetNextPort()
	defer test_helpers.ReleasePort(hostAndPort)

	baseURL := "http://" + hostAndPort

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/unreachable", 0, false, "", 1, nil)

	if html != "" {
		t.Fatalf("Expected empty HTML string when request fails, got %v", html)
	}

	if err == nil {
		t.Fatalf("Expected error due to request failure, got nil")
	}

	if !strings.Contains(err.Error(), "ERR: executing scraping request") {
		t.Fatalf("Expected executing scraping request error, got %v", err)
	}
}

func TestGetHTMLFromURLWithBase_IncludesQueryParameters(t *testing.T) {
	t.Setenv("SCRAPINGBEE_API_KEY", "test-api-key")
	waitMs := 2750
	waitForSelector := "div#content > span.value"

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if got := query.Get("api_key"); got != "test-api-key" {
			t.Fatalf("Expected api_key to be %q, got %q", "test-api-key", got)
		}
		if got := query.Get("url"); got != "https://example.com/params" {
			t.Fatalf("Expected url query to be preserved, got %q", got)
		}
		if got := query.Get("render_js"); got != "true" {
			t.Fatalf("Expected render_js query to be true, got %q", got)
		}
		if got := query.Get("wait"); got != fmt.Sprint(waitMs) {
			t.Fatalf("Expected wait query to be %d, got %q", waitMs, got)
		}
		if got := query.Get("wait_for"); got != waitForSelector {
			t.Fatalf("Expected wait_for query to be %q, got %q", waitForSelector, got)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, basicHTMLresp)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/params", waitMs, true, waitForSelector, 1, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if html != basicHTMLresp {
		t.Fatalf("Expected HTML response to match test fixture, got %v", html)
	}
}

func TestGetHTMLFromURLWithBase_NoOptionalQueryParamsWhenDefaults(t *testing.T) {
	baseWait := 0
	jsRender := false
	waitForSelector := ""

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if query.Has("render_js") {
			t.Fatalf("Did not expect render_js parameter when jsRender is %v", jsRender)
		}
		if query.Has("wait") {
			t.Fatalf("Did not expect wait parameter when waitMs is %d", baseWait)
		}
		if query.Has("wait_for") {
			t.Fatalf("Did not expect wait_for parameter when waitFor is empty")
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, basicHTMLresp)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/defaults", baseWait, jsRender, waitForSelector, 1, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if html != basicHTMLresp {
		t.Fatalf("Expected HTML response to match test fixture, got %v", html)
	}
}

func TestGetHTMLFromURLWithBase_ContentValidationAlwaysFailsReturnsLastHTML(t *testing.T) {
	var attemptCount int32

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, basicHTMLresp)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	validationFunc := func(html string) bool {
		return false
	}

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/retry", 0, false, "", 2, validationFunc)
	if err != nil {
		t.Fatalf("Expected no error when validation fails, got %v", err)
	}

	if html != basicHTMLresp {
		t.Fatalf("Expected final HTML %q, got %q", basicHTMLresp, html)
	}

	if got := atomic.LoadInt32(&attemptCount); got != 2 {
		t.Fatalf("Expected two attempts due to retries, got %d", got)
	}
}

func TestGetHTMLFromURLWithBase_ContentValidationStopsAfterSuccess(t *testing.T) {
	var attemptCount int32

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, basicHTMLresp)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	validationFunc := func(html string) bool {
		return true
	}

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/early-success", 0, false, "", 4, validationFunc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if html != basicHTMLresp {
		t.Fatalf("Expected HTML response to match test fixture, got %v", html)
	}

	if got := atomic.LoadInt32(&attemptCount); got != 1 {
		t.Fatalf("Expected single attempt after validation success, got %d", got)
	}
}

func TestGetHTMLFromURLWithBase_DefaultsRetryToOne(t *testing.T) {
	var attemptCount int32

	hostAndPort := test_helpers.GetNextPort()
	mockServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, basicHTMLresp)
	}))

	listener, err := test_helpers.BindToPort(t, hostAndPort)
	if err != nil {
		t.Fatalf("BindToPort failed: %v", err)
	}
	mockServer.Listener = listener
	mockServer.Start()
	defer mockServer.Close()

	baseURL := mockServer.URL

	html, err := GetHTMLFromURLWithBase(baseURL, "https://example.com/default", 0, false, "", 0, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if html != basicHTMLresp {
		t.Fatalf("Expected HTML response to match test fixture, got %v", html)
	}

	if got := atomic.LoadInt32(&attemptCount); got != 1 {
		t.Fatalf("Expected one attempt when maxRetries is zero, got %d", got)
	}
}

// New: verify in SCRAPE mode, non-Facebook uses OpenAI, Facebook does not.
func TestExtractEventsFromHTML_ScrapeMode(t *testing.T) {

	// OpenAI mock that counts calls
	makeAI := func(resp string) (base string, cleanup func(), calls *int32) {
		var c int32
		host := test_helpers.GetNextPort()
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&c, 1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ChatCompletionResponse{
				ID: "mock", Object: "chat.completion", Created: time.Now().Unix(), Model: "gpt-4o-mini",
				Choices: []Choice{{Index: 0, Message: Message{Role: "assistant", Content: resp}, FinishReason: "stop"}},
				Usage:   Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
			})
		}))
		l, err := test_helpers.BindToPort(t, host)
		if err != nil {
			t.Fatalf("bind AI: %v", err)
		}
		srv.Listener = l
		srv.Start()
		return srv.URL, func() { srv.Close() }, &c
	}

	t.Run("Random URL uses OpenAI in ONBOARD", func(t *testing.T) {
		aiURL, aiClose, aiCalls := makeAI(`[{"event_title":"Title","event_location":"Loc","event_start_time":"2025-01-01T10:00:00Z","event_end_time":"2025-01-01T12:00:00Z","event_url":"https://x.com"}]`)
		defer aiClose()
		prevAI, prevKey := os.Getenv("OPENAI_API_BASE_URL"), os.Getenv("OPENAI_API_KEY")
		os.Setenv("OPENAI_API_BASE_URL", aiURL)
		os.Setenv("OPENAI_API_KEY", "k")
		defer func() { os.Setenv("OPENAI_API_BASE_URL", prevAI); os.Setenv("OPENAI_API_KEY", prevKey) }()

		ms := &mockScraper{html: "<html><body>hello</body></html>"}
		job := types.SeshuJob{NormalizedUrlKey: "https://example.com/page"}
		evs, _, err := ExtractEventsFromHTML(job, constants.SESHU_MODE_ONBOARD, "init", ms)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if atomic.LoadInt32(aiCalls) == 0 {
			t.Fatalf("expected OpenAI to be called")
		}
		if len(evs) == 0 {
			t.Fatalf("expected some events from OpenAI path")
		}
	})

	t.Run("Facebook URL avoids OpenAI", func(t *testing.T) {
		// FB single-line JSON in required script tag
		fbJSON := `{"__bbox":{"result":{"data":{"event":{"__typename":"Event","name":"FB","url":"https://www.facebook.com/events/1","day_time_sentence":"2025-01-01T10:00:00Z"}}}}}`
		html := `<html><head><meta property="og:url" content="https://www.facebook.com/events/1"></head><body><script data-sjs data-content-len="4096">` + fbJSON + `</script></body></html>`
		ms := &mockScraper{htmlRetries: html}
		// AI mock should not be hit
		aiURL, aiClose, aiCalls := makeAI("[]")
		defer aiClose()
		prevAI, prevKey := os.Getenv("OPENAI_API_BASE_URL"), os.Getenv("OPENAI_API_KEY")
		os.Setenv("OPENAI_API_BASE_URL", aiURL)
		os.Setenv("OPENAI_API_KEY", "k")
		defer func() { os.Setenv("OPENAI_API_BASE_URL", prevAI); os.Setenv("OPENAI_API_KEY", prevKey) }()

		job := types.SeshuJob{NormalizedUrlKey: "https://www.facebook.com/events/1"}
		evs, _, err := ExtractEventsFromHTML(job, constants.SESHU_MODE_SCRAPE, "init", ms)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if atomic.LoadInt32(aiCalls) != 0 {
			t.Fatalf("expected no OpenAI calls for FB, got %d", atomic.LoadInt32(aiCalls))
		}
		if len(evs) == 0 {
			t.Fatalf("expected events for FB path")
		}
	})
}

func TestDeriveTimezoneFromCoordinates(t *testing.T) {
	testCases := []struct {
		name           string
		lat            float64
		lng            float64
		expectedResult string
		description    string
	}{
		{
			name:           "Empty input coordinates",
			lat:            0,
			lng:            0,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "Out of range latitude coordinates",
			lat:            91,
			lng:            0,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "Out of range longitude coordinates",
			lat:            0,
			lng:            181,
			expectedResult: "",
			description:    "Should return empty string for zero coordinates",
		},
		{
			name:           "United States coordinates",
			lat:            37.875580,
			lng:            -92.473411,
			expectedResult: "America/Chicago", // Missouri, USA
			description:    "Should return America/Chicago timezone for Missouri coordinates",
		},
		{
			name:           "United Kingdom coordinates",
			lat:            52.282165,
			lng:            -0.891387,
			expectedResult: "Europe/London", // England, UK
			description:    "Should return Europe/London timezone for England coordinates",
		},
		{
			name:           "Australia coordinates",
			lat:            -27.507454,
			lng:            144.479705,
			expectedResult: "Australia/Brisbane", // Queensland, Australia
			description:    "Should return Australia/Brisbane timezone for Queensland coordinates",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DeriveTimezoneFromCoordinates(tc.lat, tc.lng)

			if result != tc.expectedResult {
				t.Errorf("Expected timezone '%s', got '%s' for coordinates (%.6f, %.6f) - %s",
					tc.expectedResult, result, tc.lat, tc.lng, tc.description)
			}

			// Log the result for verification
			t.Logf("Coordinates (%.6f, %.6f) -> Timezone: '%s'", tc.lat, tc.lng, result)
		})
	}
}

func TestDeriveTimezoneFromCoordinatesInitialEmpty(t *testing.T) {
	result := DeriveTimezoneFromCoordinates(constants.INITIAL_EMPTY_LAT_LONG, 10)
	if result != "" {
		t.Fatalf("Expected empty string for INITIAL_EMPTY_LAT_LONG latitude, got %q", result)
	}

	result = DeriveTimezoneFromCoordinates(10, constants.INITIAL_EMPTY_LAT_LONG)
	if result != "" {
		t.Fatalf("Expected empty string for INITIAL_EMPTY_LAT_LONG longitude, got %q", result)
	}
}

func TestDeriveTimezoneFromCoordinatesNoFinder(t *testing.T) {
	originalFinder := tzfFinder
	tzfFinder = nil
	defer func() {
		tzfFinder = originalFinder
	}()

	result := DeriveTimezoneFromCoordinates(40.7128, -74.0060)
	if result != "" {
		t.Fatalf("Expected empty string when tzf finder is unavailable, got %q", result)
	}
}

type mockTZFFinder struct {
	response string
}

func (m mockTZFFinder) GetTimezoneName(lng, lat float64) string {
	return m.response
}

func (m mockTZFFinder) DataVersion() string {
	return "test-version"
}

func (m mockTZFFinder) GetTimezoneNames(lng, lat float64) ([]string, error) {
	if m.response == "" {
		return nil, nil
	}
	return []string{m.response}, nil
}

func (m mockTZFFinder) TimezoneNames() []string {
	if m.response == "" {
		return nil
	}
	return []string{m.response}
}

func TestDeriveTimezoneFromCoordinatesEmptyResult(t *testing.T) {
	originalFinder := tzfFinder
	tzfFinder = mockTZFFinder{response: ""}
	defer func() {
		tzfFinder = originalFinder
	}()

	result := DeriveTimezoneFromCoordinates(51.5074, -0.1278)
	if result != "" {
		t.Fatalf("Expected empty string when tzf returns empty value, got %q", result)
	}
}

func (m *mockScrapingService) GetHTMLFromURL(job types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return "", fmt.Errorf("unexpected GetHTMLFromURL call for %s", job.NormalizedUrlKey)
}

func (m *mockScrapingService) GetHTMLFromURLWithRetries(job types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
	resp, ok := m.responses[job.NormalizedUrlKey]
	if !ok {
		return "", fmt.Errorf("unexpected URL requested: %s", job.NormalizedUrlKey)
	}

	if validationFunc != nil && !validationFunc(resp) {
		return "", fmt.Errorf("validation failed for %s", job.NormalizedUrlKey)
	}

	m.calls = append(m.calls, mockScrapeCall{
		job:        job,
		waitMs:     waitMs,
		jsRender:   jsRender,
		waitFor:    waitFor,
		maxRetries: maxRetries,
	})

	return resp, nil
}

func TestExtractEventsFromHTML_FacebookListTriggersChildScrape(t *testing.T) {
	t.Setenv("GO_ENV", "test")

	mockService := &mockScrapingService{
		responses: map[string]string{
			"https://www.facebook.com/somepage/events": facebookListHTMLTemplate,
			"https://www.facebook.com/events/123":      facebookChildHTMLTemplate,
		},
	}

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/somepage/events",
		LocationTimezone: "America/Chicago",
	}

	events, htmlContent, err := ExtractEventsFromHTML(seshuJob, constants.SESHU_MODE_SCRAPE, "", mockService)
	if err != nil {
		t.Fatalf("Expected no error extracting Facebook events, got %v", err)
	}

	if htmlContent != facebookListHTMLTemplate {
		t.Fatalf("Expected facebook list HTML to be returned, got %q", htmlContent)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event from Facebook list, got %d", len(events))
	}

	if events[0].EventDescription != "Child Description" {
		t.Fatalf("Expected event description to be hydrated from child scrape, got %q", events[0].EventDescription)
	}

	if events[0].EventLocation != "Child Venue" {
		t.Fatalf("Expected event location to be hydrated from child scrape, got %q", events[0].EventLocation)
	}

	if len(mockService.calls) != 2 {
		t.Fatalf("Expected 2 scraping calls (list + child), got %d", len(mockService.calls))
	}

	rootCall := mockService.calls[0]
	if rootCall.job.NormalizedUrlKey != seshuJob.NormalizedUrlKey {
		t.Fatalf("First scrape should target the list URL, got %s", rootCall.job.NormalizedUrlKey)
	}

	if rootCall.waitMs != 7500 || !rootCall.jsRender || rootCall.waitFor != "script[data-sjs][data-content-len]" || rootCall.maxRetries != 7 {
		t.Fatalf("Unexpected parameters on list scrape: %+v", rootCall)
	}

	childCall := mockService.calls[1]
	if childCall.job.NormalizedUrlKey != "https://www.facebook.com/events/123" {
		t.Fatalf("Second scrape should target child event URL, got %s", childCall.job.NormalizedUrlKey)
	}
}

func TestExtractEventsFromHTML_FacebookOnboardSkipsChildScrapes(t *testing.T) {
	mockService := &mockScrapingService{
		responses: map[string]string{
			"https://www.facebook.com/somepage/events": facebookListHTMLTemplate,
		},
	}

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/somepage/events",
		LocationTimezone: "America/Chicago",
	}

	events, _, err := ExtractEventsFromHTML(seshuJob, constants.SESHU_MODE_ONBOARD, "", mockService)
	if err != nil {
		t.Fatalf("Expected no error extracting Facebook events in onboard mode, got %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event from Facebook list in onboard mode, got %d", len(events))
	}

	if len(mockService.calls) != 1 {
		t.Fatalf("Expected single scrape call in onboard mode, got %d", len(mockService.calls))
	}
}
