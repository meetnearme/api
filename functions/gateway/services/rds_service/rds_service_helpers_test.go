package rds_service

import (
	"testing"
	"time"

	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

func TestGetString(t *testing.T) {
	record := map[string]interface{}{
		"key": "value",
	}
	result := getString(record, "key")
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}

	result = getString(record, "missing")
	if result != "" {
		t.Errorf("expected '', got '%s'", result)
	}
}

func TestGetOptionalString(t *testing.T) {
	record := map[string]interface{}{
		"key": "value",
	}
	result := getOptionalString(record, "key")
	if *result != "value" {
		t.Errorf("expected 'value', got '%s'", *result)
	}

	result = getOptionalString(record, "missing")
	if result != nil {
		t.Errorf("expected nil, got '%v'", result)
	}
}

func TestGetTime(t *testing.T) {
	layout := "2006-01-02 15:04:05.999999"
	now := time.Now().Format(layout)
	record := map[string]interface{}{
		"timeKey": now,
	}
	result := getTime(record, "timeKey")
	expected, _ := time.Parse(layout, now)
	if !result.Equal(expected) {
		t.Errorf("expected '%v', got '%v'", expected, result)
	}

	result = getTime(record, "missing")
	if !result.IsZero() {
		t.Errorf("expected zero time, got '%v'", result)
	}
}

func TestGetFloat64(t *testing.T) {
	record := map[string]interface{}{
		"floatKey1": 1.23,
		"floatKey2": "4.56",
	}
	result := getFloat64(record, "floatKey1")
	if result != 1.23 {
		t.Errorf("expected 1.23, got %f", result)
	}

	result = getFloat64(record, "floatKey2")
	if result != 4.56 {
		t.Errorf("expected 4.56, got %f", result)
	}

	result = getFloat64(record, "missing")
	if result != 0.0 {
		t.Errorf("expected 0.0, got %f", result)
	}
}

func TestGetInt64(t *testing.T) {
	record := map[string]interface{}{
		"intKey1": int64(123),
		"intKey2": "456",
	}
	result := getInt64(record, "intKey1")
	if result != 123 {
		t.Errorf("expected 123, got %d", result)
	}

	result = getInt64(record, "intKey2")
	if result != 456 {
		t.Errorf("expected 456, got %d", result)
	}

	result = getInt64(record, "missing")
	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}

func TestBuildArrayString(t *testing.T) {
    input := []string{"a", "b", "c"}
    result := buildArrayString(input)
    expected := []*rds_types.FieldMemberStringValue{
        {Value: "a"},
        {Value: "b"},
        {Value: "c"},
    }

    if len(result) != len(expected) {
        t.Errorf("expected length %d, got %d", len(expected), len(result))
    }

    for i, v := range result {
        if v.Value != expected[i].Value {
            t.Errorf("expected '%s', got '%s'", expected[i].Value, v.Value)
        }
    }
}


func TestGetStringSlice(t *testing.T) {
	record := map[string]interface{}{
		"commaKey": "a,b,c",
	}
	result := getStringSlice(record, "commaKey")
	expected := []string{"a", "b", "c"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("expected '%s', got '%s'", expected[i], v)
		}
	}

	result = getStringSlice(record, "missing")
	if result != nil {
		t.Errorf("expected nil, got '%v'", result)
	}
}

func TestSliceToCommaSeparatedString(t *testing.T) {
	input := []string{"a", "b", "c"}
	result := sliceToCommaSeparatedString(input)
	expected := "a,b,c"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestCommaSeparatedStringToSlice(t *testing.T) {
	input := "a,b,c"
	result := commaSeparatedStringToSlice(input)
	expected := []string{"a", "b", "c"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("expected '%s', got '%s'", expected[i], v)
		}
	}

	result = commaSeparatedStringToSlice("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got '%v'", result)
	}
}

