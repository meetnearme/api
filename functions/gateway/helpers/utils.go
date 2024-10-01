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

func FormatDate(unixTimestamp int64) (string, error) {
	if unixTimestamp == 0 {
		return "", fmt.Errorf("not a valid unix timestamp: %v", unixTimestamp)
	}
	date := time.Unix(unixTimestamp, 0).UTC()
	return date.Format("Jan 2, 2006 (Mon)"), nil
}

func FormatTime(unixTimestamp int64) (string, error) {
	if unixTimestamp == 0 {
		return "", fmt.Errorf("not a valid unix timestamp: %v", unixTimestamp)
	}
	_time := time.Unix(unixTimestamp, 0).UTC()
	return _time.Format("3:04pm"), nil
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
	baseUrl := os.Getenv("STATIC_BASE_URL") + "/assets/img/";

	if len(event.Categories) > 0 {
		if (ArrContains(event.Categories, "Karaoke")) {
				return baseUrl + "cat_karaoke_0" + fmt.Sprint(HashIDtoImgRange(event.Id, 4)) + ".jpeg"
		} else if (ArrContains(event.Categories, "Bocce Ball")) {
			return baseUrl + "cat_bocce_ball_0" + fmt.Sprint(HashIDtoImgRange(event.Id, 4)) + ".jpeg"
		} else if (ArrContains(event.Categories, "Trivia Night")) {
			return baseUrl + "cat_trivia_night_0" + fmt.Sprint(HashIDtoImgRange(event.Id, 4)) + ".jpeg"
		}
	}
	return baseUrl + fmt.Sprint(HashIDtoImgRange(event.Id, 8)) + ".png"
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

func UpdateUserMetadataKey(userID, key, value string) error {
	url := fmt.Sprintf(DefaultProtocol+"%s/management/v1/users/%s/metadata/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID, key)
	method := "POST"

	payload := strings.NewReader(`{
		"value": "` + base64.StdEncoding.EncodeToString([]byte(value)) + `"
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		return err
	}
	fmt.Println("saved user metadata body response: ", string(body))
	return nil
}

func ArrContains(slice []string, item string) bool {
	for _, s := range slice {
			if s == item {
					return true
			}
	}
	return false
}


func GetDateOrShowNone(date int64) string {
	formattedDate, err := FormatDate(date)
	if date == 0 || err != nil {
		return ""
	}
	return formattedDate
}

func GetTimeOrShowNone(time int64) string {
	formattedTime, err := FormatTime(time)
	if time == 0 || err != nil {
		return ""
	}
	return formattedTime
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

	localUnixTime := localStartTime.Unix()

	// Populate the local date and time fields
	localStartDateStr, _ := FormatDate(localUnixTime)
	localStartTimeStr, _ := FormatTime(localUnixTime)

	return localStartTimeStr, localStartDateStr
}
