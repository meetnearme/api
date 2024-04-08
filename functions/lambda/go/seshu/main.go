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
)

var validate *validator.Validate = validator.New()
var converter = md.NewConverter("", true, nil)

type SeshuInputPayload struct {
    Url string `json:"url" validate:"required"`
}

type SeshuResponseBody struct {
	SessionID string `json:"session_id"`
	EventsMap interface{} `json:"events_map"`
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
	Role    string      `json:"role"`
	Content string `json:"content"`
}

type EventInfo struct {
	EventTitle    string `json:"event_title"`
	EventLocation string `json:"event_location"`
	EventDate     string `json:"event_date"`
	EventURL      string `json:"event_url"`
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

Do not truncate the response with an ellipsis ` + "`...`" + `, list the full event array in it's entirety. Your response must be a JSON array of event objects that is valid JSON following this example schema:

` + "```" + `
[{
	"event_title": "Meetup at the park",
	"event_location": "Espanola, NM 87532",
	"event_date": "Sep 26, 5:30-7:30pm",
	"event_url": "http://example.com/events/12345"
},
{
	"event_title": "Yoga at sunrise",
	"event_location": "Espanola, NM 87532",
	"event_date": "Oct 13, 6:30-7:30am",
	"event_url": "http://example.com/events/98765"
}]
` + "```" + `

The input is:
=====
const textStrings = `


func Router(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
    switch req.RequestContext.HTTP.Method {
    case "POST":
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

	// start of ZR code
	client := &http.Client{}
	zrReq, zrErr := http.NewRequest("GET", "https://api.zenrows.com/v1/?apikey=" + os.Getenv("ZENROWS_API_KEY") + "&url=" + url.QueryEscape(inputPayload.Url) + "&js_render=true&wait=2500", nil)
	if zrErr != nil {
		log.Fatalln(zrErr)
	}
	zrRes, zrErr := client.Do(zrReq)
	if zrErr != nil {
			log.Fatalln(zrErr)
	}
	defer zrRes.Body.Close()

	zrBody, zrErr := io.ReadAll(zrRes.Body)
	if zrErr != nil {
			log.Fatalln(zrErr)
	}

	zrBodyString := string(zrBody)
	// avoid extra parsing work for <head> content outside of <body>
	re := regexp.MustCompile(`<body>(.*?)<\/body>`)
	matches := re.FindStringSubmatch(zrBodyString)
	if len(matches) > 1 {
		zrBodyString = matches[1]
	}
	markdown, err := converter.ConvertString(zrBodyString)
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(markdown, "\n")

	// Filter out empty lines
	var nonEmptyLines []string
	for _, line := range lines {
			if line != "" {
					nonEmptyLines = append(nonEmptyLines, line)
			}
	}

	// Convert to JSON
	jsonStringBytes, err := json.Marshal(nonEmptyLines)
	if err != nil {
			fmt.Println("Error converting to JSON:", err)
	}
	jsonString := string(jsonStringBytes)

	// TODO: consider log levels / log volume
	sessionID, messageContent, err := CreateChatSession(jsonString)
	if err != nil {
		fmt.Println("Error creating chat session:", err)
		return events.LambdaFunctionURLResponse{}, err
	}

	var eventsMap []map[string]string

	msgsErr := json.Unmarshal([]byte(messageContent), &eventsMap)
	if err != nil {
			fmt.Println("Invalid JSON response from OpenAI:", msgsErr)
	}

	openAIjson, err := parseJSONString(messageContent)
	if err != nil {
			fmt.Println("Error parsing JSON response from OpenAI message:", err)
	}

	fmt.Println("Chat GPT response: ", sessionID)
	fmt.Println("Chat GPT events data: ", eventsMap)
	fmt.Println("Chat GPT message content: ", messageContent)

	json, err := json.Marshal(SeshuResponseBody{sessionID, openAIjson})

	if err != nil {
			fmt.Println("Error parsing response body as JSON:", err)
	}

	// TODO: IMPORTANT: for meetup.com we need to parse out the data located in
	// <script type="application/ld+json">...</script> and use that to grab
	// `lat` / `lng` and location name ... details here:
	// https://discord.com/channels/1096208627192320030/1151217204126290011/1226233470528000093

	return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusCreated,
			Body: string(json),
			Headers: map[string]string{
					"Location": fmt.Sprintf("/user/%s", "hello res"),
			},
	}, nil
}

func parseJSONString(jsonString string) (interface{}, error) {
	// Remove whitespace and newlines
	jsonString = strings.ReplaceAll(jsonString, " ", "")
	jsonString = strings.ReplaceAll(jsonString, "\n", "")

	// Parse the JSON string
	var result interface{}
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateChatSession(markdownLinesAsArr string) (string, string, error) {
	client := &http.Client{}
	fmt.Println("Creating chat session")
	fmt.Println("systemPrompt: ", systemPrompt)
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

	fmt.Println("Chat GPT response: ", respData)

	sessionId := respData.ID
	if sessionId == "" {
		return "", "", fmt.Errorf("unexpected response format, `id` missing")
	}

	messageContentArray := respData.Choices[0].Message.Content
	if messageContentArray == "" {
		return "", "", fmt.Errorf("unexpected response format, `message.content` missing")
	}

	messageContentArrayBytes, err := json.Marshal(messageContentArray)
	if err != nil {
			return "", "", err
	}

	messageContentArrayJSON := string(messageContentArrayBytes)

	if messageContentArrayJSON == "" {
		return "", "", fmt.Errorf("failed convert message content to JSON")
	}

	return sessionId, messageContentArrayJSON, nil
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
