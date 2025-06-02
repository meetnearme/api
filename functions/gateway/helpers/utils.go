package helpers

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go/v3"
	"github.com/cloudflare/cloudflare-go/v3/kv"
	"github.com/cloudflare/cloudflare-go/v3/option"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
	_ "github.com/imroc/req"

	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var validate *validator.Validate = validator.New()

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

func UtcToUnix64(t interface{}, timezone *time.Location) (int64, error) {
	switch v := t.(type) {
	case string:
		// First validate by parsing as RFC3339
		_, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return 0, fmt.Errorf("invalid date format: %v", err)
		}

		// Now do local time parsing
		timeStr := strings.TrimSuffix(v, "Z")
		// Note the time layout here MUST NOT be RFC3339, it must be the local time layout
		localTime, err := time.ParseInLocation("2006-01-02T15:04:05", timeStr, timezone)
		if err != nil {
			return 0, fmt.Errorf("invalid date format: %v", err)
		}
		return localTime.Unix(), nil

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
		firstCat := ArrFindFirst(event.Categories, []string{"Karaoke", "Karaoke Quirky", "Bocce Ball", "Trivia Night", "Soccer"})
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
		case "karaoke-quirky":
			catImgCountRange = 9
		case "bocce-ball":
			catImgCountRange = 4
		case "trivia-night":
			catImgCountRange = 4
		case "soccer":
			catImgCountRange = 10
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

func GetCloudflareMnmOptions(subdomainValue string) (string, error) {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	namespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
	baseURL := os.Getenv("CLOUDFLARE_API_CLIENT_BASE_URL")
	cfClient := cloudflare.NewClient(
		option.WithAPIKey(accountID),
		option.WithBaseURL(baseURL),
	)

	resp, err := cfClient.KV.Namespaces.Values.Get(
		context.TODO(),
		namespaceID,
		subdomainValue,
		kv.NamespaceValueGetParams{
			AccountID: cloudflare.F(accountID),
		},
	)
	if err != nil {
		return "", fmt.Errorf("error getting cloudflare mnm options: %w", err)
	}

	// return the response body as a string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading cloudflare mnm options: %w", err)
	}

	return string(body), nil
}

func SetCloudflareMnmOptions(subdomainValue, userID string, metadata map[string]string, cfMetadataValue string) error {
	mnmOptionsKey := SUBDOMAIN_KEY
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	namespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")
	baseURL := os.Getenv("CLOUDFLARE_API_CLIENT_BASE_URL")

	cfClient := cloudflare.NewClient(
		option.WithAPIKey(accountID),
		option.WithBaseURL(baseURL),
	)

	// First, check if the key already exists
	resp, err := cfClient.KV.Namespaces.Values.Get(
		context.TODO(),
		namespaceID,
		subdomainValue,
		kv.NamespaceValueGetParams{
			AccountID: cloudflare.F(accountID),
		},
	)

	kvValueExists := false
	if resp.StatusCode == http.StatusOK {
		kvValueExists = true
	}

	existingValueStr := ""
	existingRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading existing user subdomain body request: %+v", err.Error())
	}
	existingRespBodyStr := string(existingRespBody)
	if resp.StatusCode == http.StatusOK {
		existingValueStr = existingRespBodyStr
	}

	// check for pattern [0-9]{18} => Zitadel UserID pattern
	hasLegacyUserID := false
	if !regexp.MustCompile(`^[0-9]{` + fmt.Sprint(ZITADEL_USER_ID_LEN) + `}$`).MatchString(userID) {
		hasLegacyUserID = true
	}

	// from the `existingValueStr` `<key1>=<val1>;<key2>=<val2>;...`
	// parse the userId from the string, don't assume position, instead
	// search for `userId=` and get the value after it

	// also handle for the case where the value is simply `123` lacking
	// colons `=` and `;`
	if !hasLegacyUserID && strings.Contains(existingValueStr, "userId=") {
		existingValueStr = strings.Split(existingValueStr, "userId=")[1]
		existingValueStr = strings.Split(existingValueStr, ";")[0]
	}

	if existingValueStr != userID && resp.StatusCode == http.StatusOK {
		return fmt.Errorf(ERR_KV_KEY_EXISTS)
	}

	existingUserSubdomain, err := GetUserMetadataByKey(userID, mnmOptionsKey)
	if err != nil {
		return fmt.Errorf("error getting user metadata key: %w", err)
	}
	// Decode the base64 encoded existingUserSubdomain
	decodedValue, err := base64.StdEncoding.DecodeString(existingUserSubdomain)
	if err != nil {
		return fmt.Errorf("error decoding base64 existingUserSubdomain: %w", err)
	}
	existingUserSubdomain = string(decodedValue)

	// Write the new subdomain value to the user's metadata in zitadel
	err = UpdateUserMetadataKey(userID, mnmOptionsKey, subdomainValue)
	if err != nil {
		return fmt.Errorf("error updating user metadata: %w", err)
	}

	// If the user metadata write was successful, write the new subdomain value to the KV store in Cloudflare
	// NOTE: importantly, we write directly rather than using the cloudflare client
	// because the client does not allow raw stings, but enforces JSON in the field value
	writeURL := fmt.Sprintf("%s/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		os.Getenv("CLOUDFLARE_API_BASE_URL"), accountID, namespaceID, subdomainValue)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add value
	_ = writer.WriteField("value", cfMetadataValue)

	// Add metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}
	_ = writer.WriteField("metadata", string(metadataJSON))
	writer.Close()

	req, err := http.NewRequest("PUT", writeURL, body)
	if err != nil {
		return fmt.Errorf("error creating write request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("CLOUDFLARE_API_TOKEN"))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending write request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set KV: %s", resp.Status)
	}

	defer func() {
		// if the key already exists, and the userID is different, delete the existing key
		if kvValueExists && existingUserSubdomain != "" && existingValueStr != userID {
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

func SearchUsersByIDs(userIDs []string, dangerous bool) ([]types.UserSearchResultDangerous, error) {
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
		return []types.UserSearchResultDangerous{}, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return []types.UserSearchResultDangerous{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []types.UserSearchResultDangerous{}, err
	}

	var respData ZitadelUserSearchResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return empty array if no results
	if len(respData.Result) == 0 {
		return []types.UserSearchResultDangerous{}, nil
	}

	// Map users to UserSearchResult
	var results []types.UserSearchResultDangerous
	for _, user := range respData.Result {
		appendItem := types.UserSearchResultDangerous{
			UserID:      user.UserID,
			DisplayName: user.Human.Profile.DisplayName,
		}
		// WARNING: enabling this is a security risk if not
		// done with extreme caution
		if dangerous {
			appendItem.Email = user.Human.Email["email"].(string)
		}
		results = append(results, appendItem)
	}

	return results, nil
}

func SearchUserByEmailOrName(query string) ([]types.UserSearchResult, error) {
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
							"displayNameQuery": {
								"displayName": "%s",
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
		return []types.UserSearchResult{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return []types.UserSearchResult{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return []types.UserSearchResult{}, err
	}
	var respData ZitadelUserSearchResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Return empty array if no results
	if len(respData.Result) == 0 {
		return []types.UserSearchResult{}, nil
	}

	// Filter and map active users to UserSearchResult
	var results []types.UserSearchResult
	for _, user := range respData.Result {
		results = append(results, types.UserSearchResult{
			// we specifically omit `preferredLoginName` because it's an email address and we want
			// to prevent a scenario where we expose a user's email address to a unauthorized party
			UserID:      user.UserID,
			DisplayName: user.Human.Profile.DisplayName,
		})

	}

	return results, nil
}

func GetOtherUserByID(userID string) (types.UserSearchResult, error) {
	// https://zitadel.com/docs/apis/resources/user_service_v2/user-service-get-user-by-id
	url := fmt.Sprintf(DefaultProtocol+"%s/v2/users/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return types.UserSearchResult{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return types.UserSearchResult{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.UserSearchResult{}, err
	}

	// Check for error status codes
	if res.StatusCode != http.StatusOK {
		var errResp struct {
			Code    int32  `json:"code"`
			Message string `json:"message"`
			Details []struct {
				Type string `json:"@type"`
			} `json:"details"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return types.UserSearchResult{}, fmt.Errorf("failed to get user: status %d", res.StatusCode)
		}
		return types.UserSearchResult{}, fmt.Errorf("failed to get user: %s", errResp.Message)
	}

	var respData struct {
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Human    struct {
				Profile struct {
					DisplayName string `json:"displayName"`
				} `json:"profile"`
			} `json:"human"`
		} `json:"user"`
	}

	if err := json.Unmarshal(body, &respData); err != nil {
		return types.UserSearchResult{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	result := types.UserSearchResult{
		UserID:      respData.User.ID,
		DisplayName: respData.User.Human.Profile.DisplayName,
	}

	return result, nil
}

func GetOtherUserMetaByID(userID, key string) (string, error) {
	// https://zitadel.com/docs/apis/resources/mgmt/management-service-get-user-metadata
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
	if res.StatusCode > 400 {
		return "", fmt.Errorf("failed to get user metadata: %v", res.StatusCode)
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(body, &respData); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Get the metadata map
	metadata, ok := respData["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("metadata not found or invalid format")
	}

	// Use the helper function with the metadata map
	decodedValue := GetBase64ValueFromMap(metadata, "value")
	if decodedValue == "" {
		return "", fmt.Errorf("no value found or failed to decode")
	}

	return decodedValue, nil
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

// Define a struct for the request payload
type createUserPayload struct {
	UserID string `json:"userId"`
	Email  struct {
		Email      string `json:"email"`
		IsVerified bool   `json:"isVerified"`
	} `json:"email"`
	Password struct {
		Password       string `json:"password"`
		ChangeRequired bool   `json:"changeRequired"`
	} `json:"password"`
	Profile struct {
		GivenName   string `json:"givenName"`
		FamilyName  string `json:"familyName"`
		NickName    string `json:"nickName"`
		DisplayName string `json:"displayName"`
	} `json:"profile"`
	Metadata []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"metadata"`
}

func CreateTeamUserWithMembers(displayName, candidateUUID string, members []string) (types.UserSearchResultDangerous, error) {
	err := ValidateTeamUUID(candidateUUID)
	if err != nil {
		return types.UserSearchResultDangerous{}, err
	}

	emailSchema := os.Getenv("USER_TEAM_EMAIL_SCHEMA")
	password := os.Getenv("USER_TEAM_PASSWORD")
	email := strings.Replace(emailSchema, "<replace>", candidateUUID, 1)

	nameParts := strings.SplitN(displayName, " ", 2)
	firstPartName := nameParts[0]
	// NOTE: this is a hack, zitadel doesn't accept omission of first + last names
	secondPartName := "."
	if strings.Contains(displayName, " ") {
		secondPartName = nameParts[1]
	}

	// Create the payload struct
	payload := createUserPayload{
		UserID: candidateUUID,
	}
	payload.Email.Email = email
	payload.Email.IsVerified = true
	payload.Password.Password = password
	payload.Password.ChangeRequired = false
	payload.Profile.GivenName = firstPartName
	payload.Profile.FamilyName = secondPartName
	payload.Profile.NickName = displayName
	payload.Profile.DisplayName = displayName

	// Add metadata
	if len(members) > 0 {
		payload.Metadata = append(payload.Metadata, struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}{
			Key:   "members",
			Value: base64.StdEncoding.EncodeToString([]byte(strings.Join(members, ","))),
		})
	}

	payload.Metadata = append(payload.Metadata, struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   "userType",
		Value: base64.StdEncoding.EncodeToString([]byte("team")),
	})

	// Marshal the payload struct to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return types.UserSearchResultDangerous{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf(DefaultProtocol+"%s/v2/users/human", os.Getenv("ZITADEL_INSTANCE_HOST"))
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonPayload))
	if err != nil {
		return types.UserSearchResultDangerous{}, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

	res, err := client.Do(req)
	if err != nil {
		return types.UserSearchResultDangerous{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.UserSearchResultDangerous{}, err
	}

	var respData types.UserSearchResultDangerous
	if err := json.Unmarshal(body, &respData); err != nil {
		return types.UserSearchResultDangerous{}, err
	}

	return respData, nil
}

func ValidateTeamUUID(candidateUUID string) error {
	parts := strings.SplitN(candidateUUID, "tm_", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid UUID format after 'tm_': %s", parts[1])
	}
	_uuid := parts[1]
	_, err := uuid.Parse(_uuid)
	if err != nil {
		return fmt.Errorf("invalid UUID format after 'tm_': %s", _uuid)
	}
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

func GetDateOrShowNone(datetime int64, timezone time.Location) string {
	_, formattedDate := GetLocalDateAndTime(datetime, timezone)
	if formattedDate == "" {
		return ""
	}
	return formattedDate
}

func GetTimeOrShowNone(datetime int64, timezone time.Location) string {
	formattedTime, _ := GetLocalDateAndTime(datetime, timezone)
	if formattedTime == "" {
		return ""
	}
	return formattedTime
}

func GetDatetimePickerFormatted(datetime int64, timezone *time.Location) string {
	if timezone == nil {
		timezone = time.UTC
	}
	return time.Unix(datetime, 0).In(timezone).Format("2006-01-02T15:04")
}

func GetLocalDateAndTime(datetime int64, timezone time.Location) (string, string) {
	// Load the location based on the event's timezone
	loc, err := time.LoadLocation(timezone.String())
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

func FormatTimeMMDDYYYY(unixTimestamp int64) string {
	t := time.Unix(unixTimestamp, 0).UTC()
	return t.Format("01/02/06 03:04PM")
}

func FormatTimeForGoogleCalendar(timestamp int64, timezone time.Location) string {
	loc, err := time.LoadLocation(timezone.String())
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

func IsAuthorizedEventEditor(event *types.Event, userInfo *UserInfo) bool {
	for _, ownerId := range event.EventOwners {
		if ownerId == userInfo.Sub {
			return true
		}
	}
	return false
}

func IsAuthorizedCompetitionEditor(competition types.CompetitionConfig, userInfo *UserInfo) bool {
	allOwners := []string{competition.PrimaryOwner}
	for _, owner := range competition.AuxilaryOwners {
		allOwners = append(allOwners, owner)
	}
	for _, owner := range allOwners {
		if owner == userInfo.Sub {
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
	hasSuperAdminRole := HasRequiredRole(roleClaims, []string{"superAdmin"})
	isEditor := IsAuthorizedEventEditor(event, userInfo)
	return hasSuperAdminRole || isEditor
}

func CanEditCompetition(competition types.CompetitionConfig, userInfo *UserInfo, roleClaims []RoleClaim) bool {
	hasSuperAdminRole := HasRequiredRole(roleClaims, []string{"superAdmin", "competitionAdmin"})
	isEditor := IsAuthorizedCompetitionEditor(competition, userInfo)
	return hasSuperAdminRole || isEditor
}

func GetBase64ValueFromMap(claimsMeta map[string]interface{}, key string) string {
	metadataValue, ok := claimsMeta[key].(string)
	if !ok {
		return ""
	}

	// Add padding if needed
	padding := len(metadataValue) % 4
	if padding > 0 {
		metadataValue += strings.Repeat("=", 4-padding)
	}

	// Decode base64 string
	decodedValue, err := base64.StdEncoding.DecodeString(metadataValue)
	if err != nil {
		log.Printf("Failed to decode base64 metadata: %v", err)
		return ""
	}
	return string(decodedValue)
}

func GetUserInterestFromMap(claimsMeta map[string]interface{}, key string) []string {
	interests := GetBase64ValueFromMap(claimsMeta, key)
	return strings.Split(interests, "|")
}

func CalculateTTL(days int) int64 {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).Unix()
}

func NormalizeCompetitionRounds(competitionRounds []internal_types.CompetitionRoundUpdate) ([]internal_types.CompetitionRoundUpdate, error) {
	now := time.Now().Unix()
	for i := range competitionRounds {
		if competitionRounds[i].CreatedAt == 0 {
			competitionRounds[i].CreatedAt = now
		}
		competitionRounds[i].UpdatedAt = now
		competitionRounds[i].Matchup = FormatMatchup(
			competitionRounds[i].CompetitorA,
			competitionRounds[i].CompetitorB,
		)
		err := validate.Struct(competitionRounds[i])
		if err != nil {
			log.Printf("Handler ERROR: Validation failed for round %d: %v", i+1, err)
			return competitionRounds, err
		}
	}
	return competitionRounds, nil
}

func FormatMatchup(competitorA, competitorB string) string {
	return fmt.Sprintf("%s_%s", competitorA, competitorB)
}

// GetMnmOptionsFromContext safely retrieves mnmOptions from context
// Returns an empty map if not found
func GetMnmOptionsFromContext(ctx context.Context) map[string]string {
	if mnmOptions, ok := ctx.Value(MNM_OPTIONS_CTX_KEY).(map[string]string); ok {
		return mnmOptions
	}
	return map[string]string{}
}
