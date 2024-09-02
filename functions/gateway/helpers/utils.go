package helpers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

func FormatDate(d string) string {
	date, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return "Invalid date"
	}
	return date.Format("Jan 2, 2006 (Mon)")
}

func FormatTime(t string) string {
	_time, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return "Invalid time"
	}
	return _time.Format("3:04pm")
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

func HashIDtoImgRange(id string) int {
	hash := md5.Sum([]byte(id))
	hashInt := int(hash[0]) % 8
	return hashInt
}

func GetImgUrlFromHash(id string) string {
	return os.Getenv("STATIC_BASE_URL")+"/assets/img/"+fmt.Sprint(HashIDtoImgRange(id)) + ".png"
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

func DeleteCloudflareKV (subdomainValue, userID string) error {
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

func GetUserMetadataByKey (userID, key string) (string, error) {
	url := fmt.Sprintf("https://%s/management/v1/users/%s/metadata/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID, key)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer " + os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

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

	return respData["metadata"].(map[string]interface{})["value"].(string), nil
}

func UpdateUserMetadataKey (userID, key, value string) error {
	url := fmt.Sprintf("https://%s/management/v1/users/%s/metadata/%s", os.Getenv("ZITADEL_INSTANCE_HOST"), userID, key)
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
	req.Header.Add("Authorization", "Bearer " + os.Getenv("ZITADEL_BOT_ADMIN_TOKEN"))

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
