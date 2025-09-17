package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/ringsaturn/tzf"
)

var converter = md.NewConverter("", true, nil)

// Global tzf finder for timezone derivation from coordinates
var tzfFinder tzf.F

func init() {
	var err error
	tzfFinder, err = tzf.NewDefaultFinder()
	if err != nil {
		log.Printf("Failed to initialize tzf finder: %v", err)
		// Continue without tzf functionality - timezone derivation will fail gracefully
	}
}

// DeriveTimezoneFromCoordinates attempts to derive timezone from latitude and longitude using tzf
func DeriveTimezoneFromCoordinates(lat, lng float64) string {
	if tzfFinder == nil {
		return "" // tzf not available
	}

	timezoneName := tzfFinder.GetTimezoneName(lng, lat)
	if timezoneName == "" {
		return "" // tzf couldn't determine timezone
	}

	return timezoneName
}

const URLEscapedErrorMsg = "ERR: URL must not be encoded, it should look like this 'https://example.com/path?query=value'"

// ContentValidationFunc validates if scraped content meets success criteria
type ContentValidationFunc func(htmlContent string) bool

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CreateChatSessionPayload struct {
	Model    string    `json:"model"` // Set the model you want to use
	Messages []Message `json:"messages"`
}

type SendMessagePayload struct {
	Messages []Message `json:"messages"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
	AudioTokens  int `json:"audio_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}

type Usage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details"`
}

type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint"`
}

var systemPrompt string

func GetSystemPrompt(isRecursive bool) string {
	if isRecursive {
		return `You are a helpful LLM capable of accepting an array of strings and reorganizing them according to patterns only an LLM is capable of recognizing.

Your goal is to take the javascript array input I will provide, called the ` + "`textStrings`" + ` below and return a single grouped JSON object representing one event. This object should consist of the event metadata associated with the categories below that are to be searched for. There should be no duplicate keys. The object must contain no more than one of a given event metadata. When forming this object, prioritize proximity (meaning, the closer two strings are in array position) in associating them with the same event.

If ` + "`textStrings`" + ` is empty, return an empty array.

Any response you provide must ALWAYS begin with the characters ` + "`[{`" + ` and end with the characters ` + "`}]`" + `.

Do not provide me with example code to achieve this task. Only an LLM (you are an LLM) is capable of reading the array of text strings and determining which string is a relevance match for which category can resolve this task. Javascript alone cannot resolve this query.

Do not explain how code might be used to achieve this task. Do not explain how regex might accomplish this task. Only an LLM is capable of this pattern matching task. My expectation is a response from you that is an array containing one object, where the keys are the event metadata from the categories below.

Do not return an ordered list of strings. Return a single-element array of one object, where the object represents a single event, and the keys of the object are the event metadata from the categories below.

It is understood that the strings in the input below are in some cases not a categorical match for the event metadata categories below. This is acceptable. The LLM is capable of determining which strings are a relevance match for which category. It is acceptable to discard strings that are not a relevance match for any category.

The categories to search for relevance matches in are as follows:
=====
1. Event title
2. Event location
3. Event start date / time
4. Event end date / time
5. Event URL
6. Event description

Note that some keys may be missing, for example, in the example below, the "event description" is missing. This is acceptable. The event metadata keys are not guaranteed to be present in the input array of strings.

Do not truncate the response with an ellipsis ` + "`...`" + `, list the full object in its entirety, unless it exceeds the context window. Your response must be a JSON array with one event object that is valid JSON following this example schema:

` + "```" + `
[{"event_title": "` + FakeEventTitle1 + `", "event_location": "` + FakeCity + `", "event_start_datetime": "` + FakeStartTime1 + `", "event_end_datetime": "` + FakeEndTime1 + `", "event_url": "` + FakeUrl1 + `"}]
` + "```" + `

The input is:
=====
const textStrings = `
	}
	return `You are a helpful LLM capable of accepting an array of strings and reorganizing them according to patterns only an LLM is capable of recognizing.

Your goal is to take the javascript array input I will provide, called the ` + "`textStrings`" + `below and return a grouped array of JSON objects. Each object should represent a single event, where it's keys are the event metadata associated with the categories below that are to be searched for. There should be no duplicate keys. Each object consists of no more than one of a given event metadata. When forming these groups, prioritize proximity (meaning, the closer two strings are in array position) in creating the event objects in the returned array of objects. In other words, the closer two strings are together, the higher the likelihood that they are two different event metadata items for the same event.

If ` + "`textStrings`" + ` is empty, return an empty array.

Any response you provide must ALWAYS begin with the characters ` + "`[{`" + ` and end with the characters ` + "`}]`" + `.

Do not provide me with example code to achieve this task. Only an LLM (you are an LLM) is capable of reading the array of text strings and determining which string is a relevance match for which category can resolve this task. Javascript alone cannot resolve this query.

Do not explain how code might be used to achieve this task. Do not explain how regex might accomplish this task. Only an LLM is capable of this pattern matching task. My expectation is a response from you that is an array of objects, where the keys are the event metadata from the categories below.

Do not return an ordered list of strings. Return an array of objects, where each object is a single event, and the keys of each object are the event metadata from the categories below.

It is understood that the strings in the input below are in some cases not a categorical match for the event metadata categories below. This is acceptable. The LLM is capable of determining which strings are a relevance match for which category. It is acceptable to discard strings that are not a relevance match for any category.

The categories to search for relevance matches in are as follows:
=====
1. Event title
2. Event location
3. Event start date / time
4. Event end date / time
5. Event URL
6. Event description

Note that some keys may be missing, for example, in the example below, the "event description" is missing. This is acceptable. The event metadata keys are not guaranteed to be present in the input array of strings.

Do not truncate the response with an ellipsis ` + "`...`" + `, list the full event array in it's entirety, unless it exceeds the context window. Your response must be a JSON array of event objects that is valid JSON following this example schema:

` + "```" + `
[{"event_title": "` + FakeEventTitle1 + `", "event_location": "` + FakeCity + `", "event_start_datetime": "` + FakeStartTime1 + `", "event_end_datetime": "` + FakeEndTime1 + `", "event_url": "` + FakeUrl1 + `"},{"event_title": "` + FakeEventTitle2 + `", "event_location": "` + FakeCity + `", "event_start_datetime": "` + FakeStartTime2 + `", "event_end_datetime": "` + FakeEndTime2 + `", "event_url": "` + FakeUrl2 + `"}]
` + "```" + `

The input is:
=====
const textStrings = `
}

func UnpadJSON(jsonStr string) (string, error) {
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, []byte(jsonStr)); err != nil {
		log.Println("Error unpadding JSON: ", err)
		return jsonStr, err
	}
	return buffer.String(), nil
}

// Add this interface at the top of the file
type ScrapingService interface {
	GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error)
	GetHTMLFromURLWithRetries(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error)
}

// Modify the existing function to be a method on a struct
type RealScrapingService struct{}

func (s *RealScrapingService) GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, seshuJob.NormalizedUrlKey, waitMs, jsRender, waitFor, 1, nil)
}

func (s *RealScrapingService) GetHTMLFromURLWithRetries(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, seshuJob.NormalizedUrlKey, waitMs, jsRender, waitFor, maxRetries, validationFunc)
}

func GetHTMLFromURLWithBase(baseURL, unescapedURL string, waitMs int, jsRender bool, waitFor string, maxRetries int, validationFunc ContentValidationFunc) (string, error) {
	if maxRetries == 0 {
		maxRetries = 1
	}

	// TODO: Escaping twice, thrice or more is unlikely, but this just makes sure the URL isn't
	// single or double-encoded when passed as a param
	if strings.Contains(unescapedURL, "%") {
		return "", fmt.Errorf(URLEscapedErrorMsg)
	}

	escapedURL := url.QueryEscape(unescapedURL)

	// Calculate timeouts based on scraping service defaults and best practices
	// service default timeout is 140,000ms (140 seconds) - use this as our timeout
	// HTTP client timeout: ScrapingBee timeout + 30s buffer for network overhead
	httpClientTimeoutSec := 120000

	var htmlContent string
	var lastErr error

	// Retry logic for fail-fast approach
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// start of scraping API code
		client := &http.Client{
			Timeout: time.Duration(httpClientTimeoutSec) * time.Second,
		}
		// Build ScrapingBee URL
		scrapingUrl := baseURL + "?api_key=" + os.Getenv("SCRAPINGBEE_API_KEY") + "&url=" + escapedURL

		if jsRender {
			scrapingUrl += "&render_js=true"
		}

		if waitMs > 0 {
			scrapingUrl += "&wait=" + fmt.Sprint(waitMs)
		}
		if waitFor != "" {
			scrapingUrl += "&wait_for=" + url.QueryEscape(waitFor)
		}

		// Log a sanitized summary of the request
		req, err := http.NewRequest("GET", scrapingUrl, nil)
		if err != nil {
			lastErr = fmt.Errorf("ERR: forming scraping request: %v", err)
			if maxRetries > 1 {
				log.Printf("ERR: Attempt %d for URL %s failed with error: %v", attempt, unescapedURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		res, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("ERR: executing scraping request: %v for scrapingUrl: <sanitized> baseURL: %s", err, baseURL)
			if maxRetries > 1 {
				log.Printf("ERR: Attempt %d for URL %s failed with error: %v", attempt, baseURL, lastErr)
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
				log.Printf("ERR: Attempt %d for URL %s failed with error: %v", attempt, baseURL, lastErr)
			}
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		if res.StatusCode != 200 {
			lastErr = fmt.Errorf("ERR: %v from scraping service for URL %s", res.StatusCode, baseURL)
			if attempt == maxRetries {
				return "", lastErr
			}
			continue
		}

		htmlContent = string(body)

		// Apply content validation if provided and we're doing retries
		if validationFunc != nil && maxRetries > 1 {
			if validationFunc(htmlContent) {
				break
			} else {
				if attempt == maxRetries {
					log.Printf("ERR: All %d attempts failed content validation for URL %s", maxRetries, baseURL)
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

func GetHTMLFromURL(seshuJob types.SeshuJob, waitMs int, jsRender bool, waitFor string) (string, error) {
	defaultBaseURL := os.Getenv("SCRAPINGBEE_API_URL_BASE")
	return GetHTMLFromURLWithBase(defaultBaseURL, seshuJob.NormalizedUrlKey, waitMs, jsRender, waitFor, 1, nil)
}

func CreateChatSession(markdownLinesAsArr string) (string, string, error) {
	client := &http.Client{}
	payload := CreateChatSessionPayload{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role:    "user",
				Content: systemPrompt + markdownLinesAsArr,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	log.Println(os.Getenv("OPENAI_API_BASE_URL") + "/chat/completions")

	req, err := http.NewRequest("POST", os.Getenv("OPENAI_API_BASE_URL")+"/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", "", err
	}

	log.Println("490~")

	req.Header.Add("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Add("Content-Type", "application/json")

	log.Println(req)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	log.Println(resp)

	log.Println("501~")

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("%d: Completion API request not successful", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var respData ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", "", err
	}

	sessionId := respData.ID
	if sessionId == "" {
		return "", "", fmt.Errorf("unexpected response format, `id` missing")
	}

	messageContentArray := respData.Choices[0].Message.Content
	if messageContentArray == "" {
		return "", "", fmt.Errorf("unexpected response format, `message.content` missing")
	}

	// Use regex to remove incomplete JSON that OpenAI sometimes returns
	unpaddedJSON, err := UnpadJSON(messageContentArray)
	if err != nil {
		log.Printf("Failed to convert scraped data to readable events: %v", err)
		return "", "", fmt.Errorf("Failed to convert scraped data to readable events")
	}

	return sessionId, unpaddedJSON, nil
}

func ExtractEventsFromHTML(seshuJob types.SeshuJob, mode string, scraper ScrapingService) (eventsFound []types.EventInfo, htmlContent string, err error) {
	knownScrapeSource := ""
	isFacebook := IsFacebookEventsURL(seshuJob.NormalizedUrlKey)

	if isFacebook {
		validate := func(content string) bool {
			return strings.Contains(content, `"__typename":"Event"`)
		}

		html, err := scraper.GetHTMLFromURLWithRetries(seshuJob, 7500, true, "script[data-sjs][data-content-len]", 7, validate)
		if err != nil {
			log.Printf("ERR: Failed to get HTML from Facebook URL: %v", err)
			return nil, "", err
		}

		if mode != helpers.SESHU_MODE_ONBOARD {
			childScrapeQueue := []types.EventInfo{}
			urlToIndex := make(map[string]int)

			// Use the list mode function for the main page
			eventsFound, err = FindFacebookEventListData(html, seshuJob.LocationTimezone)
			if err != nil {
				log.Printf("ERR: Failed to extract Facebook event list data: %v", err)
				return nil, "", err
			}

			for i, event := range eventsFound {
				eventsFound[i].KnownScrapeSource = knownScrapeSource
				// TODO: we could arguably this any time we have a URL,
				// searching even for things like Title, StartTime, etc.
				// but for now we're only assuming these missing fields have a
				// chance of triggering a child scrape
				if event.EventDescription == "" || event.EventLocation == "" || event.EventTimezone == "" {
					childScrapeQueue = append(childScrapeQueue, event)
					urlToIndex[event.EventURL] = i
				}
			}

			if len(childScrapeQueue) <= 0 {
				return eventsFound, html, err
			}

			for _, event := range childScrapeQueue {
				childHtml, err := scraper.GetHTMLFromURLWithRetries(types.SeshuJob{NormalizedUrlKey: event.EventURL}, 7500, true, "script[data-sjs][data-content-len]", 7, validate)
				if err != nil {
					log.Printf("ERR: Failed to get child HTML from %s: %v", event.EventURL, err)
					continue
				}
				// Use the single event mode function for child pages
				childEvArrayOfOne, err := FindFacebookSingleEventData(childHtml, seshuJob.LocationTimezone)
				if err != nil {
					log.Printf("ERR: Failed to extract single event data from child page: %v", err)
					continue
				}

				// Look up the original index using the URL
				originalIndex := urlToIndex[event.EventURL]
				if eventsFound[originalIndex].EventDescription == "" {
					eventsFound[originalIndex].EventDescription = childEvArrayOfOne[0].EventDescription
				}
				if eventsFound[originalIndex].EventLocation == "" {
					eventsFound[originalIndex].EventLocation = childEvArrayOfOne[0].EventLocation
				}

				// If we still don't have a timezone, try to derive it from coordinates
				if eventsFound[originalIndex].EventTimezone == "" &&
					childEvArrayOfOne[0].EventLatitude != 0 &&
					childEvArrayOfOne[0].EventLongitude != 0 {
					derivedTimezone := DeriveTimezoneFromCoordinates(
						childEvArrayOfOne[0].EventLatitude,
						childEvArrayOfOne[0].EventLongitude,
					)
					if derivedTimezone != "" {
						eventsFound[originalIndex].EventTimezone = derivedTimezone
					}
				}
			}

			return eventsFound, html, err

		} else {
			// For validate mode, we still need to extract events to return them
			// Use the list mode function to get events from the main page
			eventsFound, err = FindFacebookEventListData(html, seshuJob.LocationTimezone)
			if err != nil {
				log.Printf("ERR: Failed to extract Facebook event list data in %s mode: %v", mode, err)
				return nil, "", err
			}

			return eventsFound, html, err

		}
	}

	html, err := scraper.GetHTMLFromURL(seshuJob, 4500, true, "")
	if err != nil {
		return nil, "", err
	}

	var response string

	if mode == helpers.SESHU_MODE_ONBOARD {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return nil, "", err
		}
		bodyHtml, err := doc.Find("body").Html()
		if err != nil {
			return nil, "", err
		}

		markdown, err := converter.ConvertString(bodyHtml)
		if err != nil {
			return nil, "", err
		}

		lines := strings.Split(markdown, "\n")
		var filtered []string
		for i, line := range lines {
			if line != "" && i < 1500 {
				filtered = append(filtered, line)
			}
		}

		jsonPayload, err := json.Marshal(filtered)
		if err != nil {
			return nil, "", err
		}

		_, response, err = CreateChatSession(string(jsonPayload))
		if err != nil {
			return nil, "", err
		}
	}

	var events []types.EventInfo
	err = json.Unmarshal([]byte(response), &events)
	if err != nil {
		return nil, "", err
	}

	return events, html, err
}

// FilterValidEvents removes events that fail validation checks
func FilterValidEvents(events []types.EventInfo) []types.EventInfo {
	var validEvents []types.EventInfo

	for i, event := range events {

		// Check if event has required fields
		if event.EventTitle == "" || event.EventURL == "" {
			log.Printf("INFO: Filtering out event %d: missing title or URL", i)
			continue
		}

		// Check if event has location (required for GetGeo call)
		if event.EventLocation == "" {
			log.Printf("INFO: Filtering out event %d: no location", i)
			continue
		}

		// Parse the event start time to check if it's in the past
		if event.EventStartTime == "" {
			log.Printf("INFO: Filtering out event %d: no start time", i)
			continue
		}

		if event.EventTitle == "" {
			log.Printf("INFO: Filtering out event %d: no title", i)
			continue
		}

		// Handle EventDescription fallback to EventTitle if missing
		if event.EventDescription == "" {
			event.EventDescription = event.EventTitle
		}

		// Event passed all validation checks
		validEvents = append(validEvents, event)
	}

	return validEvents
}

func PushExtractedEventsToDB(events []types.EventInfo, seshuJob types.SeshuJob) error {
	// Handle empty events array gracefully
	if len(events) == 0 {
		log.Printf("No events to push to DB for %s", seshuJob.NormalizedUrlKey)
		return nil
	}

	// Get Weaviate client
	weaviateClient, err := GetWeaviateClient()
	if err != nil {
		return fmt.Errorf("failed to get Weaviate client: %w", err)
	}

	validEvents := FilterValidEvents(events)

	if len(validEvents) == 0 {
		log.Printf("No valid events to push to DB for %s, filtered %v events down to %v", seshuJob.NormalizedUrlKey, len(events), len(validEvents))
		return nil
	}

	currentTime := time.Now()

	ownerName := ""
	// fetch event owner from zitadel
	owner, err := helpers.GetOtherUserByID(seshuJob.OwnerID)
	if err != nil {
		log.Printf("Failed to get event owner from zitadel: %v", err)
		// Continue with default owner ID if zitadel lookup fails
		owner = types.UserSearchResult{UserID: seshuJob.OwnerID}
	}

	if owner.DisplayName != "" {
		ownerName = owner.DisplayName
	} else {
		ownerName = seshuJob.OwnerID
	}

	// Convert EventInfo to Event types for Weaviate
	weaviateEvents := make([]RawEvent, 0, len(validEvents))
	for i, eventInfo := range validEvents {

		// Attempt to get geo coordinates and timezone from location string
		lat, lon, address, err := GetGeo(eventInfo.EventLocation, os.Getenv("APEX_URL"))
		if err != nil {
			log.Printf("ERR: Skipping event %d: GetGeo failed: %v", i, err)
			continue
		}

		latFloat, err := strconv.ParseFloat(lat, 64)
		if err != nil {
			return fmt.Errorf("ERR: failed to parse latitude for event #%d of %s: %v", i+1, seshuJob.NormalizedUrlKey, err)
		}

		lonFloat, err := strconv.ParseFloat(lon, 64)
		if err != nil {
			return fmt.Errorf("ERR: failed to parse longitude for event #%d of %s: %v", i+1, seshuJob.NormalizedUrlKey, err)
		}

		if address == "" {
			return fmt.Errorf("ERR: couldn't find address for event #%d of %s", i+1, seshuJob.NormalizedUrlKey)
		}

		// Set latitude - use parsed value if available, otherwise fall back to seshuJob location
		var finalLat float64
		if lat == "" && seshuJob.LocationLatitude == 0 {
			return fmt.Errorf("ERR: couldn't find latittude for event #%d of %s", i+1, seshuJob.NormalizedUrlKey)
		} else if latFloat != 0 {
			finalLat = latFloat
		} else {
			finalLat = seshuJob.LocationLatitude
		}

		// Set longitude - use parsed value if available, otherwise fall back to seshuJob location
		var finalLon float64
		if lon == "" && seshuJob.LocationLongitude == 0 {
			return fmt.Errorf("couldn't find longitude for event #%d of %s", i+1, seshuJob.NormalizedUrlKey)
		} else if lonFloat != 0 {
			finalLon = lonFloat
		} else {
			finalLon = seshuJob.LocationLongitude
		}

		// Use timezone in priority order: GetGeo() response > EventInfo derived timezone > SeshuJob timezone
		var targetTimezoneStr string
		var mapDerivedTimezone string
		// First priority: GetGeo() response timezone (if available)
		if lat != "" && lon != "" {
			latFloat, err := strconv.ParseFloat(lat, 64)
			if err != nil {
				log.Printf("ERR: Skipping event %d: failed to parse latitude: %v", i, err)
				continue
			}
			lonFloat, err := strconv.ParseFloat(lon, 64)
			if err != nil {
				log.Printf("ERR: Skipping event %d: failed to parse longitude: %v", i, err)
				continue
			}
			mapDerivedTimezone = DeriveTimezoneFromCoordinates(latFloat, lonFloat)
			if mapDerivedTimezone != "" {
				targetTimezoneStr = mapDerivedTimezone
				log.Printf("INFO: Using timezone from GetGeo coordinates: %s", targetTimezoneStr)
			}

		}

		// Second priority: EventInfo derived timezone
		if targetTimezoneStr == "" && eventInfo.EventTimezone != "" {
			targetTimezoneStr = eventInfo.EventTimezone
			log.Printf("INFO: Using derived timezone from EventInfo: %s", targetTimezoneStr)
		}

		// Third priority: SeshuJob timezone
		if targetTimezoneStr == "" && seshuJob.LocationTimezone != "" {
			targetTimezoneStr = seshuJob.LocationTimezone
			log.Printf("INFO: Using timezone from SeshuJob: %s", targetTimezoneStr)
		}

		if targetTimezoneStr == "" {
			log.Printf("INFO: Skipping event %d: couldn't derive a timezone", i)
			continue
		}

		tz, err := time.LoadLocation(targetTimezoneStr)
		if err != nil {
			log.Printf("INFO: Skipping event %d: failed to load timezone %s: %v", i, targetTimezoneStr, err)
			continue
		}
		if tz == nil {
			log.Printf("INFO: Skipping event %d: failed to load timezone %s: %v", i, targetTimezoneStr, err)
			continue
		}

		// Parse the start time using the resolved timezone
		startTime, err := time.Parse(time.RFC3339, eventInfo.EventStartTime)
		if err != nil {
			log.Printf("INFO: Skipping event %d: failed to parse event start time: %v", i, err)
			continue
		}

		// Convert to local time for comparison
		startTimeLocal := startTime.In(tz)
		if startTimeLocal.Before(currentTime.Add(-1 * time.Hour)) {
			log.Printf("INFO: Filtering out event %d: event is in the past (started at %v)", i, startTimeLocal)
			continue
		}

		// Parse end time if available (optional)
		var endTime time.Time
		if eventInfo.EventEndTime != "" {
			endTime, err = time.Parse(time.RFC3339, eventInfo.EventEndTime)
			if err != nil {
				log.Printf("INFO: Failed to parse end time for event %d: %v", i, err)
				// Continue without end time - it's optional
			}
		}

		// Convert EventInfo to RawEvent with JSON-friendly fields
		// Build RFC3339 strings for times
		startRFC3339 := startTime.In(tz).Format(time.RFC3339)
		var endVal interface{}
		if !endTime.IsZero() {
			endRFC3339 := endTime.In(tz).Format(time.RFC3339)
			endVal = endRFC3339
		} else {
			endVal = nil
		}

		tzString := tz.String()
		eventSourceID := eventInfo.EventURL

		event := RawEvent{
			RawEventData: RawEventData{
				EventOwners:     []string{seshuJob.OwnerID},
				EventOwnerName:  ownerName,
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Name:            eventInfo.EventTitle,
				Description:     eventInfo.EventDescription,
				Address:         address,
				Lat:             finalLat,
				Long:            finalLon,
				Timezone:        tzString,
			},
			EventSourceId: &eventSourceID,
			StartTime:     startRFC3339,
			EndTime:       endVal,
		}
		weaviateEvents = append(weaviateEvents, event)
	}

	log.Printf("INFO: Successfully processed %d out of %d events for %s", len(weaviateEvents), len(validEvents), seshuJob.NormalizedUrlKey)

	weaviateEventsStrict, _, err := BulkValidateEvents(weaviateEvents, false)
	if err != nil {
		return fmt.Errorf("failed to validate events: %w", err)
	}

	// Bulk upsert events to Weaviate
	if len(weaviateEvents) > 0 {
		log.Printf("Upserting %d events to Weaviate for %s", len(weaviateEvents), seshuJob.NormalizedUrlKey)
		_, err = BulkUpsertEventsToWeaviate(context.Background(), weaviateClient, weaviateEventsStrict)
		if err != nil {
			return fmt.Errorf("failed to upsert events to Weaviate for %s: %v", seshuJob.NormalizedUrlKey, err)
		}
	}

	return nil
}
