package helpers

import (
	"os"
	"testing"
)

func init() {
    os.Setenv("GO_ENV", "test")
}


func TestFormatDate(t *testing.T) {
	tests := []struct {
			name string
			input string
			expected string
	}{
			{"Valid date", "2099-05-01T12:00:00Z", "May 1, 2099 (Fri)"},
			{"Invalid date", "invalid-date", "Invalid date"},
			{"Empty string", "", "Invalid date"},
	}

	for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
					result := FormatDate(tt.input)
					if result != tt.expected {
							t.Errorf("FormatDate(%q) = %q, want %q", tt.input, result, tt.expected)
					}
			})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
			name string
			input string
			expected string
	}{
			{"Valid time", "2023-05-01T14:30:00Z", "2:30pm"},
			{"Invalid time", "invalid-time", "Invalid time"},
			{"Empty string", "", "Invalid time"},
	}

	for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
					result := FormatTime(tt.input)
					if result != tt.expected {
							t.Errorf("FormatTime(%q) = %q, want %q", tt.input, result, tt.expected)
					}
			})
	}
}

func TestTruncateStringByBytes(t *testing.T) {
	tests := []struct {
			name string
			input1 string
			input2 int
			expected string
	}{
			{"Truncate exceeds by one", "123456789012345678901", 20, "12345678901234567890"},
	}

	for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
					result, _ := TruncateStringByBytes(tt.input1, tt.input2)
					if result != tt.expected {
							t.Errorf("TruncateStringByBytes(%q) = %q, want %q", tt.input1, result, tt.expected)
					}
			})
	}
}
