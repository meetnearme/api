package services

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/ganeshdipdumbare/marqo-go" // marqo-go is an unofficial Go client library for Marqo
	"github.com/google/uuid"
)

const (
	earthRadiusKm = 6371.0
	milesPerKm    = 0.621371
)
type Event struct {
	Id          string `json:"id"`
	EventOwners []string `json:"eventOwners" validate:"required,min=1"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	StartTime   string `json:"startTime" validate:"required"`
	Address     string `json:"address" validate:"required"`
	Lat    			float64 `json:"lat" validate:"required"`
	Long    		float64 `json:"long" validate:"required"`
}

// considered the best embedding model as of 8/15/2024
var model = "hf/bge-large-en-v1.5"
var indexName = "events-search-index"

func GetMarqoClient() (*marqo.Client, error) {
	// Create a new Marqo client
	var apiBaseUrl string

	if os.Getenv("MARQO_API_BASE_URL") != "" {
		apiBaseUrl = os.Getenv("MARQO_API_BASE_URL")
	} else {
		// set to local host if no marqo lb is set
		apiBaseUrl = "http://localhost:8882"
	}

	// Get the bearer token from an environment variable
	marqoApiKey := os.Getenv("MARQO_API_KEY")

	client, err := marqo.NewClient(apiBaseUrl, marqo.WithMarqoCloudAuth(marqoApiKey))
	if err != nil {
			log.Printf("Error creating marqo client: %v", err)
			return nil, err
	}
	return client, nil
}

// TODO: this is potentially not possible programmatically via Marqo Cloud, open slack
// question to their team here:
// https://marqo-community.slack.com/archives/C03S65BEQC9/p1725456760772439

// Default index settings JSON for index creation at
// https://cloud.marqo.ai/indexes/create/

// Our first instance: https://events-search-index-di32q8-g2amp25x.dp1.marqo.ai

// {
//   "type": "structured",
//   "vectorNumericType": "float",
//   "model": "hf/bge-large-en-v1.5",
//   "normalizeEmbeddings": true,
//   "textPreprocessing": {
//     "splitLength": 2,
//     "splitOverlap": 0,
//     "splitMethod": "sentence"
//   },
//   "imagePreprocessing": {
//     "patchMethod": null
//   },
//   "annParameters": {
//     "spaceType": "prenormalized-angular",
//     "parameters": {
//       "efConstruction": 512,
//       "m": 16
//     }
//   },
//   "tensorFields": ["name_description_address"],
//   "allFields": [
//    {
//      "name": "name_description_address",
//      "type": "multimodal_combination",
//      "dependentFields": {"name": 0.3, "address": 0.2, "description": 0.5}
//    },
//    {"name": "eventOwners", "type": "array<text>", "features": ["filter"]},
//    {"name": "tags", "type": "array<text>", "features": ["filter", "lexical_search"]},
//    {"name": "categories", "type": "array<text>", "features": ["filter", "lexical_search"]},
// 		{"name": "eventSourceId", "type": "text"},
// 		{"name": "eventSourceType", "type": "text"},
// 		{"name": "name", "type": "text", "features": ["lexical_search"]},
// 		{"name": "description", "type": "text", "features": ["lexical_search"]},
// 		{"name": "startTime", "type": "text", "features": ["lexical_search"]},
// 		{"name": "endTime", "type": "text", "features": ["lexical_search"]},
// 		{"name": "recurrenceRule", "type": "text", "features": ["lexical_search"]},
// 		{"name": "hasRegistrationFields", "type": "text", "features": ["lexical_search"]},
// 		{"name": "hasPurchasable", "type": "text", "features": ["lexical_search"]},
// 		{"name": "imageUrl", "type": "text"},
// 		{"name": "lat", "type": "double", "features": ["filter"]},
// 		{"name": "long", "type": "double", "features": ["filter"]},
// 		{"name": "address", "type": "text", "features": ["lexical_search", "filter"]},
// 		{"name": "sourceUrl", "type": "text"},
// 		{"name": "createdAt", "type": "text"},
// 		{"name": "updatedAt", "type": "text"},
// 		{"name": "updatedBy", "type": "text"}
//   ]
// }


// NOTE: it's possible to programatically create index, but this is an expensive
// mistake to do programmatically, as each index costs at minimum ~$250 / mo
// hence we're commenting this out for now

// func CreateMarqoIndex(client *marqo.Client) (*marqo.CreateIndexResponse, error) {
// 	// Create a new index
// 	req := marqo.CreateIndexRequest{
// 		IndexName: indexName,
// 		IndexDefaults: &marqo.IndexDefaults{
// 			Model: &model,
// 		},
// 	}
// 	res, err := client.CreateIndex(&req)
// 	if err != nil {
// 		log.Printf("Error creating index: %v", err)
// 		return nil, err
// 	}
// 	return res, nil
// }

func UpsertEventToMarqo(client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
	// Insert an event

	events := []Event{event}
	return BulkUpsertEventToMarqo(client, events)
}

func BulkUpsertEventToMarqo(client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error) {
	// Bulk upsert multiple events
	var documents []interface{}
	for _, event := range events {
		_uuid := uuid.NewString()
		document := map[string]interface{}{
			"_id": 			_uuid,
			"eventOwners": event.EventOwners,
			"name":        event.Name,
			"description": event.Description,
			"startTime":    event.StartTime,
			"address":     event.Address,
			"lat":    float64(event.Lat),
			"long":   float64(event.Long),
		}
		documents = append(documents, document)
	}
	req := marqo.UpsertDocumentsRequest{
		Documents: documents,
		IndexName: indexName,
	}
	res, err := client.UpsertDocuments(&req)

	if err != nil {
		log.Printf("Error upserting events: %v", err)
		return nil, err
	}

	return res, nil
}

// SearchMarqoEvents searches for events based on the given query, user location, and maximum distance.
// It returns a list of events that match the search criteria.
// EX : SearchMarqoEvents(client, "music", []float64{37.7749, -122.4194}, 10)
func SearchMarqoEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64, ownerIds []string) ([]Event, error) {
	// Calculate the maximum and minimum latitude and longitude based on the user's location and maximum distance
	maxLat := userLocation[0] + miToLat(maxDistance)
	maxLong := userLocation[1] + miToLong(maxDistance, userLocation[0])
	minLat := userLocation[0] - miToLat(maxDistance)
	minLong := userLocation[1] - miToLong(maxDistance, userLocation[0])

	// Search for events based on the query
	searchMethod := "HYBRID"
	var ownerFilter string
	if len(ownerIds) > 0 {
		ownerFilter = fmt.Sprintf("eventOwners IN (%s) AND ", strings.Join(ownerIds, ","))
	}
	filter := fmt.Sprintf("%s long:[* TO %f] AND long:[%f TO *] AND lat:[* TO %f] AND lat:[%f TO *]", ownerFilter, maxLong, minLong, maxLat, minLat)
	searchRequest := marqo.SearchRequest{
		IndexName:    indexName,
		Q:            &query,
		SearchMethod: &searchMethod,
		Filter:       &filter,
		HybridParameters: &marqo.HybridParameters {
			RetrievalMethod: "disjunction",
			RankingMethod:   "rrf",
		},
	}

	searchResp, err := client.Search(&searchRequest)

	if err != nil {
		log.Printf("Error searching documents: %v", err)
		return nil, err
	}
	// Extract the events from the search response
	var events []Event
	for _, doc := range searchResp.Hits {
		// log.Printf("Event: %v", doc)
		event := Event{
			Id:          getString(doc, "_id"),
			EventOwners: getStringSlice(doc, "eventOwners"),
			Name:        getString(doc, "name"),
			Description: getString(doc, "description"),
			StartTime:   getString(doc, "startTime"),
			Address:     getString(doc, "address"),
			Lat:    getFloat64(doc, "lat"),
			Long:   getFloat64(doc, "long"),
		}
		events = append(events, event)
	}

	return events, nil
}

func GetMarqoEventByID(client *marqo.Client, docId string) (Event, error) {
	docIds := []string{docId}
	events, err := BulkGetMarqoEventByID(client, docIds)
	if err != nil {
		log.Printf("Error getting event by id: %v", err)
		return Event{}, err
	}
	if len(events) == 0 {
		return Event{}, fmt.Errorf("no event found with id: %s", docId)
	}
	event := events[0]
	return event, nil
}

func BulkGetMarqoEventByID(client *marqo.Client, docIds []string) ([]Event, error) {

	getDocumentsReq := &marqo.GetDocumentsRequest{
		IndexName: indexName,
		DocumentIDs: docIds,
	}
	res, err := client.GetDocuments(getDocumentsReq)
	if err != nil {
	    log.Printf("Failed to get documents: %v", err)
	}
	var events []Event
	for _, result := range res.Results {
		event := Event{
			Id:          getString(result, "_id"),
			EventOwners: getStringSlice(result, "eventOwners"),
			Name:        getString(result, "name"),
			Description: getString(result, "description"),
			// TODO: These need converting to unix and a new index deployed
			// in order to support marqo's unix search time range query
			StartTime:   getString(result, "startTime"),
			Address:     getString(result, "address"),
			Lat:    getFloat64(result, "lat"),
			Long:   getFloat64(result, "long"),
		}
		events = append(events, event)
	}
	return events, nil
}

// miToLat converts miles to latitude offset
func miToLat(mi float64) float64 {
	return (mi * milesPerKm) / earthRadiusKm * (180 / math.Pi)
}

// miToLong converts kilometers to longitude
func miToLong(mi float64, lat float64) float64 {
	return (mi * milesPerKm) / (earthRadiusKm * math.Cos(lat*math.Pi/180)) * (180 / math.Pi)
}

func getString(doc map[string]interface{}, key string) string {
	if value, ok := doc[key]; ok && value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(doc map[string]interface{}, key string) float64 {
	if value, ok := doc[key]; ok && value != nil {
		switch v := value.(type) {
		case float64:
			return float64(v)
		}
	}
	return 0
}

func getStringSlice(doc map[string]interface{}, key string) []string {
	if value, ok := doc[key]; ok && value != nil {
		if slice, ok := value.([]interface{}); ok {
			var result []string
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}

