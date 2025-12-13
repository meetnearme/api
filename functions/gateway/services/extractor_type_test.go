package services

import (
	"testing"

	"github.com/meetnearme/api/functions/gateway/types"
)

// Mock ScrapingService for testing
type MockScrapingService struct {
	GetHTMLFromURLFunc            func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error)
	GetHTMLFromURLWithRetriesFunc func(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error)
}

func (m *MockScrapingService) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	if m.GetHTMLFromURLFunc != nil {
		return m.GetHTMLFromURLFunc(seshuJob, waitMs, jsRender, waitFor)
	}
	return "", nil
}

func (m *MockScrapingService) GetHTMLFromURLWithRetries(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
	if m.GetHTMLFromURLWithRetriesFunc != nil {
		return m.GetHTMLFromURLWithRetriesFunc(seshuJob, waitMs, jsRender, waitFor, maxRetries, validationFunc)
	}
	return "", nil
}

// Test EventExtractor interface is properly defined
func TestEventExtractorInterface(t *testing.T) {
	var _ EventExtractor = (*FacebookExtractor)(nil)
	var _ EventExtractor = (*GenericExtractor)(nil)
}

// Test that FacebookExtractor implements EventExtractor
func TestFacebookExtractorImplementsInterface(t *testing.T) {
	extractor := &FacebookExtractor{}
	if extractor == nil {
		t.Fatal("FacebookExtractor should not be nil")
	}

	// Check that methods exist
	if canHandle := extractor.CanHandle; canHandle == nil {
		t.Fatal("FacebookExtractor should have CanHandle method")
	}

	if extract := extractor.Extract; extract == nil {
		t.Fatal("FacebookExtractor should have Extract method")
	}
}

// Test that GenericExtractor implements EventExtractor
func TestGenericExtractorImplementsInterface(t *testing.T) {
	extractor := &GenericExtractor{}
	if extractor == nil {
		t.Fatal("GenericExtractor should not be nil")
	}

	// Check that methods exist
	if canHandle := extractor.CanHandle; canHandle == nil {
		t.Fatal("GenericExtractor should have CanHandle method")
	}

	if extract := extractor.Extract; extract == nil {
		t.Fatal("GenericExtractor should have Extract method")
	}
}
