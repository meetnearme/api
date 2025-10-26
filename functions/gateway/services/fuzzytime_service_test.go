package services

import (
	"os"
	"testing"
	"time"
)

// parseFlexible tries multiple layouts including RFC3339 and no-zone formats
func parseFlexible(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}
	var err error
	var t time.Time
	for _, l := range layouts {
		t, err = time.Parse(l, value)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// withFrozenTime sets GO_ENV=test for frozen time testing
func withFrozenTime(t *testing.T, testFunc func()) {
	// Set GO_ENV=test to enable frozen time
	originalEnv := os.Getenv("GO_ENV")
	os.Setenv("GO_ENV", "test")

	// Ensure cleanup
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("GO_ENV")
		} else {
			os.Setenv("GO_ENV", originalEnv)
		}
	}()

	// Run the test
	testFunc()
}

func TestParseMaybeMultiDayEvent(t *testing.T) {
	withFrozenTime(t, func() {
		tests := []struct {
			name     string
			input    string
			expected string
			hasError bool
		}{
			// Standard formats that should work
			{
				name:     "rfc3339_format",
				input:    "2024-07-25T15:00:00Z",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "iso8601_format",
				input:    "2024-07-25T15:00:00+00:00",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "simple_date_format",
				input:    "2024-07-25 15:00:00",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "unix_timestamp",
				input:    "1721923200",
				expected: "", // dateparse handles this but with timezone offset, not UTC
				hasError: false,
			},

			// Additional dateparse examples that should work
			{
				name:     "natural_language_with_time",
				input:    "May 8, 2009 5:57:51 PM",
				expected: "2009-05-08T17:57:51Z",
				hasError: false,
			},
			{
				name:     "abbreviated_month_date",
				input:    "oct 7, 1970",
				expected: "1970-10-07T00:00:00Z",
				hasError: false,
			},
			{
				name:     "abbreviated_month_2digit_year",
				input:    "oct. 7, 70",
				expected: "1970-10-07T00:00:00Z",
				hasError: false, // 2-digit years default to 19xx
			},
			{
				name:     "go_time_format",
				input:    "Mon Jan  2 15:04:05 2006",
				expected: "2006-01-02T15:04:05Z",
				hasError: false,
			},
			{
				name:     "rfc822_format",
				input:    "Mon, 02 Jan 2006 15:04:05 MST",
				expected: "2006-01-02T15:04:05Z",
				hasError: false,
			},
			{
				name:     "us_date_with_time",
				input:    "4/8/2014 22:05",
				expected: "2014-04-08T22:05:00Z",
				hasError: false,
			},
			{
				name:     "european_date_with_time",
				input:    "2014/4/8 22:05",
				expected: "2014-04-08T22:05:00Z",
				hasError: false,
			},
			{
				name:     "colon_separated_date",
				input:    "2014:4:8 22:05",
				expected: "2014-04-08T22:05:00Z",
				hasError: false,
			},
			{
				name:     "dot_separated_date",
				input:    "3.31.2014",
				expected: "2014-03-31T00:00:00Z",
				hasError: false,
			},
			{
				name:     "compact_date_time",
				input:    "20140722105203",
				expected: "2014-07-22T10:52:03Z",
				hasError: false,
			},
			{
				name:     "mysql_log_format",
				input:    "171113 14:14:20",
				expected: "2017-11-13T14:14:20Z",
				hasError: false,
			},

			// Common date formats
			{
				name:     "us_date_format",
				input:    "07/25/2024 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "european_date_format",
				input:    "25/07/2024 15:00",
				expected: "",
				hasError: true, // dateparse doesn't handle DD/MM/YYYY format
			},
			{
				name:     "natural_language_date",
				input:    "July 25, 2024 at 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "natural_language_date_range",
				input:    "July 25 3:00 PM - July 26 4:00 PM, 2024",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "abbreviated_month",
				input:    "Jul 25, 2024 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "day_of_week_format",
				input:    "Thursday, July 25, 2024 at 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false, // New package handles day of week prefix correctly
			},

			// Time-only formats (fuzzytime fallback handles these with incomplete data)
			{
				name:     "time_only_12hour",
				input:    "3:00 PM",
				expected: "",   // fuzzytime produces incomplete time-only result
				hasError: true, // fuzzytime now returns error for time-only ambiguous inputs
			},
			{
				name:     "time_only_24hour",
				input:    "15:00",
				expected: "",   // fuzzytime produces incomplete time-only result
				hasError: true, // fuzzytime now returns error for time-only ambiguous inputs
			},

			// Date ranges (should extract start date)
			{
				name:     "date_range_with_dash",
				input:    "Fri, Jul 25 - Jul 26, 2024",
				expected: "2024-07-25T00:00:00Z",
				hasError: false, // New package handles this correctly
			},
			{
				name:     "date_range_with_en_dash",
				input:    "July 25 – July 26, 2024",
				expected: "2024-07-25T00:00:00Z",
				hasError: false, // Should work with improved cleaning
			},
			{
				name:     "time_range_event",
				input:    "Saturday, July 26, 2025 at 6:30PM – 9:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false, // Should work with improved cleaning
			},
			{
				name:     "facebook_format_with_from",
				input:    "Saturday 26 July 2025 from 18:30-21:30",
				expected: "2025-07-26T18:30:00Z",
				hasError: false, // New package handles this correctly after cleaning
			},

			// Edge cases and error conditions
			{
				name:     "empty_string",
				input:    "",
				expected: "",
				hasError: true,
			},
			{
				name:     "invalid_date",
				input:    "not a date",
				expected: "",
				hasError: true,
			},
			{
				name:     "partial_date",
				input:    "July 25",
				expected: "2026-07-25T00:00:00Z",
				hasError: false, // Our "next future" logic adds appropriate year
			},
			{
				name:     "year_only",
				input:    "2024",
				expected: "2024-01-01T00:00:00Z",
				hasError: false,
			},
			{
				name:     "month_year_only",
				input:    "July 2024",
				expected: "",
				hasError: true, // dateparse doesn't handle month-year format
			},
			{
				name:     "lowercase_month_names",
				input:    "jul 25, 2024 at 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false, // Should work with uppercase normalization
			},
			{
				name:     "uppercase_month_names",
				input:    "JULY 25, 2024 at 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false, // Should work with uppercase normalization
			},
			{
				name:     "mixed_case_month_names",
				input:    "JuLy 25, 2024 at 3:00 PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false, // Should work with uppercase normalization
			},

			// Timezone variations
			{
				name:     "with_timezone_abbrev",
				input:    "2024-07-25 3:00 PM EST",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "with_timezone_offset",
				input:    "2024-07-25 3:00 PM -0500",
				expected: "2024-07-25T15:00:00Z", // Timezone ignored: literal time as UTC
				hasError: false,
			},
			{
				name:     "with_iana_timezone",
				input:    "2024-07-25 3:00 PM America/New_York",
				expected: "2024-07-25T15:00:00Z", // fuzzytime fallback handles IANA timezones
				hasError: false,
			},

			// Edge cases discovered during debugging
			{
				name:     "day_of_week_without_comma",
				input:    "Saturday July 26, 2025 at 6:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "day_of_week_without_at",
				input:    "Saturday, July 26, 2025 6:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "day_of_week_without_comma_or_at",
				input:    "Saturday July 26, 2025 6:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "time_range_with_pipe",
				input:    "Saturday, July 26, 2025 at 6:30PM | 9:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "time_range_with_regular_dash",
				input:    "Saturday, July 26, 2025 at 6:30PM - 9:30PM",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "date_range_with_pipe",
				input:    "July 25 | July 26, 2024",
				expected: "2024-07-25T00:00:00Z",
				hasError: false,
			},
			{
				name:     "facebook_from_keyword",
				input:    "Saturday 26 July 2025 from 18:30-21:30",
				expected: "2025-07-26T18:30:00Z",
				hasError: false,
			},
			{
				name:     "24_hour_time_format",
				input:    "2024-07-25 18:30",
				expected: "2024-07-25T18:30:00Z",
				hasError: false,
			},
			{
				name:     "24_hour_time_with_seconds",
				input:    "2024-07-25 18:30:45",
				expected: "2024-07-25T18:30:45Z",
				hasError: false,
			},
			{
				name:     "ambiguous_date_past",
				input:    "Jan 15", // January 15 - should be next year if past
				expected: "2026-01-15T00:00:00Z",
				hasError: false,
			},
			{
				name:     "ambiguous_date_future",
				input:    "Dec 12", // December 12 - should be current year if not past
				expected: "2025-12-12T00:00:00Z",
				hasError: false,
			},
			{
				name:     "ambiguous_date_with_day_of_week",
				input:    "Fri, Jul 25", // Should use next future logic
				expected: "2026-07-25T00:00:00Z",
				hasError: false,
			},
			{
				name:     "ambiguous_date_uppercase_month",
				input:    "JULY 25",
				expected: "2025-07-25T00:00:00Z",
				hasError: false, // Should work with uppercase normalization
			},
			{
				name:     "non_breaking_spaces",
				input:    "July\u00A025,\u00A02024\u00A0at\u00A03:00\u00A0PM",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "time_only_ambiguous",
				input:    "3:00 PM",
				expected: "",   // fuzzytime produces incomplete time-only result
				hasError: true, // fuzzytime now returns error for ambiguous time-only inputs
			},
			{
				name:     "time_only_24hour_ambiguous",
				input:    "15:00",
				expected: "",   // fuzzytime produces incomplete time-only result
				hasError: true, // fuzzytime now returns error for ambiguous time-only inputs
			},
			// Natural language variations with yearless parsing
			{
				name:     "august_date_range",
				input:    "August 15 2:30 PM - August 16 4:00 PM, 2024",
				expected: "2024-08-15T14:30:00Z",
				hasError: false,
			},
			{
				name:     "september_date_range",
				input:    "September 10 9:00 AM - September 11 5:00 PM, 2024",
				expected: "2024-09-10T09:00:00Z",
				hasError: false,
			},
			{
				name:     "october_date_range",
				input:    "October 31 7:00 PM - November 1 12:00 AM, 2024",
				expected: "2024-10-31T19:00:00Z",
				hasError: false,
			},
			{
				name:     "december_date_range",
				input:    "December 25 10:00 AM - December 26 2:00 PM, 2024",
				expected: "2024-12-25T10:00:00Z",
				hasError: false,
			},
			{
				name:     "january_date_range",
				input:    "January 1 12:00 AM - January 2 11:59 PM, 2025",
				expected: "2025-01-01T00:00:00Z",
				hasError: false,
			},
			{
				name:     "february_date_range",
				input:    "February 14 6:00 PM - February 15 8:00 PM, 2025",
				expected: "2025-02-14T18:00:00Z",
				hasError: false,
			},
			{
				name:     "march_date_range",
				input:    "March 17 11:00 AM - March 18 1:00 PM, 2025",
				expected: "2025-03-17T11:00:00Z",
				hasError: false,
			},
			{
				name:     "april_date_range",
				input:    "April 1 3:30 PM - April 2 5:30 PM, 2025",
				expected: "2025-04-01T15:30:00Z",
				hasError: false,
			},
			{
				name:     "may_date_range",
				input:    "May 5 4:15 PM - May 6 6:15 PM, 2025",
				expected: "2025-05-05T16:15:00Z",
				hasError: false,
			},
			{
				name:     "june_date_range",
				input:    "June 21 8:00 AM - June 22 10:00 AM, 2025",
				expected: "2025-06-21T08:00:00Z",
				hasError: false,
			},
			// Edge cases for yearless parsing
			{
				name:     "date_range_no_time",
				input:    "July 25 - July 26, 2024",
				expected: "2024-07-25T00:00:00Z",
				hasError: false,
			},
			{
				name:     "date_range_with_at",
				input:    "July 25 at 3:00 PM - July 26 at 4:00 PM, 2024",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "date_range_with_comma",
				input:    "July 25, 3:00 PM - July 26, 4:00 PM, 2024",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "date_range_24hour_time",
				input:    "July 25 15:00 - July 26 16:00, 2024",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "date_range_24hour_with_seconds",
				input:    "July 25 15:30:45 - July 26 16:30:45, 2024",
				expected: "2024-07-25T15:30:45Z",
				hasError: false,
			},
			// Complex natural language variations
			{
				name:     "complex_natural_range_1",
				input:    "Saturday, July 25 3:00 PM - Sunday, July 26 4:00 PM, 2024",
				expected: "2024-07-25T15:00:00Z",
				hasError: false,
			},
			{
				name:     "complex_natural_range_2",
				input:    "Friday, August 15 at 2:30 PM - Saturday, August 16 at 4:00 PM, 2024",
				expected: "2024-08-15T14:30:00Z",
				hasError: false,
			},
			{
				name:     "complex_natural_range_3",
				input:    "Monday, September 10, 9:00 AM - Tuesday, September 11, 5:00 PM, 2024",
				expected: "2024-09-10T09:00:00Z",
				hasError: false,
			},
			// Facebook formats
			{
				name:     "facebook_format_with_year_future",
				input:    "Sat, 3 Oct at 09:00 CDT, 2099",
				expected: "2099-10-03T14:00:00Z", // fuzzytime now normalizes timezone to UTC (09:00 CDT -> 14:00Z)
				hasError: false,
			},
			{
				name:     "facebook_format_without_year",
				input:    "Fri, 10 Oct at 09:00 CDT",
				expected: "2025-10-10T14:00:00Z", // fuzzytime now normalizes timezone to UTC (09:00 CDT -> 14:00Z)
				hasError: false,
			},
			{
				name:     "facebook_format_24hr_time",
				input:    "Fri, 12 Sep at 17:00 CDT",
				expected: "2025-09-12T22:00:00Z", // 17:00 CDT -> 22:00Z after normalization
				hasError: false,
			},
			{
				name:     "facebook_format_24hr_time_with_range",
				input:    "Thursday 12 September 2024 from 17:00-21:00 CDT",
				expected: "2024-09-12T17:00:00Z", // fuzzytime + addNextFutureYear + timezone ignored (literal time as UTC)
				hasError: false,
			},
			{
				name:     "facebook_format_with_range",
				input:    "Saturday, September 14, 2024 at 2:00 PM – 4:00 PM CDT",
				expected: "2024-09-14T14:00:00Z", // fuzzytime + addNextFutureYear + timezone ignored (literal time as UTC)
				hasError: false,
			},
			{
				name:     "facebook_format_with_and_more",
				input:    "Fri, Sep 12 5:00PM CDT and 15 more",
				expected: "2025-09-12T22:00:00Z", // normalized to UTC (17:00 CDT -> 22:00Z)
				hasError: false,
			},
			// Production failure pattern - time range with timezone
			{
				name:     "production_time_range_with_timezone_future",
				input:    "Friday, September 12, 2099 at 5:00 PM – 9:00 PM CDT",
				expected: "2099-09-12T17:00:00Z", // fuzzytime handles time range + timezone, returns without seconds
				hasError: false,
			},
			{
				name:     "production_time_range_with_timezone_past",
				input:    "Friday, September 12, 2024 at 5:00 PM – 9:00 PM CDT",
				expected: "2024-09-12T17:00:00Z", // fuzzytime handles time range + timezone, returns without seconds
				hasError: false,
			},
			// Test cases to ensure both dateparse and fuzzytime paths ignore timezones
			{
				name:     "dateparse_path_with_timezone",
				input:    "2024-07-25T15:00:00-05:00", // This should use dateparse path
				expected: "2024-07-25T15:00:00Z",      // dateparse + timezone ignored (literal time as UTC)
				hasError: false,
			},
			{
				name:     "dateparse_path_with_timezone_est",
				input:    "July 25, 2024 at 3:00 PM EST", // This should use dateparse path
				expected: "2024-07-25T15:00:00Z",         // dateparse + timezone ignored (literal time as UTC)
				hasError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := ParseMaybeMultiDayEvent(tt.input)

				if tt.hasError {
					if err == nil {
						t.Errorf("expected error but got none")
					}
					return
				}

				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				// Handle special cases where we can't predict exact output
				if tt.expected == "" {
					// For time-only formats, just verify we got a valid result
					if result == "" {
						t.Errorf("expected non-empty result")
						return
					}
				} else {
					// For cases with expected output, parse both expected and result with flexible layouts
					expT, err := parseFlexible(tt.expected)
					if err != nil {
						t.Errorf("failed to parse expected time %q: %v", tt.expected, err)
						return
					}
					resT, err := parseFlexible(result)
					if err != nil {
						t.Errorf("failed to parse result time %q: %v", result, err)
						return
					}
					if !expT.Equal(resT) {
						t.Errorf("expected %q (%v), got %q (%v)", tt.expected, expT, result, resT)
						return
					}
				}

				// Verify result parses to a valid time (accept multiple formats)
				parsedTime, err := parseFlexible(result)
				if err != nil {
					t.Errorf("result is not a valid time: %v", err)
					return
				}

				// Ensure it's normalized or convertible to UTC (we treat times as UTC)
				if parsedTime.Location() != time.UTC {
					t.Errorf("expected parsed time in UTC location, got %v", parsedTime.Location())
				}

				// Log the result for debugging
				t.Logf("Input: %q -> Output: %q", tt.input, result)
			})
		}
	}) // Close withFrozenTime
}

func TestCleanDateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "date_range_with_dash",
			input:    "Fri, Jul 25 - Jul 26",
			expected: "Fri, Jul 25", // Year added by "next future" logic
		},
		{
			name:     "time_range_with_en_dash",
			input:    "Saturday, July 26, 2025 at 6:30PM – 9:30PM",
			expected: "Saturday, July 26, 2025 6:30PM", // "at" removed for dateparse compatibility
		},
		{
			name:     "time_range_with_hyphen",
			input:    "Saturday 26 July 2025 from 18:30-21:30",
			expected: "Saturday 26 July 2025 18:30", // "from" removed by character cleaning
		},
		{
			name:     "no_range",
			input:    "Friday, July 25, 2025 at 3:00 PM",
			expected: "Friday, July 25, 2025 3:00 PM", // "at" removed for dateparse compatibility
		},
		{
			name:     "normalize_spaces",
			input:    "Friday, July 25, 2025 at 3:00 PM",
			expected: "Friday, July 25, 2025 3:00 PM", // "at" removed for dateparse compatibility
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanDateString(tt.input)
			if result != tt.expected {
				t.Errorf("cleanDateString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeDateRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "date_range_with_dash",
			input:    "Fri, Jul 25 - Jul 26",
			expected: true,
		},
		{
			name:     "date_range_with_en_dash",
			input:    "Sep 12 – Oct 4",
			expected: true,
		},
		{
			name:     "single_date",
			input:    "Friday, July 25, 2025 at 3:00 PM",
			expected: false,
		},
		{
			name:     "time_range",
			input:    "6:30PM – 9:30PM",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeDateRange(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeDateRange(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLooksLikeTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "time_range_with_en_dash",
			input:    "6:30PM – 9:30PM",
			expected: true,
		},
		{
			name:     "time_range_with_hyphen",
			input:    "18:30-21:30",
			expected: true,
		},
		{
			name:     "single_time",
			input:    "6:30PM",
			expected: false,
		},
		{
			name:     "date_range",
			input:    "Fri, Jul 25 - Jul 26",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeTimeRange(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeTimeRange(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
