package main

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
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator"
	"github.com/meetnearme/api/functions/lambda/go/seshu/shared"
	partials "github.com/meetnearme/api/functions/lambda/go/seshu/templates"
)

// TODO: make sure all of these edge cases are covered for / explained / documented
// 1. OpenAI can respond with `[{ event_title: "My event", "event_location": "My locat... length}]` which is invalid JSON
// 2. Timeout for ZR can fail to fetch in time... retry? or just return an error?
// 3. Add special logic for meetup.com URLs to extract event info from <script type="application/ld+json">...</script>
// 4. Add special logic for eventbrite.com URLs to extract event info from <script type="application/ld+json">...</script>
// 5. Handle the "read more" / scroll down behavior for meetup.com / eventbrite via ZR remote JS
// 6. Handle / validate that problem with scraping API Key properly aborts
// 7. Check token consumption on OpenAI response (if present) and fork session if nearing limit
// 8. Validate that the invalid metadata strings fromt the example (e.g. `Nowhere City`) do not appear in the LLM event array output
// 9. Open AI can sometimes truncate the attempted JSON like this `<valid JSON start>...events/300401920/"}, {"event_tit...} stop}]`
// 10. Validate user input is really a URL prior to consuming API resources like ZR or OpenAI
// 11. Handle the scenario below where the scraped Markdown data is so large, that it exceeds the OpenAI API limit
//    and results in the error `Error: unexpected response format, `id` missing` because OpenAI literally returns an empty
//    Chat GPT response:  {  0  [] map[]}

// 		[markdown response from ZR]...nited KingdomUnited States\"]"}]}} 0x1288b40 50405 [] false api.openai.com map[] map[] <nil> map[]   <nil> <nil> <nil> {{}}}
// 		[sst] |  +10807ms Chat GPT response:  {  0  [] map[]}
// 		[sst] |  +10807ms Error creating chat session: unexpected response format, `id` missing
// 		[sst] |  +10808ms 2024/04/26 15:07:55 {"errorMessage":"unexpected response format, `id` missing","errorType":"errorString"}
// 		[sst] |  Error: unexpected response format, `id` missing


var validate *validator.Validate = validator.New()
var converter = md.NewConverter("", true, nil)

type SeshuInputPayload struct {
    Url string `json:"url" validate:"required"`

}

type SeshuResponseBody struct {
	SessionID string `json:"session_id"`
	EventsFound []shared.EventInfo `json:"events_found"`
}

// Define the structure for your request payload
type CreateChatSessionPayload struct {
	Model string `json:"model"` // Set the model you want to use
	Messages []Message `json:"messages"`
}

type SendMessagePayload struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}


type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []Choice       `json:"choices"`
	Usage   map[string]int `json:"usage"`
}

var systemPrompt = `You are a helpful LLM capable of accepting an array of strings and reorganizing them according to patterns only an LLM is capable of recognizing.

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
3. Event date
4. Event URL
5. Event description

Note that some keys may be missing, for example, in the example below, the "event description" is missing. This is acceptable. The event metadata keys are not guaranteed to be present in the input array of strings.

Do not truncate the response with an ellipsis ` + "`...`" + `, list the full event array in it's entirety, unless it exceeds the context window. Your response must be a JSON array of event objects that is valid JSON following this example schema:


` + "```" + `
[{"event_title": "Fake Event Title 1", "event_location": "Nowhere City, NM 11111", "event_date": "Sep 26, 5:30-7:30pm", "event_url": "http://example.com/events/12345"},{"event_title": "Fake Event Title 2", "event_location": "Nowhere City, NM 11111", "event_date": "Oct 13, 6:30-7:30am", "event_url": "http://example.com/events/98765"}]
` + "```" + `

The input is:
=====
const textStrings = `


func Router(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
    switch req.RequestContext.HTTP.Method {
    case "POST":
				req.Headers["Access-Control-Allow-Origin"] = "*"
				req.Headers["Access-Control-Allow-Credentials"] = "true"
				return handlePost(ctx, req)
    default:
        return clientError(http.StatusMethodNotAllowed)
    }
}

func handlePost(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	var inputPayload SeshuInputPayload
	err := json.Unmarshal([]byte(req.Body), &inputPayload)
	if err != nil {
			log.Printf("Invalid JSON payload: %v", err)
			return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
			log.Printf("Invalid body: %v", err)
			return clientError(http.StatusBadRequest)
	}

	log.Println("URL: ", inputPayload.Url)
	if err != nil {
			return serverError(err)
	}

	// start of scraping API code
	client := &http.Client{}
	scrReq, scrErr := http.NewRequest("GET", "https://app.scrapingbee.com/api/v1/?api_key=" + os.Getenv("SCRAPINGBEE_API_KEY") + "&url=" + url.QueryEscape(inputPayload.Url) + "&wait=4500&render_js=true", nil)
	if scrErr != nil {
		log.Println("ERR: ", scrErr)
		return SendHTMLError(scrErr, ctx, req)
	}
	scrRes, scrErr := client.Do(scrReq)
	if scrErr != nil {
		log.Println("ERR: ", scrErr)
		return SendHTMLError(scrErr, ctx, req)
	}
	defer scrRes.Body.Close()

	scrBody, scrErr := io.ReadAll(scrRes.Body)
	if scrErr != nil {
			log.Println("ERR: ", scrErr)
			return SendHTMLError(scrErr, ctx, req)
	}

	if scrRes.StatusCode!= 200 {
		scrErr := fmt.Errorf("%v from scraping API", fmt.Sprint(scrRes.StatusCode))
		log.Println("ERR: ", scrErr)
		return SendHTMLError(scrErr, ctx, req)
	}

	scrBodyString := string(scrBody)
	// avoid extra parsing work for <head> content outside of <body>

	// TODO: this doesn't appear to be working
	re := regexp.MustCompile(`<body>(.*?)<\/body>`)
	matches := re.FindStringSubmatch(scrBodyString)
	if len(matches) > 1 {
		scrBodyString = matches[1]
	}

	markdown, err := converter.ConvertString(scrBodyString)
	if err != nil {
		log.Println("ERR: ", err)
		return SendHTMLError(err, ctx, req)
	}

	lines := strings.Split(markdown, "\n")
	// Filter out empty lines
	var nonEmptyLines []string
	for i, line := range lines {
		// 30,000 as a line limit helps stay under the OpenAI API token limit of 16k, but this is not at all precise
			if line != "" && i < 1500 {
					nonEmptyLines = append(nonEmptyLines, line)
			}
	}

	// Convert to JSON
	jsonStringBytes, err := json.Marshal(nonEmptyLines)
	if err != nil {
			log.Println("Error converting to JSON:", err)
			return SendHTMLError(err, ctx, req)
	}
	jsonString := string(jsonStringBytes)

	// TODO: consider log levels / log volume
	_, messageContent, err := CreateChatSession(jsonString)
	if err != nil {
		log.Println("Error creating chat session:", err)
		return SendHTMLError(err, ctx, req)
	}

	// TODO: `CreateChatSession` returns `SessionID` which should be stored in session data
	// which is a separate DynamoDB table that is keyed with the `SessionID` and can be used for
	// follow up internally when we detect invalidity in the OpenAI response and need to re-prompt
	// for correct output

	openAIjson := messageContent

	var eventsFound []shared.EventInfo

	err = json.Unmarshal([]byte(openAIjson), &eventsFound)
	if err != nil {
		log.Println("Error unmarshaling OpenAI response into shared.EventInfo slice:", err)
		return SendHTMLError(err, ctx, req)
	}

	if err != nil {
		log.Println("Error marshaling response body as JSON:", err)
		return SendHTMLError(err, ctx, req)
	}

	layoutTemplate := partials.EventCandidatesPartial(eventsFound)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return serverError(err)
	}

	return events.LambdaFunctionURLResponse{
			Headers: map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
			// Body: string(responseBody),
			Body: buf.String(),
	}, nil
}

func SendHTMLError(err error, ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	layoutTemplate := partials.ErrorHTML(err, req.RequestContext.RequestID)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return serverError(err)
	}

	return events.LambdaFunctionURLResponse{
			Headers: map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
			Body: buf.String(),
	}, nil
}

func CreateChatSession(markdownLinesAsArr string) (string, string, error) {
	client := &http.Client{}
	log.Println("Creating chat session")
	log.Println("systemPrompt: ", systemPrompt)
	payload := CreateChatSessionPayload{
		Model: "gpt-3.5-turbo-16k",
		Messages: []Message{
			{
				Role: "user",
				Content: systemPrompt + markdownLinesAsArr,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", "", err
	}

	req.Header.Add("Authorization", "Bearer " + os.Getenv("OPENAI_API_KEY"))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

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

	unpaddedJSON := unpadJSON(messageContentArray)

	return sessionId, unpaddedJSON, nil
}

func unpadJSON(jsonStr string) string {
    buffer := new(bytes.Buffer)
    if err := json.Compact(buffer, []byte(jsonStr)); err != nil {
        log.Println("Error unpadding JSON: ", err)
        return jsonStr
    }
    return buffer.String()
}

func SendMessage(sessionID string, message string) (string, error) {
	client := &http.Client{}

	payload := SendMessagePayload{
		Messages: []Message{
			{
				Role: "user",
				Content: message,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.openai.com/v1/chat/completions/%s/messages", sessionID), bytes.NewBuffer(payloadBytes))

	req.Header.Add("Authorization", "Bearer " + os.Getenv("OPENAI_API_KEY"))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil

}

// TODO: this should share with the gateway handler, though the
// function signature typing is different
func clientError(status int) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

// TODO: this should share with the gateway handler
func serverError(err error) (events.LambdaFunctionURLResponse, error) {
	log.Println(err.Error())

	return events.LambdaFunctionURLResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func main() {
    lambda.Start(Router)
}
