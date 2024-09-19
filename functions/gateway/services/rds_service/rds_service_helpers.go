package rds_service

import (
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

func getString(record map[string]interface{}, key string) string {
    if val, ok := record[key].(string); ok {
        return val
    }
    return ""
}

func getOptionalString(record map[string]interface{}, key string) *string {
    if val, ok := record[key].(string); ok {
        return &val
    }
    return nil
}

func getTime(record map[string]interface{}, key string) time.Time {
    if val, ok := record[key].(string); ok {
        // Adjust the format string to include fractional seconds
        t, err := time.Parse("2006-01-02 15:04:05.999999", val)
        if err != nil {
            log.Printf("Error parsing time for key %s: %v", key, err)
            return time.Time{} // Return zero value if parsing fails
        }
        return t
    }
    return time.Time{} // Return zero value if field is not present
}

func getFloat64(record map[string]interface{}, key string) float64 {
    if val, ok := record[key].(float64); ok {
        return val
    } else if val, ok := record[key].(string); ok {
        // Handle case where value is a string but represents a number
        if parsedVal, err := strconv.ParseFloat(val, 64); err == nil {
            return parsedVal
        } else {
            log.Printf("Error parsing float64 from string for key %s: %v", key, err)
        }
    }
    return 0.0 // Return 0.0 if value is not present or not a float64
}

// getInt64 retrieves an int64 value from a record. It handles cases where the value is an int64 or a string representing an integer.
func getInt64(record map[string]interface{}, key string) int64 {
    if val, ok := record[key].(int64); ok {
        return val
    } else if val, ok := record[key].(string); ok {
        // Handle case where value is a string but represents an integer
        if parsedVal, err := strconv.ParseInt(val, 10, 64); err == nil {
            return parsedVal
        } else {
            log.Printf("Error parsing int64 from string for key %s: %v", key, err)
        }
    }
    return 0 // Return 0 if value is not present or not an int64
}

func buildArrayString(array []string) []*rds_types.FieldMemberStringValue {
    var result []*rds_types.FieldMemberStringValue
    for _, val := range array {
        result = append(result, &rds_types.FieldMemberStringValue{Value: val})
    }
    return result
}

// Retrieves a comma-separated string from the record and converts it to a slice of strings
func getStringSlice(record map[string]interface{}, key string) []string {
    if val, ok := record[key].(string); ok {
        // Split the comma-separated string into a slice of strings
        return strings.Split(val, ",")
    }
    return nil
}

// Converts a slice of strings into a comma-separated string
func sliceToCommaSeparatedString(slice []string) string {
    return strings.Join(slice, ",")
}

// Converts a comma-separated string into a slice of strings
func commaSeparatedStringToSlice(commaStr string) []string {
    if commaStr == "" {
        return []string{} // Return an empty slice if the string is empty
    }
    return strings.Split(commaStr, ",")
}


func parseTime(value string, t *testing.T) time.Time {
    layout := "2006-01-02 15:04:05" // RDS SQL accepted time format
    parsedTime, err := time.Parse(layout, value)
    if err != nil {
        t.Fatalf("error parsing time for key: %s, error: %v", value, err)
    }
    return parsedTime
}
