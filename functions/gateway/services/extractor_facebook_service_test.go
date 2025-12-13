package services

import (
	"context"
	"errors"
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

func TestFacebookExtractorExtractOnboardMode(t *testing.T) {
	extractor := &FacebookExtractor{}

	mockHTML := `<html><body>Test</body></html>`
	mockScraper := &MockScrapingService{
		GetHTMLFromURLWithRetriesFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
			return mockHTML, nil
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_ONBOARD)

	seshuJob := types.SeshuJob{
		NormalizedUrlKey:  "https://www.facebook.com/events/123",
		LocationTimezone:  "America/Chicago",
		LocationLatitude:  0,
		LocationLongitude: 0,
		LocationAddress:   "",
	}

	// This test will fail if FindFacebookEventListData is not mocked
	// For now, we're testing the structure
	_, html, err := extractor.Extract(ctx, seshuJob, mockScraper)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if html != mockHTML {
		t.Errorf("Expected HTML to be %s, got %s", mockHTML, html)
	}

	// Error is expected since FindFacebookEventListData may fail without proper setup
	// The test verifies the flow works correctly
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
	ctx = context.WithValue(ctx, "MODE", "default")

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://www.facebook.com/events/123",
	}

	_, _, err := extractor.Extract(ctx, seshuJob, mockScraper)

	if err == nil {
		t.Error("Expected error when HTML fetch fails, got nil")
	}

	if err.Error() != "failed to get HTML from Facebook URL: failed to fetch HTML" {
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
		EventTitle:    "Test Event",
		EventURL:      "https://facebook.com/event/123",
		EventLatitude: 50.0,
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
}
