package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const URLEscapedErrorMsg = "ERR: URL must not be encoded, it should look like this 'https://example.com/path?query=value'"

// ContentValidationFunc validates if scraped content meets success criteria
type ContentValidationFunc func(htmlContent string) bool

// Add this interface at the top of the file
type ScrapingService interface {
	GetHTMLFromURL(unescapedURL string, waitMs int, jsRender bool, waitFor string) (string, error)
	GetHTMLFromURLWithRetries(unescapedURL string, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error)
}

// Modify the existing function to be a method on a struct
type RealScrapingService struct{}

func (s *RealScrapingService) GetHTMLFromURL(unescapedURL string, waitMs int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, waitMs, jsRender, waitFor, 1, nil)
}

func (s *RealScrapingService) GetHTMLFromURLWithRetries(unescapedURL string, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, waitMs, jsRender, waitFor, maxRetries, validationFunc)
}

func GetHTMLFromURLWithBase(baseURL, unescapedURL string, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {

	// TODO: Escaping twice, thrice or more is unlikely, but this just makes sure the URL isn't
	// single or double-encoded when passed as a param
	targetHostPort := "localhost:8000" // The string we want to replace
	replacementHost := "devnear.me"

	isLocalAct := os.Getenv("IS_LOCAL_ACT")
	if isLocalAct == "true" {
		unescapedURL = strings.ReplaceAll(unescapedURL, targetHostPort, replacementHost)
	}

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

	// Calculate timeouts based on scraping service defaults and best practices
	// service default timeout is 140,000ms (140 seconds) - use this as our timeout
	scrapingBeeTimeoutMs := 140000
	// HTTP client timeout: ScrapingBee timeout + 30s buffer for network overhead
	httpClientTimeoutSec := (scrapingBeeTimeoutMs / 1000) + 30

	var htmlContent string
	var lastErr error

	// Retry logic for fail-fast approach
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// start of scraping API code
		client := &http.Client{
			Timeout: time.Duration(httpClientTimeoutSec) * time.Second,
		}
		scrapingUrl := baseURL + "?api_key=" + os.Getenv("SCRAPINGBEE_API_KEY") + "&url=" + escapedURL + "&render_js=" + fmt.Sprint(jsRender)

		// Add ScrapingBee timeout parameter
		scrapingUrl += "&timeout=" + fmt.Sprint(scrapingBeeTimeoutMs)

		if waitMs > 0 {
			scrapingUrl += "&wait=" + fmt.Sprint(waitMs)
		}
		if waitFor != "" {
			scrapingUrl += "&wait_for=" + url.QueryEscape(waitFor)
		}
		req, err := http.NewRequest("GET", scrapingUrl, nil)
		if err != nil {
			lastErr = fmt.Errorf("ERR: forming scraping request: %v", err)
			if maxRetries > 1 {
				log.Printf("❌ TRACE: Attempt %d for URL %s failed with error: %v", attempt, unescapedURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		res, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("ERR: executing scraping request: %v for scrapingUrl: %s, with baseURL: %s", err, scrapingUrl, baseURL)
			if maxRetries > 1 {
				log.Printf("❌ TRACE: Attempt %d for URL %s failed with error: %v", attempt, unescapedURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			lastErr = fmt.Errorf("ERR: reading scraping response body: %v", err)
			if maxRetries > 1 {
				log.Printf("❌ TRACE: Attempt %d for URL %s failed with error: %v", attempt, unescapedURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		if res.StatusCode != 200 {
			lastErr = fmt.Errorf("ERR: %v from scraping service", res.StatusCode)
			if maxRetries > 1 {
				log.Printf("❌ TRACE: Attempt %d for URL %s failed with error: %v", attempt, unescapedURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		htmlContent = string(body)

		// Apply content validation if provided and we're doing retries
		if validationFunc != nil && maxRetries > 1 {
			if validationFunc(htmlContent) {
				log.Printf("✅ TRACE: Attempt %d succeeded for URL %s - content validation passed!", attempt, unescapedURL)
				break
			} else {
				log.Printf("⚠️  TRACE: Attempt %d got response but content validation failed for URL %s", attempt, unescapedURL)
				if attempt == maxRetries {
					log.Printf("🚫 TRACE: All %d attempts failed content validation for URL %s", maxRetries, unescapedURL)
					// Continue with last response for debugging
				}
				continue
			}
		} else {
			// For single attempts or no validation function, return immediately on success
			break
		}
	}

	return htmlContent, nil
}

func GetHTMLFromURL(unescapedURL string, waitMs int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, unescapedURL, waitMs, jsRender, waitFor, 1, nil)
}
