package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

const URLEscapedErrorMsg = "ERR: URL must not be encoded, it should look like this 'https://example.com/path?query=value'"

func GetHTMLFromURLWithBase(baseURL, unescapedURL string, timeout int, jsRender bool) (string, error) {
	escapedURL, err := url.QueryUnescape(unescapedURL)
	if err != nil {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}
	decodedURL, err := url.QueryUnescape(escapedURL)
	if err != nil {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}
	if unescapedURL != decodedURL {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}

	// start of scraping API code
	client := &http.Client{}
	scrapingUrl := baseURL + "?api_key=" + os.Getenv("SCRAPINGBEE_API_KEY") + "&url=" + escapedURL + "&wait=" + fmt.Sprint(timeout) + "&render_js=" + fmt.Sprint(jsRender)
	log.Println("scrapingUrl: ", scrapingUrl)
	req, err := http.NewRequest("GET", scrapingUrl, nil)
	if err != nil {
		return "", fmt.Errorf("ERR: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println("ERR: ", err)
		return  "", fmt.Errorf("ERR: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return  "", fmt.Errorf("ERR: %v", err)
	}

	if res.StatusCode!= 200 {
		err := fmt.Errorf("%v from scraping API", fmt.Sprint(res.StatusCode))
		return  "", fmt.Errorf("ERR: %v", err)
	}

	htmlString := string(body)
	return htmlString, nil
}

func GetHTMLFromURL(unescapedURL string, timeout int, jsRender bool) (string, error) {
	defaultBaseURL := "https://app.scrapingbee.com/api/v1"
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, timeout, jsRender)
}
