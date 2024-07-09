package helpers

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
	"unsafe"
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

// NOTE: this is for internal debugging
func PrintContextInternals(ctx interface{}, inner bool) {
	contextValues := reflect.ValueOf(ctx).Elem()
	contextKeys := reflect.TypeOf(ctx).Elem()

	if !inner {
			fmt.Printf("\nFields for %s.%s\n", contextKeys.PkgPath(), contextKeys.Name())
	}

	if contextKeys.Kind() == reflect.Struct {
			for i := 0; i < contextValues.NumField(); i++ {
					reflectValue := contextValues.Field(i)
					reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()

					reflectField := contextKeys.Field(i)

					if reflectField.Name == "Context" {
							PrintContextInternals(reflectValue.Interface(), true)
					} else {
							fmt.Printf("field name: %+v\n", reflectField.Name)
							fmt.Printf("value: %+v\n", reflectValue.Interface())
					}
			}
	} else {
			fmt.Printf("context is empty (int)\n")
	}
}
