package services

import (
	"fmt"
	"log"
	"math"

	"github.com/ganeshdipdumbare/marqo-go" // marqo-go is an unofficial Go client library for Marqo
	"github.com/meetnearme/api/functions/gateway/helpers"
)

type Event struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Datetime    string  `json:"datetime"`
	Address     string  `json:"address"`
	ZipCode     string  `json:"zip_code"`
	Country     string  `json:"country" validate:"required"`
	Latitude    float32 `json:"latitude" validate:"required"`
	Longitude   float32 `json:"longitude" validate:"required"`
}

var marqoLbName = helpers.GetMarqoLB()

// considered the best embedding model as of 8/15/2024
var model = "hf/bge-large-en-v1.5"
var indexName = "search-index"

func GetMarqoClient() (*marqo.Client, error) {
	// Create a new Marqo client
	var creatString string

	if marqoLbName == "" {
		creatString = "http://localhost:8882" //set to local host if no marqo lb is set
	} else {
		creatString = marqoLbName
	}
	client, err := marqo.NewClient(creatString)
	if err != nil {
		log.Printf("Error creating marqo client: %v", err)
		return nil, err
	}
	return client, nil
}

func CreateMarqoIndex(client *marqo.Client) (*marqo.CreateIndexResponse, error) {
	// Create a new index
	req := marqo.CreateIndexRequest{
		IndexName: indexName,
		IndexDefaults: &marqo.IndexDefaults{
			Model: &model,
		},
	}
	res, err := client.CreateIndex(&req)
	if err != nil {
		log.Printf("Error creating index: %v", err)
		return nil, err
	}
	return res, nil
}

func UpsertEventToMarqo(client *marqo.Client, event Event) (*marqo.UpsertDocumentsResponse, error) {
	// Insert an event

	events := []Event{event}
	return BulkUpsertEventToMarqo(client, events)
}

func BulkUpsertEventToMarqo(client *marqo.Client, events []Event) (*marqo.UpsertDocumentsResponse, error) {
	// Bulk upsert multiple events

	var documents []interface{}
	for _, event := range events {
		document := map[string]interface{}{
			"ID": event.Id,
			"Fields": map[string]interface{}{
				"name":        event.Name,
				"description": event.Description,
				"datetime":    event.Datetime,
				"address":     event.Address,
				"zip_code":    event.ZipCode,
				"country":     event.Country,
				"latitude":    event.Latitude,
				"longitude":   event.Longitude,
			},
			"Mappings": map[string]interface{}{
				"name_description_address": map[string]interface{}{
					"type": "multimodal_combination",
					"weights": map[string]float64{
						"description": 0.5,
						"address":     0.2,
						"name":        0.3,
					},
				},
			},
			"TensorFields": []string{
				"name_description_address",
			},
			"Index": indexName,
		}
		documents = append(documents, document)
	}

	req := marqo.UpsertDocumentsRequest{
		Documents: documents,
	}
	err, res := client.UpsertDocuments(&req)
	if err != nil {
		log.Printf("Error upserting events: %v", err)
		return err, nil
	}
	fmt.Printf("UpsertDocumentsResponse: %+v\n", res)
	return nil, res
}

// SearchMarqoEvents searches for events based on the given query, user location, and maximum distance.
// It returns a list of events that match the search criteria.
// EX : SearchMarqoEvents(client, "music", []float64{37.7749, -122.4194}, 10)
func SearchMarqoEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64) ([]Event, error) {
	// Calculate the maximum and minimum latitude and longitude based on the user's location and maximum distance
	maxLat := userLocation[0] + kmToLat(maxDistance)
	maxLong := userLocation[1] + kmToLong(maxDistance, userLocation[0])
	minLat := userLocation[0] - kmToLat(maxDistance)
	minLong := userLocation[1] - kmToLong(maxDistance, userLocation[0])

	// Search for events based on the query
	searchMethod := "HYBRID"
	filter := fmt.Sprintf("long:[* TO %f] AND long:[%f TO *] AND lat:[* TO %f] AND lat:[%f TO *]", maxLong, minLong, maxLat, minLat)

	searchRequest := marqo.SearchRequest{
		Index:        &indexName,
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
		event := Event{
			Id:          doc["ID"].(string),
			Name:        doc["name"].(string),
			Description: doc["description"].(string),
			Datetime:    doc["datetime"].(string),
			Address:     doc["address"].(string),
			ZipCode:     doc["zip_code"].(string),
			Latitude:    float32(doc["latitude"].(float64)),
			Longitude:   float32(doc["longitude"].(float64)),
		}
		events = append(events, event)
	}
	return events, nil
}

// kmToLat converts kilometers to latitude
func kmToLat(km float64) float64 {
	return km / 110.574
}

// kmToLong converts kilometers to longitude
func kmToLong(km float64, latitude float64) float64 {
	return km / (math.Cos(latitude*math.Pi/180) * 6371)
}
