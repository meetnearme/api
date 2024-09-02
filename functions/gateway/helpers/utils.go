package helpers

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
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
