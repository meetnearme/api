package services

import (
	"os"
	"sync"
	"testing"

	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func resetSeshuService() {
	seshuService = nil
	seshuServiceOnce = sync.Once{}
}

func TestGetGeoServiceReturnsMockInTestEnv(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Setenv("GO_ENV", "test")
	ResetGeoService()

	svc := GetGeoService()
	if _, ok := svc.(*test_helpers.MockGeoService); !ok {
		t.Fatalf("expected mock geo service in test env, got %T", svc)
	}
}

func TestGetGeoServiceSingleton(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Setenv("GO_ENV", "test")
	ResetGeoService()

	first := GetGeoService()
	second := GetGeoService()
	if first != second {
		t.Fatalf("expected geo service to be singleton")
	}
}

func TestResetGeoServiceAllowsReinitialization(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Setenv("GO_ENV", "test")
	ResetGeoService()

	first := GetGeoService()
	ResetGeoService()
	second := GetGeoService()

	if first == second {
		t.Fatalf("expected new geo service instance after reset")
	}
}

func TestGetGeoServiceReturnsRealOutsideTest(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Unsetenv("GO_ENV")
	ResetGeoService()

	svc := GetGeoService()
	if _, ok := svc.(*RealGeoService); !ok {
		t.Fatalf("expected real geo service outside test env, got %T", svc)
	}
}

func TestGetSeshuServiceReturnsMockInTestEnv(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Setenv("GO_ENV", "test")
	resetSeshuService()

	svc := GetSeshuService()
	if _, ok := svc.(*test_helpers.MockSeshuService); !ok {
		t.Fatalf("expected mock seshu service in test env, got %T", svc)
	}
}

func TestGetSeshuServiceReturnsRealOutsideTest(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Unsetenv("GO_ENV")
	resetSeshuService()

	svc := GetSeshuService()
	if _, ok := svc.(*RealSeshuService); !ok {
		t.Fatalf("expected real seshu service outside test env, got %T", svc)
	}
}

func TestGetSeshuServiceSingletonAndReset(t *testing.T) {
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)
	os.Setenv("GO_ENV", "test")
	resetSeshuService()

	first := GetSeshuService()
	second := GetSeshuService()
	if first != second {
		t.Fatalf("expected seshu service to be singleton")
	}

	resetSeshuService()
	third := GetSeshuService()
	if _, ok := third.(*test_helpers.MockSeshuService); !ok {
		t.Fatalf("expected mock seshu service after reset, got %T", third)
	}
}
