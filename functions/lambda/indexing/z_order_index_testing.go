package indexing

import (
    "testing"
    "time"
)

func TestCalculateZOrderIndex(t *testing.T) {
    // Test case 1: Normal index type
    now := time.Now()

    testCases := []struct {
        name string
        startTime time.Time
        lat float64
        lon float64
        indexType string
        expectErr bool
        expectZero bool
    }{
        {
			name:      "Normal index type",
			startTime: now.AddDate(0, 1, 0),
			lat:       40.7128,
			lon:       -74.0060,
			indexType: "normal",
			expectErr: false,
		},
		{
			name:      "Min index type",
			startTime: now.AddDate(0, 2, 0),
			lat:       40.7128,
			lon:       -74.0060,
			indexType: "min",
			expectErr: false,
		},
		{
			name:      "Max index type",
			startTime: now.AddDate(0, 3, 0),
			lat:       40.7128,
			lon:       -74.0060,
			indexType: "max",
			expectErr: false,
		},

    }

    for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CalculateZOrderIndex(tc.startTime, tc.lat, tc.lon, tc.indexType)
			if tc.expectErr {
				if err == nil {
					t.Error("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			}
		})
    }
} 

func TestMapFloatToSortableInt(t *testing.T) {
    testCases := []struct {
		name     string
		value    float64
		expected uint64
	}{
		{
			name:     "Positive value",
			value:    123.456,
			expected: uint64(4638507617480429056),
		},
		{
			name:     "Negative value",
			value:    -123.456,
			expected: uint64(13842132293241364480),
		},
		// Add more test cases as needed
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapFloatToSortableInt(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %d, but got %d", tc.expected, result)
			}
		})
	}
} 
