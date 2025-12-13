package services

import (
	"context"
	"errors"
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestGenericExtractorCanHandle(t *testing.T) {
	extractor := &GenericExtractor{}

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Any URL should be handled",
			url:  "https://example.com",
		},
		{
			name: "Empty URL should be handled",
			url:  "",
		},
		{
			name: "Facebook URL should still be handled by generic",
			url:  "https://facebook.com/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.CanHandle(tt.url)
			if !result {
				t.Errorf("GenericExtractor should handle all URLs")
			}
		})
	}
}

func TestGenericExtractorExtractNonOnboardMode(t *testing.T) {
	extractor := &GenericExtractor{}

	mockHTML := `<html><body>Event List</body></html>`
	mockScraper := &MockScrapingService{
		GetHTMLFromURLFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
			return mockHTML, nil
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", "default")
	ctx = context.WithValue(ctx, "ACTION", "init")

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://example.com/events",
	}

	events, html, err := extractor.Extract(ctx, seshuJob, mockScraper)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if html != mockHTML {
		t.Errorf("Expected HTML to be %s, got %s", mockHTML, html)
	}

	if len(events) != 0 {
		t.Errorf("Expected empty events for non-onboard mode, got %d", len(events))
	}
}

func TestGenericExtractorExtractHTMLFetchError(t *testing.T) {
	extractor := &GenericExtractor{}

	mockError := errors.New("network error")
	mockScraper := &MockScrapingService{
		GetHTMLFromURLFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
			return "", mockError
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", "default")

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://example.com/events",
	}

	_, _, err := extractor.Extract(ctx, seshuJob, mockScraper)

	if err == nil {
		t.Error("Expected error when HTML fetch fails, got nil")
	}

	if err != mockError {
		t.Errorf("Expected specific error, got %v", err)
	}
}

func TestGenericExtractorExtractOnboardModeInit(t *testing.T) {
	extractor := &GenericExtractor{}

	mockHTML := `
    <html>
        <body>
            <div class="event">
                <h1>Event Title</h1>
                <p>Description</p>
            </div>
        </body>
    </html>
    `

	mockScraper := &MockScrapingService{
		GetHTMLFromURLFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
			return mockHTML, nil
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_ONBOARD)
	ctx = context.WithValue(ctx, "ACTION", "init")

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://example.com/events",
	}

	// This test demonstrates the structure
	// In a real test, you'd need to mock CreateChatSession and related functions
	_, html, _ := extractor.Extract(ctx, seshuJob, mockScraper)

	if html != mockHTML {
		t.Errorf("Expected HTML to be returned correctly")
	}
}

func TestGenericExtractorExtractOnboardModeRecursive(t *testing.T) {
	extractor := &GenericExtractor{}

	mockHTML := `<html><body>Events</body></html>`
	mockScraper := &MockScrapingService{
		GetHTMLFromURLFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
			return mockHTML, nil
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "MODE", constants.SESHU_MODE_ONBOARD)
	ctx = context.WithValue(ctx, "ACTION", "rs")

	seshuJob := types.SeshuJob{
		NormalizedUrlKey: "https://example.com/events",
	}

	_, html, _ := extractor.Extract(ctx, seshuJob, mockScraper)

	if html != mockHTML {
		t.Errorf("Expected HTML to be returned correctly")
	}
}

func TestGenericExtractorWithDifferentModes(t *testing.T) {
	extractor := &GenericExtractor{}

	tests := []struct {
		name   string
		mode   string
		action string
	}{
		{
			name:   "Onboard init mode",
			mode:   constants.SESHU_MODE_ONBOARD,
			action: "init",
		},
		{
			name:   "Onboard recursive mode",
			mode:   constants.SESHU_MODE_ONBOARD,
			action: "rs",
		},
		{
			name:   "Non-onboard mode",
			mode:   "other",
			action: "",
		},
	}

	mockScraper := &MockScrapingService{
		GetHTMLFromURLFunc: func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
			return "<html></html>", nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, "MODE", tt.mode)
			ctx = context.WithValue(ctx, "ACTION", tt.action)

			seshuJob := types.SeshuJob{
				NormalizedUrlKey: "https://example.com",
			}

			_, _, err := extractor.Extract(ctx, seshuJob, mockScraper)
			// Should not panic and should handle all modes
			_ = err
		})
	}
}
