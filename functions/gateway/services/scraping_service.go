package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const URLEscapedErrorMsg = "ERR: URL must not be encoded, it should look like this 'https://example.com/path?query=value'"

// Add this interface at the top of the file
type ScrapingService interface {
	GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool) (string, error)
}

// Modify the existing function to be a method on a struct
type RealScrapingService struct{}

func (s *RealScrapingService) GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, timeout, jsRender, waitFor)
}

func GetHTMLFromURLWithBase(baseURL, unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error) {

	// TODO: Escaping twice, thrice or more is unlikely, but this just makes sure the URL isn't
	// single or double-encoded when passed as a param
	firstPassUrl, err := url.QueryUnescape(unescapedURL)
	if err != nil {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}
	secondPassUrl, err := url.QueryUnescape(firstPassUrl)
	if err != nil {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}
	if unescapedURL != secondPassUrl {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}

	escapedURL := url.QueryEscape(unescapedURL)

	// start of scraping API code
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second * 5,
	}
	scrapingUrl := baseURL + "?api_key=" + os.Getenv("SCRAPINGBEE_API_KEY") + "&url=" + escapedURL + "&wait=" + fmt.Sprint(timeout) + "&render_js=" + fmt.Sprint(jsRender)
	if waitFor != "" {
		scrapingUrl += "&wait_for=" + waitFor
	}
	log.Println("unescapedURL: ", unescapedURL)
	log.Println("escapedURL: ", escapedURL)
	log.Println("scrapingUrl: ", scrapingUrl)
	req, err := http.NewRequest("GET", scrapingUrl, nil)
	if err != nil {
		return "", fmt.Errorf(">>>56 ERR: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println("ERR at 57: ", err)
		return "", fmt.Errorf(">>>64 ERR: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf(">>>70 ERR: %v", err)
	}

	if res.StatusCode != 200 {
		log.Printf("71 ERR: RES =  %+v", res)
		err := fmt.Errorf("%v from scraping API", fmt.Sprint(res.StatusCode))
		return "", fmt.Errorf(">>>>76 ERR: %v", err)
	}

	htmlString := string(body)
	return htmlString, nil
}

func GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, timeout, jsRender, waitFor)
}
