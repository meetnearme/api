package services

import (
	"fmt"
	"log"
	"regexp"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

const ()

func GetCity(location string, baseUrl string) (city string, err error) {
	return GetCityService().GetCity(location, baseUrl)
}

func (s *RealCityService) GetCity(location string, baseUrl string) (city string, err error) {

	if baseUrl == "" {
		return "", fmt.Errorf("base URL is empty")
	}
	targetUrl := helpers.GEO_BASE_URL + "?address=" + location
	log.Println("targetUrl", targetUrl)
	htmlString, err := GetHTMLFromURL(types.SeshuJob{NormalizedUrlKey: targetUrl}, 0, true, "")
	if err != nil {
		return "", err
	}

	// this regex captures the city which appears after the string H8X2 X2R within a group denoted by () or []
	re := regexp.MustCompile(`H8X2\+X2R\s+([^"]+)`)

	matches := re.FindStringSubmatch(htmlString)
	if len(matches) < 2 {
		return "", fmt.Errorf("location is not valid")
	}

	city = matches[1]
	fmt.Print(city)

	return city, nil
}
