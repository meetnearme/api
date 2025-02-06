package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
)

// TransformFunc defines the signature for transformer functions
type TransformFunc func(map[string]interface{}) (map[string]interface{}, error)

type TimestampInfo struct {
	OriginalTimestamp int64
	UTCTime           string
	LocalTime         string
	Timezone          string
}

// TransformerRegistry holds all available transformers
var TransformerRegistry = map[string]TransformFunc{
	"add_default_end_time_2025_02_06": func(doc map[string]interface{}) (map[string]interface{}, error) {
		if _, endTimeExists := doc["endTime"]; !endTimeExists {
			fmt.Printf("endTime value: %+v", doc["endTime"])
			doc["endTime"] = helpers.DEFAULT_UNDEFINED_END_TIME
		}
		return doc, nil
	},
	"add_competition_config_id_2025_01": func(doc map[string]interface{}) (map[string]interface{}, error) {
		// Check if competitionConfigId already exists
		if _, exists := doc["competitionConfigId"]; !exists {
			// You can set a default value or derive it from other fields
			doc["competitionConfigId"] = "" // or any other default value you prefer

			// Log the addition for debugging
			if name, ok := doc["name"].(string); ok {
				fmt.Printf("Added competitionConfigId for event: %s\n", name)
			}
		}
		return doc, nil
	},
	"name_transformer": func(doc map[string]interface{}) (map[string]interface{}, error) {
		// Check if the "name" field exists and is a string
		if name, ok := doc["name"].(string); ok {
			// Modify the "name" field
			doc["name"] = name + " - transformed"
		}
		return doc, nil
	},
	"timestamp_diagnostic": func(doc map[string]interface{}) (map[string]interface{}, error) {
		startTime, ok := doc["startTime"].(float64)
		if !ok {
			return doc, nil
		}

		timezone, _ := doc["timezone"].(string)
		if timezone == "" {
			timezone = "America/New_York" // default timezone
		}

		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("failed to load timezone %s: %w", timezone, err)
		}

		utcTime := time.Unix(int64(startTime), 0).UTC()
		localTime := time.Unix(int64(startTime), 0).In(loc)

		info := TimestampInfo{
			OriginalTimestamp: int64(startTime),
			UTCTime:           utcTime.Format(time.RFC3339),
			LocalTime:         localTime.Format(time.RFC3339),
			Timezone:          timezone,
		}

		infoBytes, _ := json.MarshalIndent(info, "", " ")
		fmt.Printf("\nTimestamp Diagnostic for event '%v':\n%s\n", doc["name"], string(infoBytes))

		return doc, nil
	},
	"timestamp_correction_2024_12_01": func(doc map[string]interface{}) (map[string]interface{}, error) {
		startTime, ok := doc["startTime"].(float64)
		if !ok {
			return doc, nil
		}

		timezone, _ := doc["timezone"].(string)
		if timezone == "" {
			timezone = "America/New_York"
		}

		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("failed to load timezone %s: %w", timezone, err)
		}

		// Parse the timestamp as if it were in the local timezone
		localTime := time.Unix(int64(startTime), 0)

		// Create a new time using the local components but in the local timezone
		targetTime := time.Date(
			localTime.Year(),
			localTime.Month(),
			localTime.Day(),
			localTime.Hour(),
			localTime.Minute(),
			localTime.Second(),
			localTime.Nanosecond(),
			loc,
		)

		// Now convert to UTC for storage
		utcTime := targetTime.UTC()

		// Store the correct UTC timestamp
		doc["startTime"] = utcTime.Unix()

		// Handle endTime similarly
		if endTime, ok := doc["endTime"].(float64); ok {
			localEndTime := time.Unix(int64(endTime), 0).In(loc)
			targetEndTime := time.Date(
				localEndTime.Year(),
				localEndTime.Month(),
				localEndTime.Day(),
				localEndTime.Hour(),
				localEndTime.Minute(),
				localEndTime.Second(),
				localEndTime.Nanosecond(),
				loc,
			)
			doc["endTime"] = targetEndTime.UTC().Unix()
		}
		// Add this before returning in the transformer
		fmt.Printf("\nEvent: %s\n", doc["name"])
		fmt.Printf("Original timestamp: %v\n", time.Unix(int64(startTime), 0).UTC())
		fmt.Printf("Corrected timestamp: %v\n", utcTime)
		fmt.Printf("Local time: %v\n", targetTime)
		fmt.Printf("Timezone: %s\n", timezone)

		return doc, nil
	},
	"add_missing_ev_type_2024_12_01": func(doc map[string]interface{}) (map[string]interface{}, error) {
		if _, ok := doc["name"].(string); ok && doc["eventSourceType"] == nil {
			doc["eventSourceType"] = "SLF"
		}
		return doc, nil
	},
	// "tensor_weights": func(doc map[string]interface{}) (map[string]interface{}, error) {
	//     // Implement the actual tensor weights transformation
	//     if _, ok := doc["name"].(string); ok {
	//         doc["_tensor_weights"] = map[string]float64{
	//             "name": 0.3,
	//             "description": 0.5,
	//             "address": 0.2,
	//         }
	//     }
	//     return doc, nil
	// },
}
