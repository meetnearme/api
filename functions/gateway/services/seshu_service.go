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

var TimezoneMap = map[string]string{
	// Build regex pattern for all timezone abbreviations
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

// ParseFlexibleDatetime attempts to parse a date string using various formats and timezone detection
// Returns unix timestamp and error. If the string doesn't look date-like, returns an error.
func ParseFlexibleDatetime(dateStr string, fallbackTimezone *time.Location) (dateUnix int64, tz string, err error) {

	if dateStr == "" {
		return 0, "", fmt.Errorf("empty date string")
	}

	// Set fallback timezone if not provided
	if fallbackTimezone == nil {
		// REMOVED: No more default timezone fallback
		// Events must have a valid timezone from the scraped data
		return 0, "", fmt.Errorf("no fallback timezone provided and no timezone found in date string")
	}

	// Try to parse with dateparse library which handles many formats and validation
	parsedTime, err := dateparse.ParseAny(dateStr)
	if err != nil {
		// If dateparse fails, try some Facebook-specific patterns
		unix, foundTz, err := TryParsingFuzzyTimeStr(dateStr, fallbackTimezone)
		return unix, foundTz, err
	}

	// If dateparse succeeded but the time has no timezone info (Local),
	// we can't proceed without a valid timezone
	if parsedTime.Location() == time.Local {
		return 0, "", fmt.Errorf("parsed time has no timezone info and no fallback provided")
	}

	return parsedTime.Unix(), parsedTime.Location().String(), nil
}

// TryParsingFuzzyTimeStr attempts to parse Facebook-specific date formats
func TryParsingFuzzyTimeStr(dateStr string, fallbackTimezone *time.Location) (int64, string, error) {

	// Facebook formats we've seen:
	// "Sat, Jul 26 at 3:00 PM CDT"
	// "Fri, Jul 25 - Jul 26"
	// "Saturday, July 26 at 3:00 PM"
	// "Saturday, July 26, 2025 at 6:30PM – 9:30PM" (en-dash format)
	// "Saturday 26 July 2025 from 18:30-21:30" (new format with "from" and 24-hour time)

	// Handle timezone abbreviations that Go doesn't recognize by default
	// Note: Some abbreviations are ambiguous (e.g., CST can mean Central/China/Cuba Standard Time)
	// This mapping prioritizes the most commonly used interpretation

	// Build regex pattern dynamically from all timezone abbreviations in the map
	var tzAbbreviations []string
	for abbr := range TimezoneMap {
		tzAbbreviations = append(tzAbbreviations, abbr)
	}
	tzPatternStr := `\b(` + strings.Join(tzAbbreviations, "|") + `)\b`
	tzPattern := regexp.MustCompile(tzPatternStr)

	// Try to extract timezone from the string
	tzMatch := tzPattern.FindString(dateStr)
	var foundTimezone string

	targetTimezone := fallbackTimezone
	if tzMatch != "" {
		if tzName, exists := TimezoneMap[tzMatch]; exists {
			if loc, err := time.LoadLocation(tzName); err == nil {
				targetTimezone = loc
				foundTimezone = tzName // Store the found timezone name
			}
		}
	} else {
		// REMOVED: No more default timezone fallback
		// If no timezone found in the string, we can't proceed
		foundTimezone = ""
	}

	// Remove timezone from string for parsing
	dateStr = tzPattern.ReplaceAllString(dateStr, "")

	// Current year for dates that don't specify year
	currentYear := time.Now().Year()

	cleanedDateStr := strings.TrimSpace(dateStr)

	// Handle time ranges by splitting on various separators and keeping only the start time
	// Examples:
	// "Fri, Jul 25 - Jul 26" -> "Fri, Jul 25"
	// "Saturday, July 26, 2025 at 6:30PM – 9:30PM" -> "Saturday, July 26, 2025 at 6:30PM"
	// "Sep 12 at 10:00AM – Sep 13 at 5:00PM" -> "Sep 12 at 10:00AM"
	// "Saturday 26 July 2025 from 18:30-21:30" -> "Saturday 26 July 2025 from 18:30"
	// Handle both regular dash and en-dash (Unicode U+2013)
	if strings.Contains(cleanedDateStr, " – ") {
		// Unicode en-dash (U+2013) with spaces
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, " – ")[0])
	} else if strings.Contains(cleanedDateStr, "–") {
		// Unicode en-dash (U+2013) without spaces
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, "–")[0])
	} else if strings.Contains(cleanedDateStr, " - ") {
		// Regular dash with spaces
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, " - ")[0])
	} else if strings.Contains(cleanedDateStr, "-") {
		// Just dash (no spaces) - this is what we have: "18:30-21:30"
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, "-")[0])
	} else if strings.Contains(cleanedDateStr, " | ") {
		cleanedDateStr = strings.TrimSpace(strings.Split(cleanedDateStr, " | ")[0])
	}

	// Try various Facebook-style formats, including the new Facebook-specific formats
	formats := []string{
		// NEW: Facebook's specific format with "from" and 24-hour time
		"Monday 2 January 2006 from 15:04",    // "Saturday 26 July 2025 from 18:30"
		"Monday 2 January 2006 from 15:04:00", // "Saturday 26 July 2025 from 18:30:00"

		// NEW: Facebook's specific format with "at" and 24-hour time
		"Monday 2 January 2006 at 15:04",    // "Friday 11 July 2025 at 21:00"
		"Monday 2 January 2006 at 15:04:00", // "Friday 11 July 2025 at 21:00:00"

		// NEW: Facebook's specific format with full day, month, year, and time (en-dash format)
		"Monday, January 2, 2006 at 3:04PM",  // "Saturday, July 26, 2025 at 6:30PM"
		"Monday, January 2, 2006 at 3:04 PM", // "Saturday, July 26, 2025 at 6:30 PM"

		// Existing formats
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
		// Add format for day-first pattern like "Fri, Aug 8 at 5:00PM"
		"Mon, Jan 2 at 3:04 PM",        // "Fri, Aug 8 at 5:00PM"
		"Monday, January 2 at 3:04 PM", // "Friday, August 8 at 5:00 PM"
		// Add format for no-space-before-PM pattern
		"Mon, Jan 2 at 3:04PM",        // "Fri, Aug 8 at 5:00PM"
		"Monday, January 2 at 3:04PM", // "Friday, August 8 at 5:00PM"
		// Add format for abbreviated month pattern like "Sep 12 at 10:00AM"
		"Jan 2 at 3:04PM",  // "Sep 12 at 10:00AM"
		"Jan 2 at 3:04 PM", // "Sep 12 at 10:00 AM"
	}

	for _, format := range formats {
		if parsedTime, err := time.ParseInLocation(format, cleanedDateStr, targetTimezone); err == nil {
			// If no year was parsed, assume current year
			if parsedTime.Year() == 0 {
				parsedTime = parsedTime.AddDate(currentYear, 0, 0)
			}
			return parsedTime.Unix(), foundTimezone, nil
		} else {
			log.Printf("INFO: Format '%s' failed for date string '%s': %v", format, cleanedDateStr, err)
		}
	}

	return 0, foundTimezone, fmt.Errorf("unable to parse date format: %s", dateStr)
}

// FindFacebookEventData extracts event data from Facebook events pages specifically
func FindFacebookEventData(htmlContent string, locationTimezone string) ([]internal_types.EventInfo, error) {

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

	childScrapeQueue := []internal_types.EventInfo{}
	for _, event := range events {
		if event.EventDescription == "" || event.EventTimezone == "" {
			childScrapeQueue = append(childScrapeQueue, event)
		}
	}

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

// // findEventsInJSON extracts events from parsed JSON data
// func findEventsInJSON(data map[string]interface{}, locationTimezone string) []internal_types.EventInfo {
// 	var events []internal_types.EventInfo

// 	// Recursively search for event objects in the JSON structure
// 	events = findEventsInJSON(data, events, locationTimezone)

// 	return events
// }

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
		urlCandidates = append(urlCandidates, unescapeJSON(url))
	}

	// Date string
	dateStr := ""
	if dateTime, ok := eventObj["day_time_sentence"].(string); ok {
		dateStr, _ = cleanDateString(cleanUnicodeText(unescapeJSON(dateTime)))
	}

	// Location candidates in priority order
	var locationCandidates []string
	if v, ok := eventObj["contextual_name"].(string); ok {
		locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSON(v)))
	}
	if v, ok := eventObj["location"].(string); ok {
		locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSON(v)))
	}
	if place, ok := eventObj["place"].(map[string]interface{}); ok {
		if name, ok := place["name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSON(name)))
		}
	}
	if venue, ok := eventObj["venue"].(map[string]interface{}); ok {
		if name, ok := venue["name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSON(name)))
		}
	}
	if eventPlace, ok := eventObj["event_place"].(map[string]interface{}); ok {
		if name, ok := eventPlace["contextual_name"].(string); ok {
			locationCandidates = append(locationCandidates, cleanUnicodeText(unescapeJSON(name)))
		}
	}

	// Description
	description := ""
	if v, ok := eventObj["description"].(string); ok {
		description = cleanUnicodeText(unescapeJSON(v))
	}

	// Organizer
	organizer := ""
	if creator, ok := eventObj["event_creator"].(map[string]interface{}); ok {
		if creatorName, ok := creator["name"].(string); ok {
			organizer = cleanUnicodeText(unescapeJSON(creatorName))
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
	info, ok := normalizeEventInfo(event.EventTitle, urlCandidates, dateStr, locationCandidates, organizer, description, defaultTimezone)
	if !ok {
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
) (internal_types.EventInfo, bool) {
	// Select URL: choose the last non-empty candidate
	url := ""
	for i := len(urlCandidates) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(urlCandidates[i]); s != "" {
			url = s
			break
		}
	}
	if title == "" || url == "" {
		return internal_types.EventInfo{}, false
	}

	// Fallback timezone if needed
	if fallbackTimezone == nil {
		return internal_types.EventInfo{}, false
	}

	// Validate date string and get timezone
	if strings.TrimSpace(dateStr) == "" {
		return internal_types.EventInfo{}, false
	}
	_, tzString, err := ParseFlexibleDatetime(dateStr, fallbackTimezone)
	if err != nil {
		return internal_types.EventInfo{}, false
	}

	// Choose first non-empty location candidate
	location := ""
	for _, cand := range locationCandidates {
		if s := strings.TrimSpace(cand); s != "" {
			location = s
			break
		}
	}

	evt := internal_types.EventInfo{
		EventTitle:       title,
		EventLocation:    location,
		EventStartTime:   dateStr, // keep original cleaned string
		EventURL:         url,
		EventDescription: description,
		EventTimezone:    tzString, // Use the timezone found during parsing
		EventHostName:    organizer,
	}
	return evt, true
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

// cleanDateString removes extra text after timezone codes like "and 20 more"
func cleanDateString(dateStr string) (date string, tz string) {

	// Build regex pattern dynamically from all timezone abbreviations in the map
	var tzAbbreviations []string
	for abbr := range TimezoneMap {
		tzAbbreviations = append(tzAbbreviations, abbr)
	}
	tzPatternStr := `\b(` + strings.Join(tzAbbreviations, "|") + `)\b`
	tzPattern := regexp.MustCompile(tzPatternStr)

	// Find the timezone in the date string
	tzMatch := tzPattern.FindString(dateStr)
	if tzMatch != "" {
		// Find the position of the timezone
		tzIndex := strings.Index(dateStr, tzMatch)
		if tzIndex != -1 {
			// Keep everything up to and including the timezone, remove everything after
			dateStr = strings.TrimSpace(dateStr[:tzIndex+len(tzMatch)])
		}
	}

	return dateStr, tzMatch
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
		// For single event pages, look deeper in the JSON structure
		events = ExtractFbSingleEventFromJSON(eventScriptContent, locationTimezone)
	} else {
		// For event list pages, use the existing logic
		events = ExtractFbEventsFromJSON(eventScriptContent, locationTimezone)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events extracted from JSON content")
	}

	return events, nil
}

// ExtractFbSingleEventFromJSON extracts a single event from JSON content
// This handles the deeper nesting structure like result.data.event
func ExtractFbSingleEventFromJSON(jsonContent string, locationTimezone string) []internal_types.EventInfo {
	var events []internal_types.EventInfo

	// Fall back to parsing the entire JSON content
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(jsonContent), &jsonData)
	if err != nil {
		log.Printf("Failed to parse JSON content: %v", err)
		return events
	}

	// Look for the specific path: result.data.event
	events = findSingleEventInJSONStructure(jsonData, locationTimezone)

	// If not found in the expected path, fall back to the general search
	if len(events) == 0 {
		events = findEventsInJSON(jsonData, events, locationTimezone)
	}

	return events
}

// findSingleEventInJSONStructure looks for events in the deeper nesting structure
// This handles paths like result.data.event
func findSingleEventInJSONStructure(data map[string]interface{}, locationTimezone string) []internal_types.EventInfo {
	var events []internal_types.EventInfo

	// Look for the specific path: result.data.event
	if result, ok := data["result"].(map[string]interface{}); ok {
		if data, ok := result["data"].(map[string]interface{}); ok {
			if event, ok := data["event"].(map[string]interface{}); ok {
				// Check if this is an event object
				if typename, ok := event["__typename"].(string); ok && typename == "Event" {
					eventInfo := extractEventFromObject(event)
					if eventInfo.EventTitle != "" {
						events = append(events, eventInfo)
					}
				}
			}
		}
	}

	// If not found in the expected path, fall back to the general search
	if len(events) == 0 {
		events = findEventsInJSON(data, events, locationTimezone)
	}

	return events
}
