package services

import (
	"testing"

	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetMockGeoService(t *testing.T) {
	mockService := getMockGeoService()

	// Check if the returned service is of the expected type
	if _, ok := mockService.(*test_helpers.MockGeoService); !ok {
		t.Errorf("expected type *test_helpers.MockGeoService, got %T", mockService)
	}
}

func TestGetMockSeshuService(t *testing.T) {
	mockService := getMockSeshuService()

	// Check if the returned service is of the expected type
	if _, ok := mockService.(*test_helpers.MockSeshuService); !ok {
		t.Errorf("expected type *test_helpers.MockSeshuService, got %T", mockService)
	}
}

