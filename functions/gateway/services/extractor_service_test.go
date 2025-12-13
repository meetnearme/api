package services

import (
	"testing"
)

func TestNewExtractorRegistry(t *testing.T) {
	registry := NewExtractorRegistry()

	if registry == nil {
		t.Fatal("NewExtractorRegistry should not return nil")
	}

	if len(registry.extractors) == 0 {
		t.Fatal("ExtractorRegistry should have extractors initialized")
	}
}

func TestExtractorRegistryHasFacebookExtractor(t *testing.T) {
	registry := NewExtractorRegistry()

	foundFacebook := false
	for _, extractor := range registry.extractors {
		if _, ok := extractor.(*FacebookExtractor); ok {
			foundFacebook = true
			break
		}
	}

	if !foundFacebook {
		t.Fatal("ExtractorRegistry should contain FacebookExtractor")
	}
}

func TestGetExtractorReturnsCorrectExtractor(t *testing.T) {
	registry := NewExtractorRegistry()

	tests := []struct {
		name         string
		url          string
		expectedType string
	}{
		{
			name:         "Facebook URL returns FacebookExtractor",
			url:          "https://www.facebook.com/events/123",
			expectedType: "*services.FacebookExtractor",
		},
		{
			name:         "Unknown URL returns GenericExtractor",
			url:          "https://example.com/events",
			expectedType: "*services.GenericExtractor",
		},
		{
			name:         "Empty URL returns GenericExtractor",
			url:          "",
			expectedType: "*services.GenericExtractor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := registry.GetExtractor(tt.url)
			if extractor == nil {
				t.Fatal("GetExtractor should not return nil")
			}

			// Verify the type is correct
			switch tt.expectedType {
			case "*services.FacebookExtractor":
				if _, ok := extractor.(*FacebookExtractor); !ok {
					t.Errorf("Expected FacebookExtractor, got %T", extractor)
				}
			case "*services.GenericExtractor":
				if _, ok := extractor.(*GenericExtractor); !ok {
					t.Errorf("Expected GenericExtractor, got %T", extractor)
				}
			}
		})
	}
}

func TestGetExtractorFallsbackToGeneric(t *testing.T) {
	registry := NewExtractorRegistry()

	unknownURLs := []string{
		"https://example.com/events",
	}

	for _, url := range unknownURLs {
		extractor := registry.GetExtractor(url)
		if _, ok := extractor.(*GenericExtractor); !ok {
			t.Errorf("Expected GenericExtractor for %s, got %T", url, extractor)
		}
	}
}
