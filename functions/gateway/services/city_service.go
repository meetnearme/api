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

func GetCity(locationQuery string) (city string, err error) {
	return GetCityService().GetCity(locationQuery)
}

func (s *RealCityService) GetCity(locationQuery string) (city string, err error) {

	htmlFetcher := s.htmlFetcher
	if htmlFetcher == nil {
		htmlFetcher = &RealCityHTMLFetcher{}
	}

	targetUrl := helpers.GEO_BASE_URL + "?address=" + locationQuery
	htmlString, err := htmlFetcher.GetHTMLFromURL(types.SeshuJob{NormalizedUrlKey: targetUrl}, 0, true, "")
	if err != nil {
		return "", err
	}

	// This regex captures the plus code pattern followed by city and state
	// Plus code format: alphanumeric characters + plus sign + more alphanumeric characters (e.g., J7JW+PM7, 8F2GJPJ3+P8)
	// Followed by city, state (unabbreviated)
	re := regexp.MustCompile(`([A-Z0-9]+\+[A-Z0-9]+)\s+([^,]+),\s*([^,"\]]+)`)

	matches := re.FindStringSubmatch(htmlString)
	if len(matches) < 4 {
		return "", fmt.Errorf("plus code and city/state not found in location data")
	}

	plusCode := matches[1]
	cityName := strings.TrimSpace(matches[2])
	stateName := strings.TrimSpace(matches[3])

	cityName = strings.TrimSpace(strings.ReplaceAll(cityName, "\n", " "))
	cityName = regexp.MustCompile(`\s+`).ReplaceAllString(cityName, " ")

	stateName = strings.TrimSpace(strings.ReplaceAll(stateName, "\n", " "))
	stateName = regexp.MustCompile(`\s+`).ReplaceAllString(stateName, " ")

	city = cityName + ", " + stateName

	log.Printf("Found plus code: %s, City: %s, State: %s", plusCode, cityName, stateName)

	return city, nil
}
