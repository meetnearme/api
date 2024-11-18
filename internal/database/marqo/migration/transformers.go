package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// TransformFunc defines the signature for transformer functions
type TransformFunc func(map[string]interface{}) (map[string]interface{}, error)

type TimestampInfo struct {
	OriginalTimestamp int
	UTCTime string
	LocalTime string
	Timezone string
}

// TransformerRegistry holds all available transformers
var TransformerRegistry = map[string]TransformFunc{
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
			UTCTime: utcTime.Format(time.RFC3339),
			LocalTime: localTime.Format(time.RFC3339),
			Timezone: timezone,
		}

		infoBytes, _ := json.MarshalIndent(info, "", " ")
		fmt.Printf("\nTimestamp Diagnostic for event '%v':\n%s\n", doc["name"], string(infoBytes))

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
