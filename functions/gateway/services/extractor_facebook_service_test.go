package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestFacebookExtractorCanHandle(t *testing.T) {
	extractor := &FacebookExtractor{}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Facebook events URL",
			url:      "https://www.facebook.com/events/123456",
			expected: true,
		},
		{
			name:     "Facebook without www",
			url:      "https://facebook.com/events/789",
			expected: true,
		},
		{
			name:     "Non-Facebook URL",
			url:      "https://eventbrite.com/events",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.CanHandle(tt.url)
			if result != tt.expected {
				t.Errorf("CanHandle(%s) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestFacebookExtractorExtractHTMLFetchError(t *testing.T) {
	extractor := &FacebookExtractor{}

	mockError := errors.New("failed to fetch HTML")
	mockScraper := &MockScrapingService{
		GetHTMLFromURLWithRetriesFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
			return "", mockError
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_SCRAPE)

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/events/123",
	}

	_, _, err := extractor.Extract(ctx, seshuJob, mockScraper)

	if err == nil {
		t.Error("Expected error when HTML fetch fails, got nil")
	}

	if !strings.Contains(err.Error(), "failed to get HTML from Facebook URL") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestFacebookExtractorLocationDataApplication(t *testing.T) {
	extractor := &FacebookExtractor{}

	event := &types.EventInfo{
		EventTitle:    "Test Event",
		EventURL:      "https://facebook.com/event/123",
		EventLocation: "",
	}

	seshuJob := types.SeshuJob{
		LocationLatitude:  40.7128,
		LocationLongitude: -74.0060,
		LocationAddress:   "New York, NY",
	}

	extractor.applyLocationData(event, seshuJob)

	if event.EventLatitude != 40.7128 {
		t.Errorf("Expected latitude 40.7128, got %f", event.EventLatitude)
	}

	if event.EventLongitude != -74.0060 {
		t.Errorf("Expected longitude -74.0060, got %f", event.EventLongitude)
	}

	if event.EventLocation != "New York, NY" {
		t.Errorf("Expected location 'New York, NY', got %s", event.EventLocation)
	}
}

func TestFacebookExtractorLocationDataNotAppliedWhenEmpty(t *testing.T) {
	extractor := &FacebookExtractor{}

	event := &types.EventInfo{
		EventTitle:     "Test Event",
		EventURL:       "https://facebook.com/event/123",
		EventLatitude:  50.0,
		EventLongitude: 50.0,
	}

	seshuJob := types.SeshuJob{
		LocationLatitude:  constants.INITIAL_EMPTY_LAT_LONG,
		LocationLongitude: constants.INITIAL_EMPTY_LAT_LONG,
		LocationAddress:   "",
	}

	extractor.applyLocationData(event, seshuJob)

	if event.EventLatitude != 50.0 {
		t.Errorf("Expected latitude to remain 50.0, got %f", event.EventLatitude)
	}

	if event.EventLongitude != 50.0 {
		t.Errorf("Expected longitude to remain 50.0, got %f", event.EventLongitude)
	}
}

func TestFacebookExtractorExtractValidatesRetry(t *testing.T) {
	extractor := &FacebookExtractor{}

	// Mock that validates the validation function is passed
	validationFuncCalled := false
	mockScraper := &MockScrapingService{
		GetHTMLFromURLWithRetriesFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
			if validationFunc != nil {
				validationFuncCalled = true
				// Test the validation function
				result := validationFunc(`{"__typename":"Event"}`)
				if !result {
					t.Error("Validation function should return true for valid Facebook event data")
				}
			}
			return "", errors.New("no Facebook event data found")
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_SCRAPE)

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/events/123",
	}

	_, _, _ = extractor.Extract(ctx, seshuJob, mockScraper)

	if !validationFuncCalled {
		t.Error("Validation function should have been called")
	}
}

func TestFacebookExtractorHTMLFetchParameters(t *testing.T) {
	extractor := &FacebookExtractor{}

	capturedParams := struct {
		waitMs     int
		jsRender   bool
		waitFor    string
		maxRetries int
	}{}

	mockScraper := &MockScrapingService{
		GetHTMLFromURLWithRetriesFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
			capturedParams.waitMs = waitMs
			capturedParams.jsRender = jsRender
			capturedParams.waitFor = waitFor
			capturedParams.maxRetries = maxRetries
			return "", errors.New("test error")
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_SCRAPE)

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/events/123",
	}

	_, _, _ = extractor.Extract(ctx, seshuJob, mockScraper)

	if capturedParams.waitMs != 7500 {
		t.Errorf("Expected waitMs 7500, got %d", capturedParams.waitMs)
	}

	if !capturedParams.jsRender {
		t.Error("Expected jsRender to be true")
	}

	if capturedParams.waitFor != "script[data-sjs][data-content-len]" {
		t.Errorf("Expected specific waitFor selector, got %s", capturedParams.waitFor)
	}

	if capturedParams.maxRetries != 7 {
		t.Errorf("Expected maxRetries 7, got %d", capturedParams.maxRetries)
	}
}
