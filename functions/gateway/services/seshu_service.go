package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/helpers"

	"github.com/itlightning/dateparse"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var seshuSessionsTableName = helpers.GetDbTableName(helpers.SeshuSessionTablePrefix)

const FakeCity = "Nowhere City, NM 11111"
const FakeUrl1 = "http://example.com/event/12345"
const FakeUrl2 = "http://example.com/event/98765"
const FakeEventTitle1 = "Fake Event Title 1"
const FakeEventTitle2 = "Fake Event Title 2"
const FakeStartTime1 = "Sep 26, 26:30pm"
const FakeStartTime2 = "Oct 10, 25:00am"
const FakeEndTime1 = "Sep 26, 27:30pm"
const FakeEndTime2 = "Oct 10, 26:00am"

const InitialEmptyLatLong = 9e+10

type ChildEventMeta struct {
	EventTitle       string
	EventURL         string
	EventDescription string
	EventLocation    string
	EventStartTime   string
	EventHostName    string
}

func init() {
	seshuSessionsTableName = helpers.GetDbTableName(helpers.SeshuSessionTablePrefix)
}

func GetSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionGet) (*internal_types.SeshuSession, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Key: map[string]types.AttributeValue{
			"url": &types.AttributeValueMemberS{Value: seshuPayload.Url},
		},
	}

	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

	var seshuSession internal_types.SeshuSession
	err = attributevalue.UnmarshalMap(result.Item, &seshuSession)
	if err != nil {
		return nil, err
	}

	return &seshuSession, nil
}

func InsertSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionInput) (*internal_types.SeshuSessionInsert, error) {
	currentTime := time.Now().Unix()
	if len(seshuPayload.EventCandidates) < 1 {
		seshuPayload.EventCandidates = []internal_types.EventInfo{}
	}
	if len(seshuPayload.EventValidations) < 1 {
		seshuPayload.EventValidations = []internal_types.EventBoolValid{}
	}
	newSeshuSession := internal_types.SeshuSessionInsert{
		OwnerId:           seshuPayload.OwnerId,
		Url:               seshuPayload.Url,
		UrlDomain:         seshuPayload.UrlDomain,
		UrlPath:           seshuPayload.UrlPath,
		UrlQueryParams:    seshuPayload.UrlQueryParams,
		LocationLatitude:  seshuPayload.LocationLatitude,
		LocationLongitude: seshuPayload.LocationLongitude,
		LocationAddress:   seshuPayload.LocationAddress,
		Html:              seshuPayload.Html,
		ChildId:           seshuPayload.ChildId,
		EventCandidates:   seshuPayload.EventCandidates,
		// TODO: this needs to become a map to avoid regressions pertaining
		// to key ordering in the future
		EventValidations: seshuPayload.EventValidations,
		Status:           "draft",
		ExpireAt:         currentTime + 3600*24, // 24 hrs expiration
		CreatedAt:        currentTime,
		UpdatedAt:        currentTime,
	}

	item, err := attributevalue.MarshalMap(newSeshuSession)
	if err != nil {
		return nil, err
	}

	if seshuSessionsTableName == "" {
		return nil, fmt.Errorf("ERR: seshuSessionsTableName is empty")
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Item:      item,
	}

	// TODO: Before this db PUT, check for existing seshu job
	// key via query to "jobs" table... if it exists, then return a
	// 409 error to the client, explaining it can't be added because
	// that URL is already owned by someone else

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &newSeshuSession)
	if err != nil {
		return nil, err
	}

	// TODO: omit newSeshuSession.Html from response, it's too large
	return &newSeshuSession, nil
}

func UpdateSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionUpdate) (*internal_types.SeshuSessionUpdate, error) {

	// TODO: DB call to check if it exists first, and the the owner is the same as the one updating

	if seshuSessionsTableName == "" {
		return nil, fmt.Errorf("ERR: seshuSessionsTableName is empty")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(seshuSessionsTableName),
		Key: map[string]types.AttributeValue{
			"url": &types.AttributeValueMemberS{Value: seshuPayload.Url},
		},
		ExpressionAttributeNames:  make(map[string]string),
		ExpressionAttributeValues: make(map[string]types.AttributeValue),
		UpdateExpression:          aws.String("SET"),
	}

	if seshuPayload.OwnerId != "" {
		input.ExpressionAttributeNames["#ownerId"] = "ownerId"
		input.ExpressionAttributeValues[":ownerId"] = &types.AttributeValueMemberS{Value: seshuPayload.OwnerId}
		*input.UpdateExpression += " #ownerId = :ownerId,"
	}

	if seshuPayload.UrlDomain != "" {
		input.ExpressionAttributeNames["#urlDomain"] = "urlDomain"
		input.ExpressionAttributeValues[":urlDomain"] = &types.AttributeValueMemberS{Value: seshuPayload.UrlDomain}
		*input.UpdateExpression += " #urlDomain = :urlDomain,"
	}

	if seshuPayload.UrlPath != "" {
		input.ExpressionAttributeNames["#urlPath"] = "urlPath"
		input.ExpressionAttributeValues[":urlPath"] = &types.AttributeValueMemberS{Value: seshuPayload.UrlPath}
		*input.UpdateExpression += " #urlPath = :urlPath,"
	}

	if len(seshuPayload.UrlQueryParams) > 0 {
		input.ExpressionAttributeNames["#urlQueryParams"] = "urlQueryParams"
		queryParams, err := attributevalue.MarshalMap(seshuPayload.UrlQueryParams)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":urlQueryParams"] = &types.AttributeValueMemberM{Value: queryParams}
		*input.UpdateExpression += " #urlQueryParams = :urlQueryParams,"
	}

	if seshuPayload.LocationLatitude != InitialEmptyLatLong {
		input.ExpressionAttributeNames["#locationLatitude"] = "locationLatitude"
		input.ExpressionAttributeValues[":locationLatitude"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(seshuPayload.LocationLatitude, 'f', -1, 64)}
		*input.UpdateExpression += " #locationLatitude = :locationLatitude,"
	}

	if seshuPayload.LocationLongitude != InitialEmptyLatLong {
		input.ExpressionAttributeNames["#locationLongitude"] = "locationLongitude"
		input.ExpressionAttributeValues[":locationLongitude"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(seshuPayload.LocationLongitude, 'f', -1, 64)}
		*input.UpdateExpression += " #locationLongitude = :locationLongitude,"
	}

	if seshuPayload.LocationAddress != "" {
		input.ExpressionAttributeNames["#locationAddress"] = "locationAddress"
		input.ExpressionAttributeValues[":locationAddress"] = &types.AttributeValueMemberS{Value: seshuPayload.LocationAddress}
		*input.UpdateExpression += " #locationAddress = :locationAddress,"
	}

	if seshuPayload.Html != "" {
		input.ExpressionAttributeNames["#html"] = "html"
		input.ExpressionAttributeValues[":html"] = &types.AttributeValueMemberS{Value: seshuPayload.Html}
		*input.UpdateExpression += " #html = :html,"
	}

	if seshuPayload.ChildId != "" {
		input.ExpressionAttributeNames["#childId"] = "childId"
		input.ExpressionAttributeValues[":childId"] = &types.AttributeValueMemberS{Value: seshuPayload.ChildId}
		*input.UpdateExpression += " #childId = :childId,"
	}

	if seshuPayload.EventCandidates != nil {
		input.ExpressionAttributeNames["#eventCandidates"] = "eventCandidates"
		eventCandidates, err := attributevalue.MarshalList(seshuPayload.EventCandidates)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":eventCandidates"] = &types.AttributeValueMemberL{Value: eventCandidates}
		*input.UpdateExpression += " #eventCandidates = :eventCandidates,"
	}

	if seshuPayload.EventValidations != nil {
		input.ExpressionAttributeNames["#eventValidations"] = "eventValidations"
		eventValidations, err := attributevalue.MarshalList(seshuPayload.EventValidations)
		if err != nil {
			return nil, err
		}
		input.ExpressionAttributeValues[":eventValidations"] = &types.AttributeValueMemberL{Value: eventValidations}
		*input.UpdateExpression += " #eventValidations = :eventValidations,"
	}

	if seshuPayload.Status != "" {
		input.ExpressionAttributeNames["#status"] = "status"
		input.ExpressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: seshuPayload.Status}
		*input.UpdateExpression += " #status = :status,"
	}

	currentTime := time.Now().Unix()
	input.ExpressionAttributeNames["#updatedAt"] = "updatedAt"
	input.ExpressionAttributeValues[":updatedAt"] = &types.AttributeValueMemberN{Value: strconv.FormatFloat(float64(currentTime), 'f', -1, 64)}
	*input.UpdateExpression += " #updatedAt = :updatedAt"

	_, err := db.UpdateItem(ctx, input)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// TODO: `CommitSeshuSession` needs to consider:
// 1. Validate the session
// 2. Update the status to `committed`
// 3. If the "are these in the same location?" checkbox in UI is not checked,
//    then `latitude` `longitude` and `address` should be ignored from the OpenAI
//    structured response
// 4. Iterate over the union array of `EventCandidates` and `EventValidations` to
//    create a new array that removes any `EventCandidates` that lack any of:
//    `event_title`, `event_location`, `event_start_time` which are all required
// 5. Use that reduced array to find the corresponding strings in the stored
//    `SeshuSession.Html` in the db
// 6. Store the deduced DOM query strings in the new "Scraping Jobs" db table we've
//    not created yet
// 7. The input URL should have it's query params algorithmically sorted to prevent
//    db index (the URL itself is the index) collision / duplicates
// 8. Delete the session from the `SeshuSessions` table once it's confirmed to be
//    successfully committed to the "Scraping Jobs" table

// cleanUnicodeText removes problematic Unicode characters while preserving readable text
func cleanUnicodeText(text string) string {
	// Remove specific problematic Unicode characters
	text = strings.ReplaceAll(text, string([]byte{226, 128, 175}), "") // Remove bytes 226 128 175
	text = strings.ReplaceAll(text, "\u202f", " ")                     // Replace narrow no-break space with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")                     // Replace non-breaking space with regular space

	// Remove other non-printable characters but preserve middle dot and normal punctuation
	var result strings.Builder
	for _, r := range text {
		if unicode.IsPrint(r) || r == '·' || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Clean up extra whitespace
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(result.String(), " "))
}

// Check if URL is a Facebook events page
func IsFacebookEventsURL(targetURL string) bool {
	// Pattern: facebook.com/<any string>/events
	// pattern := regexp.MustCompile(`facebook\.com\/[^\/]+\/events`)
	// also allow for https://www.facebook.com/people/<user>/<user_id>/?sk=events
	pattern := regexp.MustCompile(`facebook\.com\/[^\/]+\/events|facebook\.com\/people\/(.*)sk=events`)
	return pattern.MatchString(targetURL)
}

// ParseFlexibleDatetime attempts to parse a date string using various formats and timezone detection
// Returns unix timestamp and error. If the string doesn't look date-like, returns an error.
func ParseFlexibleDatetime(dateStr string, fallbackTimezone *time.Location) (dateNormalized string, tz string, err error) {

	if dateStr == "" {
		return "", "", fmt.Errorf("empty date string")
	}

	// Set fallback timezone if not provided
	if fallbackTimezone == nil {
		// REMOVED: No more default timezone fallback
		// Events must have a valid timezone from the scraped data
		return "", "", fmt.Errorf("no fallback timezone provided and no timezone found in date string")
	}

	// Try to parse with dateparse library which handles many formats and validation
	parsedTime, err := dateparse.ParseAny(dateStr)
	if err != nil {
		// If dateparse fails, try some Facebook-specific patterns
		result, err := ParseMaybeMultiDayEvent(dateStr)
		if err != nil {
			return "", "", err
		}

		return result, "", nil
	}

	// If dateparse succeeded but the time has no timezone info (Local),
	// we can't proceed without a valid timezone
	if parsedTime.Location() == time.Local {
		return "", "", fmt.Errorf("parsed time has no timezone info and no fallback provided")
	}

	return parsedTime.String(), parsedTime.Location().String(), nil
}

// extractChildEventMeta extracts all event metadata from HTML content
// Returns error if the number of captured items is not equal across all fields
func extractChildEventMeta(htmlContent string) (ChildEventMeta, error) {

	titles, err := extractJSONField(htmlContent, `"og:title" content="((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event title: %w", err)
	}

	urls, err := extractJSONField(htmlContent, `__typename":"Event".*?"url":"((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event URLs: %w", err)
	}

	// Extract event descriptions
	descriptions, err := extractJSONField(htmlContent, `"event_description":\{.*?"text":"((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event descriptions: %w", err)
	}

	// Extract event locations (one_line_address)
	locations, err := extractJSONField(htmlContent, `"one_line_address":"((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event locations: %w", err)
	}

	// Extract event start times (placeholder for now - will be implemented next)
	startTimes, err := extractJSONField(htmlContent, `"day_time_sentence":"((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event start times: %w", err)
	}

	// Extract event host names
	hostNames, err := extractJSONField(htmlContent, `"__typename":"User".*?"name":"((?:[^"\\]|\\.)*)"`)
	if err != nil {
		return ChildEventMeta{}, fmt.Errorf("failed to extract event host names: %w", err)
	}

	if len(titles) < 1 || len(urls) < 1 || len(descriptions) < 1 || len(locations) < 1 || len(startTimes) < 1 || len(hostNames) < 1 {
		return ChildEventMeta{}, fmt.Errorf("mismatched array lengths: titles=%d, urls=%d, descriptions=%d, locations=%d, startTimes=%d, hostNames=%d",
			len(titles), len(urls), len(descriptions), len(locations), len(startTimes), len(hostNames))
	}

	// Build the result array
	var result ChildEventMeta = ChildEventMeta{
		EventTitle:       titles[0],
		EventURL:         urls[0],
		EventDescription: descriptions[0],
		EventLocation:    locations[0],
		EventStartTime:   startTimes[0],
		EventHostName:    hostNames[0],
	}

	return result, nil
}

// extractJSONField is a helper function that extracts JSON field values using regex
// and handles JSON string unescaping
func extractJSONField(htmlContent string, pattern string) ([]string, error) {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	var results []string
	for _, match := range matches {
		if len(match) >= 2 {
			// Use Go's built-in JSON unmarshaling to handle all escape sequences
			var value string
			if err := json.Unmarshal([]byte(`"`+match[1]+`"`), &value); err == nil {
				results = append(results, value)
			} else {
				// Fallback to raw string if JSON unmarshaling fails
				results = append(results, match[1])
			}
		}
	}

	return results, nil
}

// FindFacebookEventData extracts event data from Facebook events pages specifically
func FindFacebookEventData(htmlContent string, locationTimezone string) ([]internal_types.EventInfo, error) {
	// log.Printf("=======\n\n\n\n\n (562) HTML \n: %s", htmlContent)
	// First, check if the HTML contains event data
	if !strings.Contains(htmlContent, `"__typename":"Event"`) {
		return nil, fmt.Errorf("no Facebook event data found (no __typename: Event)")
	}

	// Find script tags containing event data
	scriptPattern := regexp.MustCompile(`<script[^>]*>(.*?)</script>`)
	scriptMatches := scriptPattern.FindAllStringSubmatch(htmlContent, -1)

	var eventScriptContent string
	for _, match := range scriptMatches {
		if len(match) >= 2 {
			scriptContent := match[1]
			if strings.Contains(scriptContent, `"__typename":"Event"`) {
				eventScriptContent = scriptContent
				break
			}
		}
	}

	if eventScriptContent == "" {
		return nil, fmt.Errorf("no script tag found containing event data")
	}

	// Clean up the script content (remove extra whitespace)
	eventScriptContent = strings.TrimSpace(eventScriptContent)

	// Extract events from the JSON content
	events := ExtractFbEventsFromJSON(eventScriptContent, locationTimezone)

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}

// ExtractFbEventsFromJSON extracts events from JSON content - exported for use as callback
func ExtractFbEventsFromJSON(jsonContent string, locationTimezone string) []internal_types.EventInfo {
	var events []internal_types.EventInfo

	// Try to parse the entire JSON content
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &jsonData)
	if err != nil {
		log.Printf("Failed to parse JSON content: %v", err)
	}

	// Extract events from the parsed JSON structure
	events = findEventsInJSON(jsonData, events, locationTimezone)

	// If no events found in structured JSON, fall back to regex
	if len(events) == 0 {
		log.Printf("No events found: %v", err)
	}

	return events
}

// findEventsInJSON recursively searches for event objects in JSON data
func findEventsInJSON(data interface{}, events []internal_types.EventInfo, locationTimezone string) []internal_types.EventInfo {
	switch v := data.(type) {
	case map[string]interface{}:
		// Check if this is an event object
		if typename, ok := v["__typename"].(string); ok && typename == "Event" {
			event := extractEventFromObject(v)
			if event.EventTitle != "" && event.EventURL != "" {
				events = append(events, event)
			}
		}

		// Recursively search in all values
		for _, value := range v {
			events = findEventsInJSON(value, events, locationTimezone)
		}

	case []interface{}:
		// Recursively search in array elements
		for _, item := range v {
			events = findEventsInJSON(item, events, locationTimezone)
		}
	}

	return events
}

// extractEventFromObject extracts event data from a single event object
func extractEventFromObject(eventObj map[string]interface{}) internal_types.EventInfo {
	event := internal_types.EventInfo{}
	// Extract title
	if name, ok := eventObj["name"].(string); ok {
		event.EventTitle = cleanUnicodeText(name)
	}

	// URL candidates (prefer more specific if available later)
	var urlCandidates []string
	if url, ok := eventObj["url"].(string); ok {
		urlCandidates = append(urlCandidates, unescapeJSONString(url))
	}

	// Date string
	dateStr := ""
	if dateTime, ok := eventObj["day_time_sentence"].(string); ok {
		// Parse the date string using our multi-day event parser
		parsedDate, err := ParseMaybeMultiDayEvent(cleanUnicodeText(unescapeJSONString(dateTime)))
		if err != nil {
			log.Printf("Failed to parse date string: %v", err)
		} else {
			dateStr = parsedDate
		}
	}

	// Location candidates in priority order
	var locationCandidates []string
	if v, ok := eventObj["contextual_name"].(string); ok {
		locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSONString(v)))
	}
	if v, ok := eventObj["location"].(string); ok {
		locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSONString(v)))
	}
	if place, ok := eventObj["place"].(map[string]interface{}); ok {
		if name, ok := place["name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSONString(name)))
		}
	}
	if venue, ok := eventObj["venue"].(map[string]interface{}); ok {
		if name, ok := venue["name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSONString(name)))
		}
	}
	if eventPlace, ok := eventObj["event_place"].(map[string]interface{}); ok {
		if name, ok := eventPlace["contextual_name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSONString(name)))
		}
	}

	// Description
	description := ""
	if v, ok := eventObj["description"].(string); ok {
		description = cleanUnicodeText(unescapeJSONString(v))
	}

	// Organizer
	organizer := ""
	if creator, ok := eventObj["event_creator"].(map[string]interface{}); ok {
		if creatorName, ok := creator["name"].(string); ok {
			organizer = cleanUnicodeText(unescapeJSONString(creatorName))
		}
	}

	// Extract coordinates if available
	var eventLat, eventLng float64
	if place, ok := eventObj["place"].(map[string]interface{}); ok {
		if lat, ok := place["latitude"].(float64); ok {
			eventLat = lat
		}
		if lng, ok := place["longitude"].(float64); ok {
			eventLng = lng
		}
	}
	if venue, ok := eventObj["venue"].(map[string]interface{}); ok {
		if eventLat == 0 {
			if lat, ok := venue["latitude"].(float64); ok {
				eventLat = lat
			}
		}
		if eventLng == 0 {
			if lng, ok := venue["longitude"].(float64); ok {
				eventLng = lng
			}
		}
	}

	// Use normalize helper with default fallback timezone
	var defaultTimezone *time.Location
	if loc, err := time.LoadLocation("America/Chicago"); err == nil {
		defaultTimezone = loc
	} else {
		defaultTimezone = time.UTC
	}
	info, err := normalizeEventInfo(event.EventTitle, urlCandidates, dateStr, locationCandidates, organizer, description, defaultTimezone)
	if err != nil {
		return internal_types.EventInfo{}
	}

	// Set coordinates if we found them
	info.EventLatitude = eventLat
	info.EventLongitude = eventLng

	return info
}

// normalizeEventInfo centralizes validation and normalization shared by both extraction paths
func normalizeEventInfo(
	title string,
	urlCandidates []string,
	dateStr string,
	locationCandidates []string,
	organizer string,
	description string,
	fallbackTimezone *time.Location,
) (internal_types.EventInfo, error) {
	// Select URL: choose the last non-empty candidate
	url := ""
	for i := len(urlCandidates) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(urlCandidates[i]); s != "" {
			url = s
			break
		}
	}
	if title == "" || url == "" {
		return internal_types.EventInfo{}, fmt.Errorf("title or url is empty")
	}

	// Fallback timezone if needed
	if fallbackTimezone == nil {
		return internal_types.EventInfo{}, fmt.Errorf("fallback timezone is nil")
	}

	// Validate date string and get timezone
	if strings.TrimSpace(dateStr) == "" {
		return internal_types.EventInfo{}, fmt.Errorf("date string is empty")
	}

	startTimeRFC3339, err := ParseMaybeMultiDayEvent(dateStr)
	if err != nil {
		return internal_types.EventInfo{}, fmt.Errorf("failed to parse date string: %w", err)
	}

	// Choose first non-empty location candidate
	location := ""
	for _, cand := range locationCandidates {
		if s := strings.TrimSpace(cand); s != "" {
			location = s
			break
		}
	}

	// No timezone extraction from date string - will be resolved later via coordinates
	// Use parsed RFC3339 time for consistent date handling
	evt := internal_types.EventInfo{
		EventTitle:       title,
		EventLocation:    location,
		EventStartTime:   startTimeRFC3339, // Use parsed RFC3339 time
		EventURL:         url,
		EventDescription: description,
		EventTimezone:    "", // Will be resolved later via coordinates
		EventHostName:    organizer,
	}
	return evt, nil
}

// unescapeJSONString uses Go's built-in JSON unmarshaling to handle all escape sequences
func unescapeJSONString(s string) string {
	var result string
	if err := json.Unmarshal([]byte(`"`+s+`"`), &result); err == nil {
		return result
	}
	// Fallback to original string if JSON unmarshaling fails
	return s
}

// FindFacebookEventListData extracts event data from Facebook event LIST pages
// This handles the structure where multiple events are found at a higher nesting level
func FindFacebookEventListData(htmlContent string, locationTimezone string) ([]internal_types.EventInfo, error) {
	return findFacebookEventData(htmlContent, "list", locationTimezone)
}

// FindFacebookSingleEventData extracts event data from Facebook SINGLE event pages
// This handles the structure where a single event is nested under result.data.event
func FindFacebookSingleEventData(htmlContent string, locationTimezone string) ([]internal_types.EventInfo, error) {
	return findFacebookEventData(htmlContent, "single", locationTimezone)
}

// findFacebookEventData is the shared implementation that handles both modes
func findFacebookEventData(htmlContent string, mode string, locationTimezone string) ([]internal_types.EventInfo, error) {

	// First, check if the HTML contains event data
	if !strings.Contains(htmlContent, `"__typename":"Event"`) {
		return nil, fmt.Errorf("no Facebook event data found (no __typename: Event)")
	}

	// Find script tags containing event data
	scriptPattern := regexp.MustCompile(`<script[^>]*>(.*?)</script>`)
	scriptMatches := scriptPattern.FindAllStringSubmatch(htmlContent, -1)

	var eventScriptContent string
	for _, match := range scriptMatches {
		if len(match) >= 2 {
			scriptContent := match[1]
			if strings.Contains(scriptContent, `"__typename":"Event"`) {
				eventScriptContent = scriptContent
				break
			}
		}
	}

	if eventScriptContent == "" {
		return nil, fmt.Errorf("no script tag found containing event data")
	}

	// Clean up the script content (remove extra whitespace)
	eventScriptContent = strings.TrimSpace(eventScriptContent)

	// Extract events from the JSON content based on mode
	var events []internal_types.EventInfo
	if mode == "single" {
		// Extract all event metadata using regex
		childEvent, err := extractChildEventMeta(htmlContent)
		if err != nil {
			log.Printf("Failed to extract child event metadata: %v", err)
			// Continue with the rest of the function even if metadata extraction fails
		}
		tz, err := time.LoadLocation(locationTimezone)
		if err != nil {
			log.Printf("Failed to load timezone: %v", err)
			// Continue with the rest of the function even if timezone loading fails
		}
		eventInfo, err := normalizeEventInfo(childEvent.EventTitle, []string{childEvent.EventURL}, childEvent.EventStartTime, []string{childEvent.EventLocation}, childEvent.EventHostName, childEvent.EventDescription, tz)
		if err != nil {
			log.Printf("Failed to normalize child event metadata: %+v", err)
			// Continue with the rest of the function even if normalization fails
		}
		events = append(events, eventInfo)

	} else {
		// For event list pages, use the existing logic

		events = ExtractFbEventsFromJSON(eventScriptContent, locationTimezone)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}
