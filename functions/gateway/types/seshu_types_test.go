package types

import (
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
	"gorm.io/gorm"
)

func approxEqual(a, b float64) bool {
	const eps = 1e-9
	if a == b {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < eps
}

func TestSeshuSession_GetSetLocation(t *testing.T) {
	s := &SeshuSession{}
	if !approxEqual(0.0, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 0.0, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(0.0, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 0.0, got %v", s.GetLocationLongitude())
	}

	s.SetLocationLatitude(12.34)
	s.SetLocationLongitude(56.78)
	if !approxEqual(12.34, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 12.34, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(56.78, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 56.78, got %v", s.GetLocationLongitude())
	}
}

func TestSeshuSessionInsert_GetSetLocation(t *testing.T) {
	s := &SeshuSessionInsert{}
	if !approxEqual(0.0, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 0.0, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(0.0, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 0.0, got %v", s.GetLocationLongitude())
	}

	s.SetLocationLatitude(1.23)
	s.SetLocationLongitude(4.56)
	if !approxEqual(1.23, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 1.23, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(4.56, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 4.56, got %v", s.GetLocationLongitude())
	}
}

func TestSeshuSessionUpdate_GetSetLocation(t *testing.T) {
	s := &SeshuSessionUpdate{}
	// nil pointers should return 0
	if !approxEqual(0.0, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 0.0 for nil pointer, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(0.0, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 0.0 for nil pointer, got %v", s.GetLocationLongitude())
	}

	s.SetLocationLatitude(7.89)
	s.SetLocationLongitude(0.12)
	if !approxEqual(7.89, s.GetLocationLatitude()) {
		t.Fatalf("expected latitude 7.89, got %v", s.GetLocationLatitude())
	}
	if !approxEqual(0.12, s.GetLocationLongitude()) {
		t.Fatalf("expected longitude 0.12, got %v", s.GetLocationLongitude())
	}
}

func TestSeshuJob_GetSetLocation(t *testing.T) {
	j := &SeshuJob{}
	if !approxEqual(0.0, j.GetLocationLatitude()) {
		t.Fatalf("expected latitude 0.0, got %v", j.GetLocationLatitude())
	}
	if !approxEqual(0.0, j.GetLocationLongitude()) {
		t.Fatalf("expected longitude 0.0, got %v", j.GetLocationLongitude())
	}

	j.SetLocationLatitude(9.87)
	j.SetLocationLongitude(6.54)
	if !approxEqual(9.87, j.GetLocationLatitude()) {
		t.Fatalf("expected latitude 9.87, got %v", j.GetLocationLatitude())
	}
	if !approxEqual(6.54, j.GetLocationLongitude()) {
		t.Fatalf("expected longitude 6.54, got %v", j.GetLocationLongitude())
	}
}

func TestSeshuJob_BeforeCreateAndBeforeSave_Defaults(t *testing.T) {
	j := &SeshuJob{}
	// simulate GORM call, pass empty DB struct (safe for these hooks)
	err := j.BeforeCreate(&gorm.DB{})
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	// If LocationLatitude/Longitude were zero, they should be set to constants.INITIAL_EMPTY_LAT_LONG
	if !approxEqual(constants.INITIAL_EMPTY_LAT_LONG, j.LocationLatitude) {
		t.Fatalf("expected LocationLatitude to be set to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, j.LocationLatitude)
	}
	if !approxEqual(constants.INITIAL_EMPTY_LAT_LONG, j.LocationLongitude) {
		t.Fatalf("expected LocationLongitude to be set to %v, got %v", constants.INITIAL_EMPTY_LAT_LONG, j.LocationLongitude)
	}

	// Set to zero and call BeforeSave
	j.LocationLatitude = 0
	j.LocationLongitude = 0
	err = j.BeforeSave(&gorm.DB{})
	if err != nil {
		t.Fatalf("BeforeSave returned error: %v", err)
	}
	if !approxEqual(constants.INITIAL_EMPTY_LAT_LONG, j.LocationLatitude) {
		t.Fatalf("expected LocationLatitude to be set to %v after BeforeSave, got %v", constants.INITIAL_EMPTY_LAT_LONG, j.LocationLatitude)
	}
	if !approxEqual(constants.INITIAL_EMPTY_LAT_LONG, j.LocationLongitude) {
		t.Fatalf("expected LocationLongitude to be set to %v after BeforeSave, got %v", constants.INITIAL_EMPTY_LAT_LONG, j.LocationLongitude)
	}
}
