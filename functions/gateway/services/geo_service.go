package services

import (
	"fmt"
	"regexp"
	"strings"
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
	re := regexp.MustCompile(`\[\-?\+?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*\-?\+?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)\]`)
	latLon := re.FindString(htmlString)
	if latLon == "" {
		return "", "", "", fmt.Errorf("location is not valid")
	}

	latLonArr := strings.Split(latLon, ",")
	lat = latLonArr[0]
	lon = latLonArr[1]
	re = regexp.MustCompile(`[^\d.]`)
	lat = re.ReplaceAllString(lat, "")
	lon = re.ReplaceAllString(lon, "")

	// Regular expression pattern
	pattern := `"([^"]*)"\s*,\s*\` + latLon
	re = regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(htmlString)
	if len(matches) > 0 {
		address = matches[1]
	} else {
		address = "No address found"
	}

	return lat, lon, address, nil
}
