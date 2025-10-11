package types

import (
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
)

func TestSeshuSessionApplyDefaults(t *testing.T) {
	session := &SeshuSession{}
	session.ApplyDefaults()

	if session.LocationLatitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected latitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, session.LocationLatitude)
	}

	if session.LocationLongitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected longitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, session.LocationLongitude)
	}

	if session.EventCandidates == nil {
		t.Fatalf("expected event candidates slice to be initialized, got nil")
	}

	if session.EventValidations == nil {
		t.Fatalf("expected event validations slice to be initialized, got nil")
	}
}

func TestSeshuSessionUpdateApplyDefaults(t *testing.T) {
	lat := 0.0
	lon := 0.0
	update := &SeshuSessionUpdate{
		LocationLatitude:  &lat,
		LocationLongitude: &lon,
	}

	update.ApplyDefaults()

	if update.LocationLatitude == nil || *update.LocationLatitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected update latitude pointer to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, update.LocationLatitude)
	}

	if update.LocationLongitude == nil || *update.LocationLongitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected update longitude pointer to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, update.LocationLongitude)
	}
}

func TestSeshuSessionInsertApplyDefaults(t *testing.T) {
	insert := &SeshuSessionInsert{}
	insert.ApplyDefaults()

	if insert.LocationLatitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected insert latitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, insert.LocationLatitude)
	}

	if insert.LocationLongitude != constants.INITIAL_EMPTY_LAT_LONG {
		t.Fatalf("expected insert longitude to default to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, insert.LocationLongitude)
	}

	if insert.EventCandidates == nil {
		t.Fatalf("expected event candidates slice to be initialized, got nil")
	}

	if insert.EventValidations == nil {
		t.Fatalf("expected event validations slice to be initialized, got nil")
	}
}
