package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/meetnearme/api/functions/gateway/types"
)

var converter = md.NewConverter("", true, nil)

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

func UnpadJSON(jsonStr string) string {
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, []byte(jsonStr)); err != nil {
		log.Println("Error unpadding JSON: ", err)
		return jsonStr
	}
	return buffer.String()
}

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

	targetHostPort := "localhost:8000" // The string we want to replace
	replacementHost := "devnear.me"

	isLocalAct := os.Getenv("IS_LOCAL_ACT")
	if isLocalAct == "true" {
		unescapedURL = strings.ReplaceAll(unescapedURL, targetHostPort, replacementHost)
	}

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

	// TODO: figure out why this isn't working
	// Use regex to remove incomplete JSON that OpenAI sometimes returns
	unpaddedJSON := UnpadJSON(messageContentArray)

	return sessionId, unpaddedJSON, nil
}

func ExtractEventsFromHTML(urlToScrape string, action string, scraper ScrapingService) (eventsFound []types.EventInfo, htmlContent string, err error) {
	isFacebook := IsFacebookEventsURL(urlToScrape)
	if isFacebook {
		validate := func(content string) bool {
			return strings.Contains(content, `"__typename":"Event"`)
		}
		html, err := scraper.GetHTMLFromURLWithRetries(urlToScrape, 7500, true, "script[data-sjs][data-content-len]", 7, validate)
		if err != nil {
			return nil, "", err
		}
		eventsFound, err = FindFacebookEventData(html)
		return eventsFound, html, err
	}
	html, err := scraper.GetHTMLFromURL(urlToScrape, 4500, true, "")
	if err != nil {
		return nil, "", err
	}

	// TODO: refactor this so extraction can be done as a separate step that doesn't
	// include the

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

	_, response, err := CreateChatSession(string(jsonPayload))
	if err != nil {
		return nil, "", err
	}

	var events []types.EventInfo
	err = json.Unmarshal([]byte(response), &events)
	for i := range events {
		events[i].EventSource = action
	}
	return events, html, err
}
