package services

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

// HTMLFetcher interface allows us to mock the HTML fetching behavior
type CityHTMLFetcher interface {
	GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error)
}

// RealHTMLFetcher wraps the actual GetHTMLFromURL function
type RealCityHTMLFetcher struct{}

func (r *RealCityHTMLFetcher) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return GetHTMLFromURL(seshuJob, waitMs, jsRender, waitFor)
}

const ()

// extractCityAndState extracts city and state from a full address
// Examples:
// "1st street New York, NY 10001, USA" -> "New York, NY"
// "123 Main St, San Francisco, CA 94102, USA" -> "San Francisco, CA"
// "Georgetown, TX" -> "Georgetown, TX"
func extractCityAndState(fullAddress string) string {
	// Split by commas to get address parts
	parts := strings.Split(fullAddress, ",")

	// If we have at least 2 parts, look for city, state pattern
	if len(parts) >= 2 {
		// Look for the pattern "City, State" in the middle parts
		for i := 0; i < len(parts)-1; i++ {
			cityPart := strings.TrimSpace(parts[i])
			statePart := strings.TrimSpace(parts[i+1])

			// Check if state part looks like a state (2-3 letters, possibly with zip)
			stateRegex := regexp.MustCompile(`^[A-Z]{2,3}(\s+\d{5})?$`)
			if stateRegex.MatchString(statePart) {
				// Found city, state pattern
				return cityPart + ", " + strings.Fields(statePart)[0] // Remove zip if present
			}
		}
	}

	// If no clear city, state pattern found, try to extract from the end
	// Look for pattern like "City, State" at the end
	if len(parts) >= 2 {
		lastPart := strings.TrimSpace(parts[len(parts)-1])

		// Check if last part is a country (USA, United States, etc.)
		countryRegex := regexp.MustCompile(`(?i)^(usa|united states|us)$`)
		if countryRegex.MatchString(lastPart) && len(parts) >= 3 {
			// Look at the part before the country
			statePart := strings.TrimSpace(parts[len(parts)-2])
			cityPart := strings.TrimSpace(parts[len(parts)-3])

			stateRegex := regexp.MustCompile(`^[A-Z]{2,3}(\s+\d{5})?$`)
			if stateRegex.MatchString(statePart) {
				return cityPart + ", " + strings.Fields(statePart)[0]
			}
		}
	}

	// Fallback: if we can't parse it properly, return the original address
	return fullAddress
}

func GetCity(location string, baseUrl string) (city string, err error) {
	return GetCityService().GetCity(location, baseUrl)
}

func (s *RealCityService) GetCity(location string, baseUrl string) (city string, err error) {

	htmlFetcher := s.htmlFetcher
	if htmlFetcher == nil {
		htmlFetcher = &RealHTMLFetcher{}
	}

	if baseUrl == "" {
		return "", fmt.Errorf("base URL is empty")
	}
	targetUrl := helpers.GEO_BASE_URL + "?address=" + location
	log.Println("targetUrl", targetUrl)
	htmlString, err := htmlFetcher.GetHTMLFromURL(types.SeshuJob{NormalizedUrlKey: targetUrl}, 0, true, "")
	if err != nil {
		return "", err
	}

	// this regex captures the city which appears after the string H8X2 X2R within a group denoted by () or []
	// re := regexp.MustCompile(`H8X2\+X2R\s+([^"]+)`)

	// matches := re.FindStringSubmatch(htmlString)
	// fmt.Printf("matches are %#v", matches)
	// if len(matches) < 1 {
	// 	return "", fmt.Errorf("location is not valid")
	// }

	// this regex specifically captures the pattern of a lat/lon pair e.g. [40.7128, -74.0060]
	re := regexp.MustCompile(`\[(\-?(?:[1-8]?\d(?:\.\d+)?|90(?:\.0+)?)),\s*(\-?(?:180(?:\.0+)?|(?:1[0-7]\d|[1-9]?\d)(?:\.\d+)?))\]`)

	coordMatches := re.FindStringSubmatch(htmlString)
	foundValidCoordinates := len(coordMatches) >= 3
	if !foundValidCoordinates {
		return "", fmt.Errorf("location is not valid")
	}

	lat := coordMatches[1]
	lon := coordMatches[2]

	// This regex captures the pattern of an address string that is wrapped in double quotes
	// and followed by a lat/lon pair e.g. "123 Main St", [40.7128, -74.0060]
	// Use (?s) flag to make . match newlines, and clean up whitespace in the captured address
	pattern := `"((?s)[^"]*)"\s*,\s*\[\s*` + regexp.QuoteMeta(lat) + `\s*,\s*` + regexp.QuoteMeta(lon) + `\s*\]`
	re = regexp.MustCompile(pattern)
	addressMatches := re.FindStringSubmatch(htmlString) // can be a city or an actual address
	if len(addressMatches) > 0 {
		// Clean up the address by replacing newlines and extra whitespace with single spaces
		fullAddress := strings.TrimSpace(strings.ReplaceAll(addressMatches[1], "\n", " "))
		// Remove any multiple consecutive spaces
		fullAddress = regexp.MustCompile(`\s+`).ReplaceAllString(fullAddress, " ")

		// Extract city and state from the full address
		city = extractCityAndState(fullAddress)
	} else {
		city = "No address found"
	}

	//city = matches[1]
	fmt.Print(city)

	return city, nil
}
