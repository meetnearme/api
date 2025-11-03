package services

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcampbell/fuzzytime"
	"github.com/itlightning/dateparse"
)

var reYear = regexp.MustCompile(`\b(?:19|20)\d{2}\b|\b2100\b`)

// getCurrentTime returns the current time, or a frozen time if GO_ENV=test
func getCurrentTime() time.Time {
	if os.Getenv("GO_ENV") == "test" {
		// Freeze time to September 11, 2025 for consistent test results
		return time.Date(2025, 9, 11, 15, 0, 0, 0, time.UTC)
	}
	return time.Now()
}

// ParseMaybeMultiDayEvent handles multi-day spanning events by extracting start time
// Note: Offset resolution is handled separately via geolocation + tzf library
// to avoid conflicts between timezone abbreviations and location-based timezone detection
func ParseMaybeMultiDayEvent(input string) (string, error) {
	cleanedDateStr := cleanDateString(input)

	// Try dateparse first
	startDt, err := dateparse.ParseAny(cleanedDateStr)
	if err == nil {
		year, month, day := startDt.Date()
		if year == 0 {
			// dateparse returned year 0 → fallback to fuzzytime
			patchedDateStr := addNextFutureYear(cleanedDateStr)
			dt, _, err := fuzzytime.Extract(patchedDateStr)
			if err != nil || dt.Empty() {
				return "", fmt.Errorf("dateparse returned year 0 and fuzzytime failed: %s", input)
			}

			isoFormat := dt.ISOFormat()

			// Check if fuzzytime returned year 0 and handle it directly
			if strings.Contains(isoFormat, "0000") {
				now := getCurrentTime()
				currentYear := now.Year()

				// Parse without timezone
				formats := []string{"2006-01-02T15:04:05", "2006-01-02T15:04"}
				var parsedTime time.Time
				for _, f := range formats {
					parsedTime, err = time.Parse(f, isoFormat)
					if err == nil {
						break
					}
				}
				if err != nil {
					return "", fmt.Errorf("failed to parse fuzzytime result with year 0: %s", isoFormat)
				}

				testDate := time.Date(currentYear, parsedTime.Month(), parsedTime.Day(),
					parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, time.UTC)
				if testDate.Before(now) {
					testDate = time.Date(currentYear+1, parsedTime.Month(), parsedTime.Day(),
						parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, time.UTC)
				}

				return testDate.Format("2006-01-02T15:04:05"), nil
			}
		} else {
			// dateparse succeeded with proper year, ignore timezone
			hour, min, sec := startDt.Clock()
			utcTime := time.Date(year, month, day, hour, min, sec, 0, time.UTC)
			return utcTime.Format("2006-01-02T15:04:05"), nil
		}
	}

	// Patch year for fuzzytime
	patchedDateStr := addNextFutureYear(cleanedDateStr)

	// Try fuzzytime with the patched string
	dt, _, err := fuzzytime.Extract(patchedDateStr)
	if err != nil || dt.Empty() {
		return "", fmt.Errorf("both dateparse and fuzzytime failed to parse: %s", input)
	}

	isoFormat := dt.ISOFormat()

	if strings.Contains(isoFormat, "0000") {
		now := getCurrentTime()
		currentYear := now.Year()

		formats := []string{"2006-01-02T15:04:05", "2006-01-02T15:04"}
		var parsedTime time.Time
		for _, f := range formats {
			parsedTime, err = time.Parse(f, isoFormat)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "", fmt.Errorf("failed to parse fuzzytime result with year 0: %s", isoFormat)
		}

		testDate := time.Date(currentYear, parsedTime.Month(), parsedTime.Day(),
			parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, time.UTC)
		if testDate.Before(now) {
			testDate = time.Date(currentYear+1, parsedTime.Month(), parsedTime.Day(),
				parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), 0, time.UTC)
		}

		return testDate.Format("2006-01-02T15:04:05"), nil
	}

	// Convert fuzzytime result. First try RFC3339-like layouts (with timezone offsets)
	// because fuzzytime may include a timezone offset like "-05:00" or a trailing Z.
	rfcLayouts := []string{
		time.RFC3339,             // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04Z07:00", // "2006-01-02T15:04-05:00" (no seconds)
		"2025-12-17T17:00-06:00",
	}
	var parsedTime time.Time
	for _, l := range rfcLayouts {
		parsedTime, err = time.Parse(l, isoFormat)
		if err == nil {
			return parsedTime.Format("2006-01-02T15:04:05"), nil
		}
	}

	// Fallback: try date/time formats without timezone information
	formats := []string{
		"2006-01-02T15:04:05", // full date and time
		"2006-01-02T15:04",
		"2006-01-02",
		"15:04:05",
		"15:04",
	}

	for _, f := range formats {
		parsedTime, err = time.Parse(f, isoFormat)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to parse fuzzytime result as time: %s", isoFormat)
	}

	// Extract time components and reconstruct as UTC (ignoring timezone)
	year, month, day := parsedTime.Date()
	hour, min, sec := parsedTime.Clock()
	utcTime := time.Date(year, month, day, hour, min, sec, 0, time.UTC)

	return utcTime.Format("2006-01-02T15:04:05"), nil
}

// cleanDateString implements the cleaning logic from seshu_service.go
func cleanDateString(dateStr string) string {
	// Normalize non-breaking spaces to regular spaces
	cleanedDateStr := strings.ReplaceAll(dateStr, "\u00A0", " ")

	// Handle time ranges by splitting on various separators and keeping only the start time
	// Examples:
	// "Fri, Jul 25 - Jul 26" -> "Fri, Jul 25"
	// "Saturday, July 26, 2025 at 6:30PM – 9:30PM" -> "Saturday, July 26, 2025 at 6:30PM"
	// "Sep 12 at 10:00AM – Sep 13 at 5:00PM" -> "Sep 12 at 10:00AM"
	// "Saturday 26 July 2025 from 18:30-21:30" -> "Saturday 26 July 2025 from 18:30"

	var yearFromRight string

	// Check if this looks like a date range or time range
	if looksLikeDateRange(cleanedDateStr) || looksLikeTimeRange(cleanedDateStr) {
		// Split on various range separators (order matters - check longer patterns first)
		separators := []string{" – ", "–", " - ", "-", " | "}
		for _, sep := range separators {
			if strings.Contains(cleanedDateStr, sep) {
				parts := strings.Split(cleanedDateStr, sep)
				if len(parts) > 1 {
					leftPart := strings.TrimSpace(parts[0])
					rightPart := strings.TrimSpace(parts[1])

					// Always take the left part (start time)
					cleanedDateStr = leftPart

					// If we find a year in the right part, remember it for later
					yearFromRight = findYearInString(rightPart)
					break
				}
			}
		}
	}

	// Remove "from" keyword that might confuse dateparse
	cleanedDateStr = strings.ReplaceAll(cleanedDateStr, " from ", " ")

	// Remove "at" keyword that confuses dateparse
	cleanedDateStr = strings.ReplaceAll(cleanedDateStr, " at ", " ")

	// Remove Facebook's "and X more" pattern (e.g., "and 15 more")
	andMoreRegex := regexp.MustCompile(`\s+and\s+\d{1,2}\s+more\s*$`)
	cleanedDateStr = andMoreRegex.ReplaceAllString(cleanedDateStr, "")

	// Let fuzzytime handle day of week prefixes - it's more robust than string stripping

	// If we found a year in the right part of a range, use yearless parsing approach
	if yearFromRight != "" {
		cleanedDateStr = parseWithYearlessApproach(cleanedDateStr, yearFromRight)
	}
	// Note: addNextFutureYear is called later in ParseMaybeMultiDayEvent for fuzzytime fallback only
	return cleanedDateStr
}

// parseWithYearlessApproach parses a date string without year, then sets the desired year
// This handles cases where we found a year in the right part of a range
func parseWithYearlessApproach(dateStr, year string) string {
	// Convert full month names to abbreviations for better dateparse compatibility
	convertedStr := convertFullMonthToAbbrev(dateStr)

	// Remove "at" and "@" keywords that might confuse dateparse
	convertedStr = strings.ReplaceAll(convertedStr, " at ", " ")
	convertedStr = strings.ReplaceAll(convertedStr, " @ ", " ")

	// Remove comma after day (e.g., "Jul 25, 3:00 PM" -> "Jul 25 3:00 PM")
	// This handles patterns like "Jul 25, 3:00 PM" or "July 25, 3:00 PM"
	convertedStr = strings.ReplaceAll(convertedStr, ", ", " ")

	// Try to parse the yearless string
	parsed, err := dateparse.ParseAny(convertedStr)
	if err != nil {
		// If parsing fails, return the original string and let the main flow handle it
		return dateStr
	}

	// Convert year string to int
	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return dateStr
	}

	// Create new time with the desired year
	newTime := time.Date(yearInt, parsed.Month(), parsed.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), time.UTC)

	return newTime.Format(time.RFC3339)
}

// convertFullMonthToAbbrev converts full month names to abbreviations
func convertFullMonthToAbbrev(input string) string {
	monthMap := map[string]string{
		"January":   "Jan",
		"February":  "Feb",
		"March":     "Mar",
		"April":     "Apr",
		"May":       "May",
		"June":      "Jun",
		"July":      "Jul",
		"August":    "Aug",
		"September": "Sep",
		"October":   "Oct",
		"November":  "Nov",
		"December":  "Dec",
	}

	result := input
	for full, abbrev := range monthMap {
		result = strings.ReplaceAll(result, full, abbrev)
	}
	return result
}

func findYearInString(str string) string {
	// Use a more flexible approach: find any 4-digit number that looks like a year
	// This handles cases like "Jul 26 at 4:00pm, 2025" or "2025" or "at 4:00pm, 2025"
	// or "July 25 at 3:00 PM, 2024" or "Friday, August 15 at 2:30 PM, 2024"

	// Look for any 4 consecutive digits that could be a year
	for i := 0; i <= len(str)-4; i++ {
		if isAllDigits(str[i : i+4]) {
			// Check if it's a reasonable year
			year := int(str[i]-'0')*1000 + int(str[i+1]-'0')*100 + int(str[i+2]-'0')*10 + int(str[i+3]-'0')
			if year >= 1900 && year <= 2100 {
				return str[i : i+4]
			}
		}
	}
	return ""

}

// isAllDigits checks if a string contains only digits
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// addNextFutureYear adds the appropriate year to ambiguous dates (no year specified)
// Uses "next future" logic: if the date has already passed this year, use next year
func addNextFutureYear(dateStr string) string {
	// Check if the string already contains a 4-digit year
	yearRegex := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	if yearRegex.MatchString(dateStr) {
		return dateStr // Already has a year
	}

	// Check if the string contains a 2-digit year (like "70" for 1970)
	// Look for 2-digit numbers that are likely years (not day numbers or time components)
	// Years are typically at the end of the string or after a comma, but not followed by ":"
	// We need to be more specific to avoid matching day numbers
	twoDigitYearRegex := regexp.MustCompile(`,\s*\d{2}(?:\s*$|\s+[A-Z]{3,}|\s+[A-Z]{2}\s)`)
	if twoDigitYearRegex.MatchString(dateStr) {
		return dateStr // Already has a 2-digit year
	}

	// Check if it looks like a date (contains month name or day)
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	hasMonth := false
	upper := strings.ToUpper(dateStr)
	for _, month := range monthNames {
		if strings.Contains(upper, strings.ToUpper(month)) {
			hasMonth = true
			break
		}
	}

	if !hasMonth {
		return dateStr // Doesn't look like a date
	}

	// Try to parse the date with current year first to determine if it's in the future
	now := getCurrentTime()
	currentYear := now.Year()
	testDate := dateStr + ", " + fmt.Sprintf("%d", currentYear)

	// For Facebook-style formats (with timezone abbreviations), just use current year
	// since fuzzytime will handle the parsing properly

	tzRegex := regexp.MustCompile(`[A-Z]{3}`)
	if tzRegex.MatchString(testDate) {
		return testDate
	}

	// For simple formats like "July 25", try parsing to determine if it's in the future
	formats := []string{
		testDate,                              // "July 25, 2025"
		strings.ReplaceAll(testDate, ",", ""), // "July 25 2025"
	}

	for _, format := range formats {
		if parsed, err := dateparse.ParseAny(format); err == nil {
			// Current year works, check if the date is in the future
			if parsed.After(now) {
				// Date is in the future, use current year
				return testDate
			} else {
				// Date has passed this year, use next year
				return dateStr + ", " + fmt.Sprintf("%d", currentYear+1)
			}
		} else {
		}
	}

	// If current year doesn't work with any format, try next year
	nextYear := currentYear + 1
	return dateStr + ", " + fmt.Sprintf("%d", nextYear)
}

// looksLikeDateRange checks if the string contains date range patterns
func looksLikeDateRange(str string) bool {
	// Look for patterns like "Jul 25 - Jul 26" or "Sep 12 – Oct 4"
	dateRangePatterns := []string{
		" - ", "–", " | ",
	}

	for _, pattern := range dateRangePatterns {
		if strings.Contains(str, pattern) {
			// Additional check: look for month abbreviations around the separator
			parts := strings.Split(str, pattern)
			if len(parts) >= 2 {
				left := strings.ToUpper(parts[0])
				right := strings.ToUpper(parts[1])

				// Check if both sides contain month-like patterns (both abbreviated and full names)
				months := []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN",
					"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
					"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
					"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER"}

				leftHasMonth := false
				rightHasMonth := false

				for _, month := range months {
					if strings.Contains(left, month) {
						leftHasMonth = true
					}
					if strings.Contains(right, month) {
						rightHasMonth = true
					}
				}

				if leftHasMonth && rightHasMonth {
					return true
				}
			}
		}
	}

	return false
}

// looksLikeTimeRange checks if the string contains time range patterns
func looksLikeTimeRange(str string) bool {
	// Look for patterns like "6:30PM – 9:30PM" or "18:30-21:30"
	timeRangePatterns := []string{
		" – ", "–", " - ", "-", " | ",
	}

	for _, pattern := range timeRangePatterns {
		if strings.Contains(str, pattern) {
			// Additional check: look for time patterns around the separator
			parts := strings.Split(str, pattern)
			if len(parts) >= 2 {
				left := strings.TrimSpace(parts[0])
				right := strings.TrimSpace(parts[1])

				// Check if both sides contain time-like patterns
				leftHasTime := strings.Contains(left, ":") || strings.Contains(left, "AM") || strings.Contains(left, "PM")
				rightHasTime := strings.Contains(right, ":") || strings.Contains(right, "AM") || strings.Contains(right, "PM")

				if leftHasTime && rightHasTime {
					return true
				}
			}
		}
	}

	return false
}

// getNextOccurrenceYear finds the next year when the given month/day will occur
func getNextOccurrenceYear(month, day int) int {
	now := getCurrentTime()
	currentYear := now.Year()

	// Try current year first
	testDate := time.Date(currentYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if testDate.After(now) {
		return currentYear
	}

	// If current year has passed, try next year
	return currentYear + 1
}

func PrettifyTime(t string) string {
	layouts := []string{
		"2006-01-02T15:04:05", // with seconds
		"2006-01-02T15:04",    // without seconds
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, t)
		if err == nil {
			return parsed.Format("Jan 2, 2006 3:04 PM")
		}
	}

	// fallback if unparseable
	return t
}
