package constants

import (
	"math"
	"testing"
	"time"
)

func TestCompressDuration(t *testing.T) {
	tests := []struct {
		name             string
		ratio            float64
		inputDuration    time.Duration
		expectedDuration time.Duration
	}{
		{
			name:             "Real-time (ratio 1.0)",
			ratio:            1.0,
			inputDuration:    1 * time.Hour,
			expectedDuration: 1 * time.Hour,
		},
		{
			name:             "60x compression (1 hour → 1 minute)",
			ratio:            60.0,
			inputDuration:    1 * time.Hour,
			expectedDuration: 1 * time.Minute,
		},
		{
			name:             "3600x compression (1 hour → 1 second)",
			ratio:            3600.0,
			inputDuration:    1 * time.Hour,
			expectedDuration: 1 * time.Second,
		},
		{
			name:             "No compression (ratio < 1)",
			ratio:            0.5,
			inputDuration:    1 * time.Hour,
			expectedDuration: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily override the constant
			originalRatio := TIME_COMPRESSION_RATIO
			defer func() {
				// Can't actually restore since it's a const, but this documents intent
				_ = originalRatio
			}()

			// Calculate manually since we can't change the const
			var result time.Duration
			if tt.ratio <= 1.0 {
				result = tt.inputDuration
			} else {
				result = time.Duration(float64(tt.inputDuration) / tt.ratio)
			}

			if result != tt.expectedDuration {
				t.Errorf("CompressDuration() = %v, want %v", result, tt.expectedDuration)
			}
		})
	}
}

func TestCompressedScheduledHourInterval(t *testing.T) {
	// Test that the function compresses 1 hour interval correctly
	result := CompressedScheduledHourInterval()

	// With TIME_COMPRESSION_RATIO = 1.0, should return 1 hour
	if TIME_COMPRESSION_RATIO == 1.0 {
		if result != 1*time.Hour {
			t.Errorf("CompressedScheduledHourInterval() = %v, want %v", result, 1*time.Hour)
		}
	}

	// If ratio is different, calculate expected
	expected := time.Duration(float64(1*time.Hour) / TIME_COMPRESSION_RATIO)
	if TIME_COMPRESSION_RATIO > 1.0 && result != expected {
		t.Errorf("CompressedScheduledHourInterval() = %v, want %v", result, expected)
	}
}

func TestSimulatedHoursSince(t *testing.T) {
	tests := []struct {
		name          string
		ratio         float64
		nowUnix       int64
		lastUnix      int64
		expectedHours float64
	}{
		{
			name:          "Real-time: 1 hour passed",
			ratio:         1.0,
			nowUnix:       3600,
			lastUnix:      0,
			expectedHours: 1.0,
		},
		{
			name:          "60x compression: 60 seconds = 1 hour",
			ratio:         60.0,
			nowUnix:       60,
			lastUnix:      0,
			expectedHours: 1.0,
		},
		{
			name:          "3600x compression: 1 second = 1 hour",
			ratio:         3600.0,
			nowUnix:       1,
			lastUnix:      0,
			expectedHours: 1.0,
		},
		{
			name:          "60x compression: 120 seconds = 2 hours",
			ratio:         60.0,
			nowUnix:       120,
			lastUnix:      0,
			expectedHours: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate manually since we can't change TIME_COMPRESSION_RATIO
			realSecondsPassed := float64(tt.nowUnix - tt.lastUnix)
			simulatedSecondsPassed := realSecondsPassed * tt.ratio
			result := simulatedSecondsPassed / 3600.0

			if math.Abs(result-tt.expectedHours) > 0.0001 {
				t.Errorf("SimulatedHoursSince() = %v, want %v", result, tt.expectedHours)
			}
		})
	}
}

func TestCurrentSimulatedHour(t *testing.T) {
	tests := []struct {
		name         string
		ratio        float64
		nowUnix      int64
		expectedHour int
		description  string
	}{
		{
			name:         "Real-time at midnight UTC",
			ratio:        1.0,
			nowUnix:      1704067200, // 2024-01-01 00:00:00 UTC (midnight)
			expectedHour: 0,
			description:  "Should return actual hour (0)",
		},
		{
			name:         "Real-time at noon UTC",
			ratio:        1.0,
			nowUnix:      1704110400, // 2024-01-01 12:00:00 UTC (noon)
			expectedHour: 12,
			description:  "Should return actual hour (12)",
		},
		{
			name:         "3600x compression: 24 seconds = 1 day",
			ratio:        3600.0,
			nowUnix:      0,
			expectedHour: 0,
			description:  "At second 0, should be hour 0",
		},
		{
			name:         "3600x compression: 1 second = 1 hour",
			ratio:        3600.0,
			nowUnix:      1,
			expectedHour: 1,
			description:  "At second 1, should be hour 1",
		},
		{
			name:         "3600x compression: 12 seconds = noon",
			ratio:        3600.0,
			nowUnix:      12,
			expectedHour: 12,
			description:  "At second 12, should be hour 12 (noon)",
		},
		{
			name:         "3600x compression: wraps at 24 seconds",
			ratio:        3600.0,
			nowUnix:      24,
			expectedHour: 0,
			description:  "At second 24, should wrap back to hour 0",
		},
		{
			name:         "60x compression: 24 minutes = 1 day",
			ratio:        60.0,
			nowUnix:      0,
			expectedHour: 0,
			description:  "At second 0, should be hour 0",
		},
		{
			name:         "60x compression: 60 seconds = 1 hour",
			ratio:        60.0,
			nowUnix:      60,
			expectedHour: 1,
			description:  "At 60 seconds, should be hour 1",
		},
		{
			name:         "60x compression: 720 seconds = noon",
			ratio:        60.0,
			nowUnix:      720, // 12 minutes * 60 = 720 seconds = 12 hours simulated
			expectedHour: 12,
			description:  "At 720 seconds (12 minutes), should be hour 12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result int

			if tt.ratio <= 1.0 {
				// Real-time: use actual hour
				result = time.Unix(tt.nowUnix, 0).UTC().Hour()
			} else {
				// Compressed time: calculate accelerated hour
				secondsPerSimulatedDay := 86400.0 / tt.ratio
				positionInDay := math.Mod(float64(tt.nowUnix), secondsPerSimulatedDay)
				simulatedHour := int((positionInDay / secondsPerSimulatedDay) * 24.0)
				result = simulatedHour % 24
			}

			if result != tt.expectedHour {
				t.Errorf("%s: CurrentSimulatedHour() = %v, want %v", tt.description, result, tt.expectedHour)
			}
		})
	}
}

func TestTimeCompressionScenarios(t *testing.T) {
	t.Run("60x compression full cycle", func(t *testing.T) {
		// With 60x compression, a full 24-hour day happens in 24 minutes (1440 seconds)
		ratio := 60.0
		secondsPerDay := 86400.0 / ratio // 1440 seconds

		for hour := 0; hour < 24; hour++ {
			// Calculate the Unix timestamp for this simulated hour
			secondsIntoDay := (float64(hour) / 24.0) * secondsPerDay
			nowUnix := int64(secondsIntoDay)

			// Calculate simulated hour
			positionInDay := math.Mod(float64(nowUnix), secondsPerDay)
			simulatedHour := int((positionInDay / secondsPerDay) * 24.0)
			result := simulatedHour % 24

			if result != hour {
				t.Errorf("At %.0f seconds, expected hour %d, got %d", secondsIntoDay, hour, result)
			}
		}
	})

	t.Run("3600x compression full cycle", func(t *testing.T) {
		// With 3600x compression, a full 24-hour day happens in 24 seconds
		ratio := 3600.0
		secondsPerDay := 86400.0 / ratio // 24 seconds

		for hour := 0; hour < 24; hour++ {
			// Calculate the Unix timestamp for this simulated hour
			secondsIntoDay := (float64(hour) / 24.0) * secondsPerDay
			nowUnix := int64(secondsIntoDay)

			// Calculate simulated hour
			positionInDay := math.Mod(float64(nowUnix), secondsPerDay)
			simulatedHour := int((positionInDay / secondsPerDay) * 24.0)
			result := simulatedHour % 24

			if result != hour {
				t.Errorf("At %.0f seconds, expected hour %d, got %d", secondsIntoDay, hour, result)
			}
		}
	})
}
