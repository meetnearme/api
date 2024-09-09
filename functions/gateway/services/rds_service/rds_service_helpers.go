package rds_service

import (
	"log"
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


// func getTime(record map[string]interface{}, key string) time.Time {
//     if val, ok := record[key].(string); ok {
//         t, err := time.Parse("2006-01-02 15:04:05", val)
//         if err != nil {
//             return time.Time{} // Return zero value if parsing fails
//         }
//         return t
//     }
//     return time.Time{} // Return zero value if field is not present
// }

