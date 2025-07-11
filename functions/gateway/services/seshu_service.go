package services

import (
	"context"
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

	log.Printf("Updated seshu session: %+v", seshuPayload.Url)

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

// findEventData extracts event data from Facebook events pages
func FindEventData(htmlContent string) ([]internal_types.EventInfo, error) {
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

	// Log the JSON for debugging
	fmt.Printf("DEBUG: Found event script content (first 1000 chars):\n%s\n", eventScriptContent[:min(1000, len(eventScriptContent))])

	// Extract events from the JSON content
	events := extractEventsFromJSON(eventScriptContent)

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}

// extractEventsFromJSON extracts events from JSON content
func extractEventsFromJSON(jsonContent string) []internal_types.EventInfo {
	var events []internal_types.EventInfo
	eventID := 1

	// Look for event objects in the JSON
	eventPattern := regexp.MustCompile(`"__typename":"Event"[^}]*?"name":"([^"]+)"[^}]*?"url":"([^"]+)"`)
	eventMatches := eventPattern.FindAllStringSubmatch(jsonContent, -1)

	for _, match := range eventMatches {
		if len(match) >= 3 {
			title := cleanUnicodeText(match[1])
			url := unescapeJSON(match[2]) // Unescape the URL

			// Extract date pattern
			datePattern := regexp.MustCompile(`"day_time_sentence":"([^"]+)"`)
			dateMatches := datePattern.FindStringSubmatch(jsonContent)
			var date string
			if len(dateMatches) >= 2 {
				date = cleanUnicodeText(unescapeJSON(dateMatches[1])) // Unescape date
			}

			// Extract location pattern
			locationPattern := regexp.MustCompile(`"contextual_name":"([^"]+)"`)
			locationMatches := locationPattern.FindStringSubmatch(jsonContent)
			var location string
			if len(locationMatches) >= 2 {
				location = cleanUnicodeText(unescapeJSON(locationMatches[1])) // Unescape location
			}

			// Extract organizer pattern
			organizerPattern := regexp.MustCompile(`"event_creator"[^}]*?"name":"([^"]+)"`)
			organizerMatches := organizerPattern.FindStringSubmatch(jsonContent)
			var organizer string
			if len(organizerMatches) >= 2 {
				organizer = cleanUnicodeText(unescapeJSON(organizerMatches[1])) // Unescape organizer
			}

			fmt.Printf("DEBUG: Event %d - Title: %s, URL: %s, Date: %s, Location: %s, Organizer: %s\n",
				eventID, title, url, date, location, organizer)

			// Only add events with required fields
			if title != "" && url != "" {
				// events = append(events, EventFb{
				// 	ID:        eventID,
				// 	Date:      date,
				// 	Title:     title,
				// 	URL:       url,
				// 	Location:  location,
				// 	Organizer: organizer,
				// })

				// TODO: can golang `time` package handle loose dates like
				// this one `Fri, Jul 25 - Jul 26`

				dateTime, err := time.Parse("Mon, Jan 2, 2006 15:04", date)
				if err != nil {
					continue
				}

				log.Printf("DEBUG: Parsed date: %s", dateTime)

				events = append(events, internal_types.EventInfo{
					EventTitle:       title,
					EventLocation:    location,
					EventStartTime:   date,
					EventEndTime:     date,
					EventURL:         url,
					EventDescription: "",
					EventSource:      "facebook",
				})
				eventID++
			}
		}
	}

	return events
}

// unescapeJSON unescapes JSON-encoded strings (removes \\/ and other escape sequences)
func unescapeJSON(s string) string {
	// Replace escaped forward slashes
	s = strings.ReplaceAll(s, `\/`, `/`)
	// Replace escaped backslashes
	s = strings.ReplaceAll(s, `\\`, `\`)
	// Replace escaped quotes
	s = strings.ReplaceAll(s, `\"`, `"`)
	// Replace escaped newlines
	s = strings.ReplaceAll(s, `\n`, "\n")
	// Replace escaped tabs
	s = strings.ReplaceAll(s, `\t`, "\t")
	// Replace Unicode escape sequences
	s = strings.ReplaceAll(s, `\u202f`, " ") // Narrow no-break space
	s = strings.ReplaceAll(s, `\u00a0`, " ") // Non-breaking space
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
