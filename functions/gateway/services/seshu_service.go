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

	"github.com/araddon/dateparse"
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

// Check if URL is a Facebook events page
func IsFacebookEventsURL(targetURL string) bool {
	// Pattern: facebook.com/<any string>/events
	pattern := regexp.MustCompile(`facebook\.com\/[^\/]+\/events`)
	return pattern.MatchString(targetURL)
}

// ParseFlexibleDatetime attempts to parse a date string using various formats and timezone detection
// Returns unix timestamp and error. If the string doesn't look date-like, returns an error.
func ParseFlexibleDatetime(dateStr string, fallbackTimezone *time.Location) (int64, error) {

	if dateStr == "" {
		return 0, fmt.Errorf("empty date string")
	}

	// Set fallback timezone if not provided
	if fallbackTimezone == nil {
		fallbackTimezone = time.UTC
	} else {
	}

	// Try to parse with dateparse library which handles many formats and validation
	parsedTime, err := dateparse.ParseAny(dateStr)
	if err != nil {
		// If dateparse fails, try some Facebook-specific patterns
		return tryFacebookDateFormats(dateStr, fallbackTimezone)
	}

	// If dateparse succeeded but the time has no timezone info (Local),
	// assume it's in the fallback timezone
	if parsedTime.Location() == time.Local {
		// Re-interpret the time components in the fallback timezone
		adjustedTime := time.Date(
			parsedTime.Year(),
			parsedTime.Month(),
			parsedTime.Day(),
			parsedTime.Hour(),
			parsedTime.Minute(),
			parsedTime.Second(),
			parsedTime.Nanosecond(),
			fallbackTimezone,
		)
		return adjustedTime.Unix(), nil
	}

	return parsedTime.Unix(), nil
}

// tryFacebookDateFormats attempts to parse Facebook-specific date formats
func tryFacebookDateFormats(dateStr string, fallbackTimezone *time.Location) (int64, error) {

	// Facebook formats we've seen:
	// "Sat, Jul 26 at 3:00 PM CDT"
	// "Fri, Jul 25 - Jul 26"
	// "Saturday, July 26 at 3:00 PM"

	// Handle timezone abbreviations that Go doesn't recognize by default
	// Note: Some abbreviations are ambiguous (e.g., CST can mean Central/China/Cuba Standard Time)
	// This mapping prioritizes the most commonly used interpretation
	timezoneMap := map[string]string{
		// North America - United States
		"CDT":  "America/Chicago",     // Central Daylight Time
		"CST":  "America/Chicago",     // Central Standard Time (US - most common usage)
		"EDT":  "America/New_York",    // Eastern Daylight Time
		"EST":  "America/New_York",    // Eastern Standard Time
		"PDT":  "America/Los_Angeles", // Pacific Daylight Time
		"PST":  "America/Los_Angeles", // Pacific Standard Time
		"MDT":  "America/Denver",      // Mountain Daylight Time
		"MST":  "America/Denver",      // Mountain Standard Time
		"AKDT": "America/Anchorage",   // Alaska Daylight Time
		"AKST": "America/Anchorage",   // Alaska Standard Time
		"HDT":  "America/Adak",        // Hawaii-Aleutian Daylight Time
		"HST":  "Pacific/Honolulu",    // Hawaii Standard Time (most common usage)

		// North America - Canada
		"ADT": "America/Halifax",  // Atlantic Daylight Time
		"AST": "America/Halifax",  // Atlantic Standard Time (most common usage)
		"NDT": "America/St_Johns", // Newfoundland Daylight Time
		"NST": "America/St_Johns", // Newfoundland Standard Time

		// Europe
		"CET":  "Europe/Paris",  // Central European Time
		"CEST": "Europe/Paris",  // Central European Summer Time
		"EET":  "Europe/Athens", // Eastern European Time
		"EEST": "Europe/Athens", // Eastern European Summer Time
		"WET":  "Europe/Lisbon", // Western European Time
		"WEST": "Europe/Lisbon", // Western European Summer Time
		"GMT":  "Europe/London", // Greenwich Mean Time
		"BST":  "Europe/London", // British Summer Time (most common usage)
		"IST":  "Asia/Kolkata",  // India Standard Time (most common usage - large population)
		"MSK":  "Europe/Moscow", // Moscow Standard Time

		// Asia - China/East Asia
		"JST": "Asia/Tokyo",     // Japan Standard Time
		"KST": "Asia/Seoul",     // Korea Standard Time
		"HKT": "Asia/Hong_Kong", // Hong Kong Time
		"CTT": "Asia/Shanghai",  // China Standard Time (using CTT to avoid CST conflict)

		// Asia - South/Southeast Asia
		"PKT":  "Asia/Karachi",      // Pakistan Standard Time
		"NPT":  "Asia/Kathmandu",    // Nepal Time
		"ICT":  "Asia/Bangkok",      // Indochina Time
		"WIB":  "Asia/Jakarta",      // Western Indonesian Time
		"WITA": "Asia/Makassar",     // Central Indonesian Time
		"WIT":  "Asia/Jayapura",     // Eastern Indonesian Time
		"PHT":  "Asia/Manila",       // Philippine Time
		"SGT":  "Asia/Singapore",    // Singapore Time
		"MYT":  "Asia/Kuala_Lumpur", // Malaysia Time

		// Asia - Middle East
		"GST":  "Asia/Dubai",  // Gulf Standard Time
		"IRST": "Asia/Tehran", // Iran Standard Time
		"IRDT": "Asia/Tehran", // Iran Daylight Time

		// Asia - Central Asia
		"UZT":  "Asia/Tashkent", // Uzbekistan Time
		"TMT":  "Asia/Ashgabat", // Turkmenistan Time
		"TJT":  "Asia/Dushanbe", // Tajikistan Time
		"KGT":  "Asia/Bishkek",  // Kyrgyzstan Time
		"ALMT": "Asia/Almaty",   // Alma-Ata Time

		// Asia - Russia
		"YEKT": "Asia/Yekaterinburg", // Yekaterinburg Time
		"OMST": "Asia/Omsk",          // Omsk Time
		"KRAT": "Asia/Krasnoyarsk",   // Krasnoyarsk Time
		"IRKT": "Asia/Irkutsk",       // Irkutsk Time
		"YAKT": "Asia/Yakutsk",       // Yakutsk Time
		"VLAT": "Asia/Vladivostok",   // Vladivostok Time
		"MAGT": "Asia/Magadan",       // Magadan Time
		"PETT": "Asia/Kamchatka",     // Kamchatka Time
		"ANAT": "Asia/Anadyr",        // Anadyr Time

		// Africa
		"CAT":  "Africa/Maputo",       // Central Africa Time
		"EAT":  "Africa/Nairobi",      // East Africa Time
		"WAT":  "Africa/Lagos",        // West Africa Time
		"SAST": "Africa/Johannesburg", // South African Standard Time

		// Australia/Oceania
		"AEST": "Australia/Sydney",   // Australian Eastern Standard Time
		"AEDT": "Australia/Sydney",   // Australian Eastern Daylight Time
		"ACST": "Australia/Adelaide", // Australian Central Standard Time
		"ACDT": "Australia/Adelaide", // Australian Central Daylight Time
		"AWST": "Australia/Perth",    // Australian Western Standard Time
		"NZST": "Pacific/Auckland",   // New Zealand Standard Time
		"NZDT": "Pacific/Auckland",   // New Zealand Daylight Time
		"CHST": "Pacific/Guam",       // Chamorro Standard Time

		// South America
		"ART":  "America/Argentina/Buenos_Aires", // Argentina Time
		"BRT":  "America/Sao_Paulo",              // Brasília Time
		"BRST": "America/Sao_Paulo",              // Brasília Summer Time
		"CLT":  "America/Santiago",               // Chile Standard Time
		"CLST": "America/Santiago",               // Chile Summer Time
		"COT":  "America/Bogota",                 // Colombia Time
		"PET":  "America/Lima",                   // Peru Time
		"VET":  "America/Caracas",                // Venezuelan Standard Time
		"UYT":  "America/Montevideo",             // Uruguay Standard Time
		"UYST": "America/Montevideo",             // Uruguay Summer Time
		"PYT":  "America/Asuncion",               // Paraguay Time
		"PYST": "America/Asuncion",               // Paraguay Summer Time
		"BOT":  "America/La_Paz",                 // Bolivia Time
		"ECT":  "America/Guayaquil",              // Ecuador Time
		"GYT":  "America/Guyana",                 // Guyana Time
		"SRT":  "America/Paramaribo",             // Suriname Time
		"GFT":  "America/Cayenne",                // French Guiana Time

		// Additional common abbreviations
		"UTC":  "UTC",                // Coordinated Universal Time
		"CADT": "Australia/Adelaide", // Central Australia Daylight Time (deprecated)
		"CAST": "Australia/Adelaide", // Central Australia Standard Time (deprecated)
		"EAST": "Australia/Sydney",   // Eastern Australia Standard Time (deprecated)
		"EADT": "Australia/Sydney",   // Eastern Australia Daylight Time (deprecated)
		"NZT":  "Pacific/Auckland",   // New Zealand Time (deprecated)
	}

	// Build regex pattern dynamically from all timezone abbreviations in the map
	var tzAbbreviations []string
	for abbr := range timezoneMap {
		tzAbbreviations = append(tzAbbreviations, abbr)
	}
	tzPatternStr := `\b(` + strings.Join(tzAbbreviations, "|") + `)\b`
	tzPattern := regexp.MustCompile(tzPatternStr)

	// Try to extract timezone from the string
	tzMatch := tzPattern.FindString(dateStr)

	targetTimezone := fallbackTimezone
	if tzMatch != "" {
		if tzName, exists := timezoneMap[tzMatch]; exists {
			if loc, err := time.LoadLocation(tzName); err == nil {
				targetTimezone = loc
			}
		}
		// Remove timezone from string for parsing
		dateStr = tzPattern.ReplaceAllString(dateStr, "")
	}

	// Current year for dates that don't specify year
	currentYear := time.Now().Year()

	cleanedDateStr := strings.TrimSpace(dateStr)

	// Handle date ranges by splitting on '-' or '|' and keeping only the left side
	// Examples: "Fri, Jul 25 - Jul 26" -> "Fri, Jul 25"
	if strings.Contains(cleanedDateStr, " - ") {
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, " - ")[0])
	} else if strings.Contains(cleanedDateStr, " | ") {
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, " | ")[0])
	}

	// Try various Facebook-style formats
	formats := []string{
		"Mon, Jan 2 at 3:04 PM",        // "Sat, Jul 26 at 3:00 PM"
		"Monday, January 2 at 3:04 PM", // "Saturday, July 26 at 3:00 PM"
		"Mon, Jan 2",                   // "Sat, Jul 26"
		"Monday, January 2",            // "Saturday, July 26"
		"Jan 2 at 3:04 PM",             // "Jul 26 at 3:00 PM"
		"January 2 at 3:04 PM",         // "July 26 at 3:00 PM"
		// New formats for day-month order and 24-hour time
		"Mon, 2 Jan at 15:04",        // "Sat, 26 Jul at 15:00"
		"Monday, 2 January at 15:04", // "Saturday, 26 July at 15:00"
		"Mon, 2 Jan",                 // "Sat, 26 Jul"
		"Monday, 2 January",          // "Saturday, 26 January"
		"2 Jan at 15:04",             // "26 Jul at 15:00"
		"2 January at 15:04",         // "26 July at 15:00"
	}

	for _, format := range formats {
		if parsedTime, err := time.ParseInLocation(format, cleanedDateStr, targetTimezone); err == nil {
			// If no year was parsed, assume current year
			if parsedTime.Year() == 0 {
				parsedTime = parsedTime.AddDate(currentYear, 0, 0)
			}
			return parsedTime.Unix(), nil
		}
	}

	return 0, fmt.Errorf("unable to parse date format: %s", dateStr)
}

// FindFacebookEventData extracts event data from Facebook events pages specifically
func FindFacebookEventData(htmlContent string) ([]internal_types.EventInfo, error) {

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
	events := ExtractEventsFromJSON(eventScriptContent)

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}

// ExtractEventsFromJSON extracts events from JSON content - exported for use as callback
func ExtractEventsFromJSON(jsonContent string) []internal_types.EventInfo {

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

			// Only add events with required fields
			if title != "" && url != "" {
				fmt.Printf("DEBUG: Event %d - Title: %s, URL: %s, Date: %s, Location: %s, Organizer: %s\n",
					eventID, title, url, date, location, organizer)

				// Try to parse the date with flexible parsing
				// Default to Central Time for Facebook events (most US events)
				centralTZ, _ := time.LoadLocation("America/Chicago")
				_, dateErr := ParseFlexibleDatetime(date, centralTZ)

				// Only include events where we can parse the date successfully
				// This validates that the date string contains actual date information
				if dateErr != nil {
					continue
				}

				events = append(events, internal_types.EventInfo{
					EventTitle:       title,
					EventLocation:    location,
					EventStartTime:   date, // Keep raw date for display
					EventEndTime:     date, // Keep raw date for display
					EventURL:         url,
					EventDescription: "",
					EventSource:      "facebook",
					// TODO: Add parsed timestamp field to EventInfo struct for proper time handling
				})
				eventID++
			}
		} else {
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
