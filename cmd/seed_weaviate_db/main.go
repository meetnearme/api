package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/types"
)

type MarqoSearchResponse struct {
	Hits []MarqoEvent `json:"hits"`
}

type MarqoEvent struct {
	ID                  string   `json:"_id"`
	EventOwners         []string `json:"eventOwners"`
	EventOwnerName      string   `json:"eventOwnerName"`
	EventSourceType     string   `json:"eventSourceType"`
	CompetitionConfigId string   `json:"competitionConfigId"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	StartTime           int64    `json:"startTime"`
	EndTime             int64    `json:"endTime"`
	Lat                 float64  `json:"lat"`
	Long                float64  `json:"long"`
	Timezone            string   `json:"timezone"`
	Address             string   `json:"address"`
	CreatedAt           int64    `json:"createdAt"`
	UpdatedAt           int64    `json:"updatedAt"`
	Categories          []string `json:"categories"`
}

func TransformMarqoEventToWeaviate(mEvent MarqoEvent) types.Event {
	// Convert the timezone string into a time.Location object.
	loc, err := time.LoadLocation(mEvent.Timezone)
	if err != nil {
		log.Printf("WARN: Could not parse timezone '%s' for event '%s'. Defaulting to UTC. Error: %v", mEvent.Timezone, mEvent.ID, err)
		loc = time.UTC
	}

	// Create a time.Time object from the Unix timestamp and the loaded location.
	startTime := time.Unix(mEvent.StartTime, 0).In(loc)

	// Map fields from the Marqo structure to the Weaviate/internal Event structure.
	// This handles all fields, including optional ones which will have their zero value.
	event := types.Event{
		Id:                  mEvent.ID,
		EventOwners:         mEvent.EventOwners,
		EventOwnerName:      mEvent.EventOwnerName,
		EventSourceType:     mEvent.EventSourceType,
		Name:                mEvent.Name,
		Description:         mEvent.Description,
		StartTime:           mEvent.StartTime,
		EndTime:             mEvent.EndTime,
		Address:             mEvent.Address,
		Lat:                 mEvent.Lat,
		Long:                mEvent.Long,
		Timezone:            *loc,
		Categories:          mEvent.Categories,
		CreatedAt:           mEvent.CreatedAt,
		UpdatedAt:           mEvent.UpdatedAt,
		CompetitionConfigId: mEvent.CompetitionConfigId,
		// All other fields that exist in both MarqoEvent and types.Event
		// can be added here for direct mapping.
	}

	// FIX: Add the missing logic to generate the localized UI fields.
	// We use the time.Time object we created above.
	event.LocalizedStartDate = startTime.Format("Mon, Jan 2")
	event.LocalizedStartTime = startTime.Format("3:04 PM")

	// This business logic is preserved from your original normalization function.
	if apexURL := os.Getenv("APEX_URL"); apexURL != "" {
		event.RefUrl = apexURL + "/event/" + event.Id
	} else {
		log.Println("WARN: APEX_URL environment variable not set. RefUrl will be empty.")
	}

	if event.ImageUrl == "" {
		// Note: Ensure your helpers.GetImgUrlFromHash function can accept a types.Event value.
		// If it requires a pointer, use &event.
		event.ImageUrl = helpers.GetImgUrlFromHash(event)
	}

	return event
}

func main() {
	filePath := flag.String("file", "", "Path to the JSON file containing Marqo events")
	flag.Parse()

	if *filePath == "" {
		log.Println("Error: The --file argument is required.")
		os.Exit(1)
	}

	log.Printf("Starting ingestion from file: %s", *filePath)

	fileBytes, err := os.ReadFile(*filePath)
	if err != nil {
		log.Fatalf("FATAL: Could not read file '%s': %v", *filePath, err)
	}
	var marqoResponse MarqoSearchResponse
	if err := json.Unmarshal(fileBytes, &marqoResponse); err != nil {
		log.Fatalf("FATAL: Could not parse JSON from fil: %v", err)
	}

	marqoEvents := marqoResponse.Hits
	log.Printf("Successfully read %d events from the JSON file.", len(marqoEvents))

	log.Println("Transforming events for Weaviate...")
	eventsToUpsert := make([]types.Event, 0, len(marqoEvents))
	for _, mEvent := range marqoEvents {
		weaviateEvent := TransformMarqoEventToWeaviate(mEvent)
		eventsToUpsert = append(eventsToUpsert, weaviateEvent)
	}

	log.Println("Connecting to Weaviate...")
	client, err := services.GetWeaviateClient()
	if err != nil {
		log.Fatalf("FATAL: Could not get Weaviate client: %v", err)
	}
	log.Println("Successfully connected. Now upserting documents...")

	resp, err := services.BulkUpsertEventsToWeaviate(context.Background(), client, eventsToUpsert)
	if err != nil {
		log.Fatalf("FATAL: Failed to bulk upsert events: %v", err)
	}

	successfulCount := 0
	if resp != nil {
		for _, res := range resp {
			if res.Result != nil && res.Result.Status != nil && *res.Result.Status == "SUCCESS" {
				successfulCount++
			}
		}
	}

	log.Printf("Ingestion complete. Successfully upserted %d out of %d events.", successfulCount, len(eventsToUpsert))
}
