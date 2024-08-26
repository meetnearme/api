package helpers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
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

func ConvertBinaryStringToBase64(binaryStr string) (string, error) {
	// Ensure the binary string length is a multiple of 8
	if len(binaryStr)%8 != 0 {
		return "", fmt.Errorf("binary string length must be a multiple of 8")
	}

	// Convert binary string to byte slice
	byteSlice := make([]byte, len(binaryStr)/8)
	for i := 0; i < len(binaryStr); i += 8 {
		byteVal, err := strconv.ParseUint(binaryStr[i:i+8], 2, 8)
		if err != nil {
			return "", fmt.Errorf("invalid binary string: %v", err)
		}
		byteSlice[i/8] = byte(byteVal)
	}

	// Encode byte slice to base64
	base64Str := base64.StdEncoding.EncodeToString(byteSlice)
	return base64Str, nil
}

func BinaryStringToBinary(s string) ([]byte, error) {
	// Ensure the string length is a multiple of 8
	for len(s)%8 != 0 {
			s = "0" + s
	}

	bytes := make([]byte, len(s)/8)
	for i := 0; i < len(s); i += 8 {
			b, err := strconv.ParseUint(s[i:i+8], 2, 8)
			if err != nil {
					return nil, err
			}
			bytes[i/8] = byte(b)
	}
	return bytes, nil
}

func SetCloudflareKV(key, value string, metadata map[string]string) error {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	namespaceID := os.Getenv("CLOUDFLARE_MNM_SUBDOMAIN_KV_NAMESPACE_ID")

	// First, check if the key already exists
	readURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		accountID, namespaceID, key)

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
		return fmt.Errorf("key already exists in KV store")
	}

	// If the key doesn't exist, proceed with writing
	writeURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		accountID, namespaceID, key)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add value
	_ = writer.WriteField("value", value)

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

	return nil
}
