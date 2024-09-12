package rds_service

import (
	"log"
	"strconv"
	"time"
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
