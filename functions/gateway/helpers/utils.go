package helpers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/meetnearme/api/functions/gateway/types"
)

var DefaultProtocol string

func InitDefaultProtocol() {
	if os.Getenv("GO_ENV") == "test" {
		DefaultProtocol = "http://"
	} else {
		DefaultProtocol = "https://"
	}
}

func init() {
	InitDefaultProtocol()
}

func UtcOrUnixToUnix64(t interface{}) (int64, error) {
	switch v := t.(type) {
	case int64:
		// Validate that the timestamp is within 100 years of now
		now := time.Now().Unix()
		hundredYearsInSeconds := int64(100 * 365.25 * 24 * 60 * 60)
		if v < now-hundredYearsInSeconds || v > now+hundredYearsInSeconds {
			return 0, fmt.Errorf("unix timestamp must be within 100 years of the current time")
		}
		return v, nil
	case string:
		parsedTime, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return 0, fmt.Errorf("invalid date format, must be RFC3339: %v", err)
		}
		return parsedTime.Unix(), nil
	default:
		return 0, fmt.Errorf("unsupported time format")
	}
}

func FormatDateLocal(t time.Time) (string, error) {
	if t.IsZero() {
		return "", fmt.Errorf("not a valid time: %v", t)
	}
	// Format: "Dec 24, 2024 (Tue)"
	return t.Format("Jan 2, 2006 (Mon)"), nil
}

func FormatTimeLocal(t time.Time) (string, error) {
	if t.IsZero() {
		return "", fmt.Errorf("not a valid time: %v", t)
	}
	// Format: "7:00pm"
	return strings.ToLower(t.Format("3:04PM")), nil
}

func TruncateStringByBytes(str string, limit int) (s string, exceededLimit bool) {
	byteLen := len([]byte(str))
	if byteLen <= limit {
		return str, false
	}
	return string([]byte(str)[:limit]), false
}

func GetBaseUrlFromReq(r *http.Request) string {
	return r.URL.Scheme + "://" + r.URL.Host
}

func HashIDtoImgRange(id string, num int) int {
	hash := md5.Sum([]byte(id))
	hashInt := int(hash[0]) % num
	return hashInt
}

func GetImgUrlFromHash(event types.Event) string {
	baseUrl := os.Getenv("STATIC_BASE_URL") + "/assets/img/"
	noneCatImgCount := 18
	catNumString := "_"
	if len(event.Categories) > 0 {
		firstCat := ArrFindFirst(event.Categories, []string{"Karaoke", "Bocce Ball", "Trivia Night"})
		firstCat = strings.ToLower(strings.ReplaceAll(firstCat, " ", "-"))
		if firstCat == "" {
			firstCat = "none"
		}
		catImgCountRange := 0
		switch firstCat {
		case "none":
			catImgCountRange = noneCatImgCount
		case "karaoke":
			catImgCountRange = 4
		case "bocce-ball":
			catImgCountRange = 4
		case "trivia-night":
			catImgCountRange = 4
		}
		imgNum := HashIDtoImgRange(event.Id, catImgCountRange)
		if imgNum < 10 {
			catNumString = catNumString + "0"
		}

		imgName := "cat_" + firstCat + catNumString + fmt.Sprint(imgNum) + ".jpeg"
		return baseUrl + imgName
	}
	imgNum := HashIDtoImgRange(event.Id, noneCatImgCount)
	if imgNum < 10 {
		catNumString = catNumString + "0"
	}
	return baseUrl + "cat_none" + catNumString + fmt.Sprint(imgNum) + ".jpeg"
}

func SetCloudflareKV(subdomainValue, userID, userMetadataKey string, metadata map[string]string) error {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	namespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")

	// First, check if the key already exists
	readURL := fmt.Sprintf("%s/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		os.Getenv("CLOUDFLARE_API_BASE_URL"), accountID, namespaceID, subdomainValue)

	req, err := http.NewRequest("GET", readURL, nil)
	if err != nil {
		return fmt.Errorf("error creating read request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("CLOUDFLARE_API_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending read request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return fmt.Errorf(ERR_KV_KEY_EXISTS)
	}

	existingUserSubdomain, err := GetUserMetadataByKey(userID, userMetadataKey)
	if err != nil {
		return fmt.Errorf("error getting user metadata key: %w", err)
	}
	// Decode the base64 encoded existingUserSubdomain
	decodedValue, err := base64.StdEncoding.DecodeString(existingUserSubdomain)
	if err != nil {
		return fmt.Errorf("error decoding base64 existingUserSubdomain: %w", err)
	}
	existingUserSubdomain = string(decodedValue)

	// Next check if the user already has a subdomain set
	err = UpdateUserMetadataKey(userID, userMetadataKey, subdomainValue)
	if err != nil {
		return fmt.Errorf("error updating user metadata: %w", err)
	}

	// If the key doesn't exist, proceed with writing
	writeURL := fmt.Sprintf("%s/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		os.Getenv("CLOUDFLARE_API_BASE_URL"), accountID, namespaceID, subdomainValue)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add value
	_ = writer.WriteField("value", userID)

	// Add metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}
	_ = writer.WriteField("metadata", string(metadataJSON))

	writer.Close()

	req, err = http.NewRequest("PUT", writeURL, body)
	if err != nil {
		return fmt.Errorf("error creating write request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("CLOUDFLARE_API_TOKEN"))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending write request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set KV: %s", resp.Status)
	}

	defer func() {
		if existingUserSubdomain != "" {
			DeleteCloudflareKV(existingUserSubdomain, userID)
		}
	}()

	return nil
}

func DeleteCloudflareKV(subdomainValue, userID string) error {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	namespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")

	readURL := fmt.Sprintf("%s/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		os.Getenv("CLOUDFLARE_API_BASE_URL"), accountID, namespaceID, subdomainValue)

	req, err := http.NewRequest("DELETE", readURL, nil)
	if err != nil {
		return fmt.Errorf("error creating read request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("CLOUDFLARE_API_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending read request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete key / value pair in kv store: %w", err)
	}

	return nil
}

func GetUserMetadataByKey(userID, key string) (string, error) {
	url := fmt.Sprintf(DefaultProtocol+"%s/management/v1/users/%s/metadata/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID, key)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", err
	}

	if len(respData) == 0 || respData["metadata"] == nil {
		log.Printf("respData is empty or nil")
		return "", nil
	}
	metadata, ok := respData["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("metadata is not of type map[string]interface{}")
	}

	value, ok := metadata["value"].(string)
	if !ok {
		return "", fmt.Errorf("value is not of type string")
	}

	return value, nil
}

type ZitadelUserSearchResponse struct {
	Details struct {
		TotalResult string `json:"totalResult"`
		Timestamp   string `json:"timestamp"`
	} `json:"details"`
	Result []struct {
		UserID             string `json:"userId"`
		Username           string `json:"username"`
		PreferredLoginName string `json:"preferredLoginName"`
		State              string `json:"state"`
		Human              struct {
			Profile struct {
				DisplayName string `json:"displayName"`
			} `json:"profile"`
			Email map[string]interface{} `json:"email"`
		} `json:"human"`
	} `json:"result"`
}

type UserSearchResult struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

func SearchUsersByIDs(userIDs []string) ([]UserSearchResult, error) {
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("userIDs must contain at least one element")
	}
	url := fmt.Sprintf(DefaultProtocol+"%s/v2/users", os.Getenv("ZITADEL_INSTANCE_HOST"))
	method := "POST"

	// Create the payload with inUserIdsQuery
	payload := fmt.Sprintf(`{
        "query": {
            "offset": 0,
            "limit": 100,
            "asc": true
        },
        "sortingColumn": "USER_FIELD_NAME_UNSPECIFIED",
        "queries": [
            {
                "inUserIdsQuery": {
                    "userIds": %s
                }
            }
        ]
    }`, ToJSON(userIDs))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(payload))
	if err != nil {
		return []UserSearchResult{}, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return []UserSearchResult{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []UserSearchResult{}, err
	}

	var respData ZitadelUserSearchResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return empty array if no results
	if len(respData.Result) == 0 {
		return []UserSearchResult{}, nil
	}

	// Map users to UserSearchResult
	var results []UserSearchResult
	for _, user := range respData.Result {
		results = append(results, UserSearchResult{
			UserID:      user.UserID,
			DisplayName: user.Human.Profile.DisplayName,
		})
	}

	return results, nil
}

func SearchUserByEmailOrName(query string) ([]UserSearchResult, error) {
	url := fmt.Sprintf(DefaultProtocol+"%s/v2/users", os.Getenv("ZITADEL_INSTANCE_HOST"))
	method := "POST"

	payload := fmt.Sprintf(`{
		"query": {
			"offset": 0,
			"limit": 100,
			"asc": true
		},
		"sortingColumn": "USER_FIELD_NAME_UNSPECIFIED",
		"queries": [
			{
				"typeQuery": {
					"type": "TYPE_HUMAN"
				}
			},
			{
				"orQuery": {
					"queries": [
						{
							"emailQuery": {
								"emailAddress": "%s",
								"method": "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE"
							}
						},
						{
							"userNameQuery": {
								"userName": "%s",
								"method": "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE"
							}
						}
					]
				}
			}
		]
	}`, query, query)

	readerPayload := strings.NewReader(payload)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, readerPayload)
	if err != nil {
		return []UserSearchResult{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return []UserSearchResult{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []UserSearchResult{}, err
	}
	var respData ZitadelUserSearchResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return empty array if no results
	if len(respData.Result) == 0 {
		return []UserSearchResult{}, nil
	}

	// Filter and map active users to UserSearchResult
	var results []UserSearchResult
	for _, user := range respData.Result {
		results = append(results, UserSearchResult{
			// we specifically omit `preferredLoginName` because it's an email address and we want
			// to prevent a scenario where we expose a user's email address to a unauthorized party
			UserID:      user.UserID,
			DisplayName: user.Human.Profile.DisplayName,
		})

	}

	return results, nil
}

func UpdateUserMetadataKey(userID, key, value string) error {
	url := fmt.Sprintf(DefaultProtocol+"%s/management/v1/users/%s/metadata/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID, key)
	method := "POST"

	payload := strings.NewReader(`{
		"value": "` + base64.StdEncoding.EncodeToString([]byte(value)) + `"
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Println(err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		var respData map[string]interface{}
		if err := json.Unmarshal(body, &respData); err != nil {
			return err
		}
		return fmt.Errorf("failed to update user metadata: %s, reason: %s", res.Status, respData)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("saved user metadata body response: ", string(body))
	return nil
}

func ArrFindFirst(needles []string, haystack []string) string {
	for _, s := range needles {
		for _, item := range haystack {
			if s == item {
				return s
			}
		}
	}
	return ""
}

func GetDateOrShowNone(datetime int64, timezone string) string {
	_, formattedDate := GetLocalDateAndTime(datetime, timezone)
	if formattedDate == "" {
		return ""
	}
	return formattedDate
}

func GetTimeOrShowNone(datetime int64, timezone string) string {
	formattedTime, _ := GetLocalDateAndTime(datetime, timezone)
	if formattedTime == "" {
		return ""
	}
	return formattedTime
}

func GetDatetimePickerFormatted(datetime int64) string {
	return time.Unix(datetime, 0).Format("2006-01-02T15:04")
}

func GetLocalDateAndTime(datetime int64, timezone string) (string, string) {
	// Load the location based on the event's timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		fmt.Println("Error loading timezone:", err)
		return "", ""
	}

	// Convert start time to local time
	localStartTime := time.Unix(datetime, 0).In(loc)

	// Populate the local date and time fields
	localStartDateStr, _ := FormatDateLocal(localStartTime)
	localStartTimeStr, _ := FormatTimeLocal(localStartTime)

	return localStartTimeStr, localStartDateStr
}

func FormatTimeRFC3339(unixTimestamp int64) string {
	t := time.Unix(unixTimestamp, 0).UTC()
	return t.Format("20060102T150405Z")
}

func FormatTimeForGoogleCalendar(timestamp int64, timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// If there's an error loading the timezone, fall back to UTC
		loc = time.UTC
	}

	t := time.Unix(timestamp, 0).In(loc)
	return t.Format("20060102T150405")
}

func ToJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func IsAuthorizedEditor(event *types.Event, userInfo *UserInfo) bool {
	for _, ownerId := range event.EventOwners {
		if ownerId == userInfo.Sub {
			return true
		}
	}
	return false
}

func HasRequiredRole(roleClaims []RoleClaim, requiredRoles []string) bool {
	for _, claim := range roleClaims {
		for _, validRole := range requiredRoles {
			if claim.Role == validRole {
				return true
			}
		}
	}
	return false
}

func CanEditEvent(event *types.Event, userInfo *UserInfo, roleClaims []RoleClaim) bool {
	hasSuperAdminRole := HasRequiredRole(roleClaims, []string{"superAdmin", "eventAdmin"})
	isEditor := IsAuthorizedEditor(event, userInfo)
	return hasSuperAdminRole || isEditor
}

func GetBase64ValueFromMap(claimsMeta map[string]interface{}, key string) string {
	interestMetadataValue, ok := claimsMeta[key].(string)
	if !ok {
		return ""
	}

	// Add padding if needed
	padding := len(interestMetadataValue) % 4
	if padding > 0 {
		interestMetadataValue += strings.Repeat("=", 4-padding)
	}

	// Decode base64 string
	decodedValue, err := base64.StdEncoding.DecodeString(interestMetadataValue)
	if err != nil {
		log.Printf("Failed to decode user interests metadata: %v", err)
		return ""
	}
	return string(decodedValue)

}

func GetUserInterestFromMap(claimsMeta map[string]interface{}, key string) []string {
	interests := GetBase64ValueFromMap(claimsMeta, key)
	return strings.Split(interests, "|")
}
