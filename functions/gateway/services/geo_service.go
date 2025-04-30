package services

import (
	"fmt"
	"regexp"
)

const (
	LatitudeRegex  = `^[-+]?([1-8]?\d(\.\d+)?|90(\.0+)?)$`
	LongitudeRegex = `^[-+]?((1[0-7]\d)|([1-9]?\d))(\.\d+)?$`
)

func GetGeo(location string, baseUrl string) (lat string, lon string, address string, err error) {
	return GetGeoService().GetGeo(location, baseUrl)
}

func (s *RealGeoService) GetGeo(location string, baseUrl string) (lat string, lon string, address string, err error) {
	// TODO: this needs to be parameterized!
	if baseUrl == "" {
		return "", "", "", fmt.Errorf("base URL is empty")
	}
	htmlString, err := GetHTMLFromURL(baseUrl+"/map-embed?address="+location, 0, true, "#mapDiv")
	if err != nil {
		return "", "", "", err
	}
	// this regex specifically captures the pattern of a lat/lon pair e.g. [40.7128, -74.0060]
	re := regexp.MustCompile(`\[(\-?(?:[1-8]?\d(?:\.\d+)?|90(?:\.0+)?)),\s*(\-?(?:180(?:\.0+)?|(?:1[0-7]\d|[1-9]?\d)(?:\.\d+)?))\]`)

	matches := re.FindStringSubmatch(htmlString)
	if len(matches) < 3 {
		return "", "", "", fmt.Errorf("location is not valid")
	}

	lat = matches[1]
	lon = matches[2]

	// This regex captures the pattern of an address string that is wrapped in double quotes
	// and followed by a lat/lon pair e.g. "123 Main St", [40.7128, -74.0060]
	pattern := `"([^"]*)"\s*,\s*\[\s*` + regexp.QuoteMeta(lat) + `\s*,\s*` + regexp.QuoteMeta(lon) + `\s*\]`
	re = regexp.MustCompile(pattern)
	matches = re.FindStringSubmatch(htmlString)
	if len(matches) > 0 {
		address = matches[1]
	} else {
		address = "No address found"
	}

	return lat, lon, address, nil
}
