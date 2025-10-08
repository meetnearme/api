package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

// HTMLFetcher interface allows us to mock the HTML fetching behavior
type HTMLFetcher interface {
	GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error)
}

// RealHTMLFetcher wraps the actual GetHTMLFromURL function
type RealHTMLFetcher struct{}

func (r *RealHTMLFetcher) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	return GetHTMLFromURL(seshuJob, waitMs, jsRender, waitFor)
}

const (
	LatitudeRegex  = `^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?)$`
	LongitudeRegex = `^[-+]?((1[0-7]\d)|([1-9]?\d))(\.\d+)?$`
)

func GetGeo(locationQuery string, baseUrl string) (lat string, lon string, address string, err error) {
	return GetGeoService().GetGeo(locationQuery, baseUrl)
}

func (s *RealGeoService) GetGeo(locationQuery string, baseUrl string) (lat string, lon string, address string, err error) {

	htmlFetcher := s.htmlFetcher
	if htmlFetcher == nil {
		htmlFetcher = &RealHTMLFetcher{}
	}

	if baseUrl == "" {
		return "", "", "", fmt.Errorf("base URL is empty")
	}
	targetUrl := constants.GEO_BASE_URL + "?address=" + locationQuery
	// Log escaped for clarity (scraper will escape internally)
	// escaped := url.QueryEscape(targetUrl)
	// log.Println("targetUrl (escaped)", escaped)
	htmlString, err := htmlFetcher.GetHTMLFromURL(types.SeshuJob{NormalizedUrlKey: targetUrl}, 0, true, "")
	if err != nil {
		return "", "", "", err
	}

	// this regex specifically captures the pattern of a lat/lon pair e.g. [40.7128, -74.0060]
	re := regexp.MustCompile(`\[(\-?(?:[1-8]?\d(?:\.\d+)?|90(?:\.0+)?)),\s*(\-?(?:180(?:\.0+)?|(?:1[0-7]\d|[1-9]?\d)(?:\.\d+)?))\]`)

	coordMatches := re.FindStringSubmatch(htmlString)
	foundValidCoordinates := len(coordMatches) >= 3
	if !foundValidCoordinates {
		return "", "", "", fmt.Errorf("location is not valid")
	}

	lat = coordMatches[1]
	lon = coordMatches[2]

	// This regex captures the pattern of an address string that is wrapped in double quotes
	// and followed by a lat/lon pair e.g. "123 Main St", [40.7128, -74.0060]
	// Use (?s) flag to make . match newlines, and clean up whitespace in the captured address
	pattern := `"((?s)[^"]*)"\s*,\s*\[\s*` + regexp.QuoteMeta(lat) + `\s*,\s*` + regexp.QuoteMeta(lon) + `\s*\]`
	re = regexp.MustCompile(pattern)
	addressMatches := re.FindStringSubmatch(htmlString) // can be a city or an actual address
	if len(addressMatches) > 0 {
		// Clean up the address by replacing newlines and extra whitespace with single spaces
		address = strings.TrimSpace(strings.ReplaceAll(addressMatches[1], "\n", " "))
		// Remove any multiple consecutive spaces
		address = regexp.MustCompile(`\s+`).ReplaceAllString(address, " ")
	} else {
		address = "No address found"
	}

	return lat, lon, address, nil
}
