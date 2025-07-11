package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

type EventFb struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	Location  string `json:"location"`
	Organizer string `json:"organizer"`
}

type EventResultFb struct {
	Success bool      `json:"success"`
	Count   int       `json:"count"`
	Events  []EventFb `json:"events"`
}

// Check if a container contains actual event data (not just navigation)
func isEventDataContainer(container *goquery.Selection) bool {
	text := strings.TrimSpace(container.Text())
	// Clean Unicode characters and normalize whitespace
	cleanText := cleanUnicodeText(text)

	// Should contain date pattern (either single time or multi-day) and not contain navigation text
	datePattern := regexp.MustCompile(`\w+,\s+\w+\s+\d+(\s+at\s+\d+:\d+\s+[AP]M\s+[A-Z]{3}|\s+-\s+\w+\s+\d+)`)
	hasDatePattern := datePattern.MatchString(cleanText)
	hasNavigationText := strings.Contains(cleanText, "EventsUpcomingPastMore") || strings.Contains(cleanText, "UpcomingPast")

	return hasDatePattern && !hasNavigationText
}

// Find location element using DOM structure
func findLocationElement(container *goquery.Selection) string {
	var location string

	container.Find("*").Each(func(i int, el *goquery.Selection) {
		if location != "" {
			return // Already found, skip
		}

		text := strings.TrimSpace(el.Text())
		// Clean Unicode characters and normalize whitespace
		cleanText := cleanUnicodeText(text)

		// Skip if this element contains date patterns (to avoid date/title elements)
		datePattern := regexp.MustCompile(`\w+,\s+\w+\s+\d+(\s+at\s+\d+:\d+\s+[AP]M|\s+-\s+\w+\s+\d+)`)
		if datePattern.MatchString(cleanText) {
			return
		}

		// Look for location pattern with comma + whitespace structure
		locationPattern := regexp.MustCompile(`([^,]+),\s+([^,]+),\s+([^,]+)(?:,\s+([^,]+))?`)
		if locationPattern.MatchString(cleanText) && len(cleanText) < 200 {
			// Extract everything before any separator
			cleanTextParts := strings.Split(cleanText, " ·")[0]
			cleanTextParts = strings.Split(cleanTextParts, " Event by")[0]
			cleanTextParts = strings.TrimSpace(cleanTextParts)

			// If it looks like it starts with a street address, use it
			streetPattern := regexp.MustCompile(`^\d+\s+`)
			if streetPattern.MatchString(cleanTextParts) {
				// Remove redundant state name and zip code at the end
				cleanedPattern := regexp.MustCompile(`,\s+[A-Za-z]+\s+\d{5}(-\d{4})?\s*$`)
				cleanedLocation := cleanedPattern.ReplaceAllString(cleanTextParts, "")
				location = cleanedLocation
			} else if strings.Contains(cleanTextParts, ",") {
				location = cleanTextParts
			}
		}
	})

	return location
}

// cleanUnicodeText removes problematic Unicode characters while preserving readable text
func cleanUnicodeText(text string) string {
	// Remove specific problematic Unicode characters
	text = strings.ReplaceAll(text, string([]byte{226, 128, 175}), "") // Remove bytes 226 128 175
	text = strings.ReplaceAll(text, "\u202f", " ")                     // Replace narrow no-break space with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")                     // Replace non-breaking space with regular space

	// Remove other non-printable characters but preserve middle dot and normal punctuation
	var result strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == '·' || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Clean up extra whitespace
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(result.String(), " "))
}

// findEventData extracts event data from Facebook events pages
func findEventData(htmlContent string) ([]EventFb, error) {
	// First, check if the HTML contains event data
	if !strings.Contains(htmlContent, `"__typename":"Event"`) {
		return nil, fmt.Errorf("no Facebook event data found (no __typename: Event)")
	}

	// Find script tags containing event data
	scriptPattern := regexp.MustCompile(`<script[^>]*>(.*?)</script>`)
	scriptMatches := scriptPattern.FindAllStringSubmatch(htmlContent, -1)

	var eventScriptContent string
	for _, match := range scriptMatches {
		if len(match) >= 2 {
			scriptContent := match[1]
			if strings.Contains(scriptContent, `"__typename":"Event"`) {
				eventScriptContent = scriptContent
				break
			}
		}
	}

	if eventScriptContent == "" {
		return nil, fmt.Errorf("no script tag found containing event data")
	}

	// Clean up the script content (remove extra whitespace)
	eventScriptContent = strings.TrimSpace(eventScriptContent)

	// Log the JSON for debugging
	fmt.Printf("DEBUG: Found event script content (first 1000 chars):\n%s\n", eventScriptContent[:min(1000, len(eventScriptContent))])

	// Extract events from the JSON content
	events := extractEventsFromJSON(eventScriptContent)

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}

// extractEventsFromJSON extracts events from JSON content
func extractEventsFromJSON(jsonContent string) []EventFb {
	var events []EventFb
	eventID := 1

	// Look for event objects in the JSON
	eventPattern := regexp.MustCompile(`"__typename":"Event"[^}]*?"name":"([^"]+)"[^}]*?"url":"([^"]+)"`)
	eventMatches := eventPattern.FindAllStringSubmatch(jsonContent, -1)

	fmt.Printf("DEBUG: Found %d event matches\n", len(eventMatches))

	for _, match := range eventMatches {
		if len(match) >= 3 {
			title := cleanUnicodeText(match[1])
			url := unescapeJSON(match[2]) // Unescape the URL

			// Extract date pattern
			datePattern := regexp.MustCompile(`"day_time_sentence":"([^"]+)"`)
			dateMatches := datePattern.FindStringSubmatch(jsonContent)
			var date string
			if len(dateMatches) >= 2 {
				date = cleanUnicodeText(unescapeJSON(dateMatches[1])) // Unescape date
			}

			// Extract location pattern
			locationPattern := regexp.MustCompile(`"contextual_name":"([^"]+)"`)
			locationMatches := locationPattern.FindStringSubmatch(jsonContent)
			var location string
			if len(locationMatches) >= 2 {
				location = cleanUnicodeText(unescapeJSON(locationMatches[1])) // Unescape location
			}

			// Extract organizer pattern
			organizerPattern := regexp.MustCompile(`"event_creator"[^}]*?"name":"([^"]+)"`)
			organizerMatches := organizerPattern.FindStringSubmatch(jsonContent)
			var organizer string
			if len(organizerMatches) >= 2 {
				organizer = cleanUnicodeText(unescapeJSON(organizerMatches[1])) // Unescape organizer
			}

			fmt.Printf("DEBUG: Event %d - Title: %s, URL: %s, Date: %s, Location: %s, Organizer: %s\n",
				eventID, title, url, date, location, organizer)

			// Only add events with required fields
			if title != "" && url != "" {
				events = append(events, EventFb{
					ID:        eventID,
					Date:      date,
					Title:     title,
					URL:       url,
					Location:  location,
					Organizer: organizer,
				})
				eventID++
			}
		}
	}

	return events
}

// unescapeJSON unescapes JSON-encoded strings (removes \\/ and other escape sequences)
func unescapeJSON(s string) string {
	// Replace escaped forward slashes
	s = strings.ReplaceAll(s, `\/`, `/`)
	// Replace escaped backslashes
	s = strings.ReplaceAll(s, `\\`, `\`)
	// Replace escaped quotes
	s = strings.ReplaceAll(s, `\"`, `"`)
	// Replace escaped newlines
	s = strings.ReplaceAll(s, `\n`, "\n")
	// Replace escaped tabs
	s = strings.ReplaceAll(s, `\t`, "\t")
	// Replace Unicode escape sequences
	s = strings.ReplaceAll(s, `\u202f`, " ") // Narrow no-break space
	s = strings.ReplaceAll(s, `\u00a0`, " ") // Non-breaking space
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Check if URL is a Facebook events page
func isFacebookEventsURL(targetURL string) bool {
	// Pattern: facebook.com/<any string>/events
	pattern := regexp.MustCompile(`facebook\.com\/[^\/]+\/events`)
	return pattern.MatchString(targetURL)
}

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <URL>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s https://www.facebook.com/aiicomics/events\n", os.Args[0])
		os.Exit(1)
	}

	targetURL := os.Args[1]

	// Validate URL format
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		log.Fatalf("URL must start with http:// or https://")
	}

	// Check if this is a Facebook events URL
	if !isFacebookEventsURL(targetURL) {
		log.Fatalf("This tool currently only supports Facebook events URLs (pattern: facebook.com/<page>/events)")
	}

	// Initialize scraping service
	scrapingService := &RealScrapingService{}

	// Fetch HTML from URL using scraping service
	// Use JavaScript rendering for Facebook pages and wait for events content
	htmlContent, err := scrapingService.GetHTMLFromURL(targetURL, 60, true, `[data-pagelet="ProfileAppSection_0"]`)
	if err != nil {
		log.Fatalf("Failed to fetch HTML from URL: %v", err)
	}

	// Debug: Show HTML content info
	fmt.Fprintf(os.Stderr, "DEBUG: HTML content length: %d bytes\n", len(htmlContent))

	// Debug: Save HTML to file for inspection
	debugFile := "debug_scraped_content.html"
	if err := os.WriteFile(debugFile, []byte(htmlContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: Failed to write debug file: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG: HTML content saved to %s\n", debugFile)
	}

	// Debug: Show first 500 chars of HTML
	preview := htmlContent
	if len(preview) > 500 {
		preview = preview[:500]
	}
	fmt.Fprintf(os.Stderr, "DEBUG: HTML preview:\n%s\n", preview)

	// Extract event data using our existing function
	events, err := findEventData(htmlContent)
	if err != nil {
		log.Fatalf("Event data extraction failed: %v", err)
	}

	// Create result structure
	result := EventResultFb{
		Success: true,
		Count:   len(events),
		Events:  events,
	}

	// Convert to JSON and print
	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonOutput))
}
