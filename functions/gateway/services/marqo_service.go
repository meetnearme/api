package services

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/ganeshdipdumbare/marqo-go" // marqo-go is an unofficial Go client library for Marqo
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

const (
	earthRadiusKm = 6371.0
	milesPerKm    = 0.621371
)

// considered the best embedding model as of 8/15/2024
// var model = "hf/bge-large-en-v1.5"

func GetMarqoIndexName () string {
	sstStage := os.Getenv("SST_STAGE")
	if sstStage == "prod" {
		return os.Getenv("PROD_MARQO_INDEX_NAME")
	} else {
		return os.Getenv("DEV_MARQO_INDEX_NAME")
	}

}

func GetMarqoClient() (*marqo.Client, error) {
	// Create a new Marqo client
	var apiBaseUrl string

	sstStage := os.Getenv("SST_STAGE")
	if sstStage == "prod" {
		apiBaseUrl = os.Getenv("PROD_MARQO_API_BASE_URL")
	// IMPORTANT: This assumes we don't set `SST_STAGE`
	// in unit tests, we assume this is a non-prod deployment
	} else if sstStage != "" {
		apiBaseUrl = os.Getenv("DEV_MARQO_API_BASE_URL")
	} else if os.Getenv("GO_ENV") == helpers.GO_TEST_ENV {
		apiBaseUrl = os.Getenv("DEV_MARQO_API_BASE_URL")
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
//     "splitMethod": "sentence",
//     "splitOverlap": 0
//   },
//   "imagePreprocessing": {},
//   "annParameters": {
//     "spaceType": "prenormalized-angular",
//     "parameters": {
//       "efConstruction": 512,
//       "m": 16
//     }
//   },
//   "tensorFields": [
//     "name_description_address"
//   ],
//   "allFields": [
//     {
//       "name": "name_description_address",
//       "features": [],
//       "type": "multimodal_combination",
//       "dependentFields": {
//         "name": 0.3,
//         "description": 0.5,
//         "address": 0.2
//       }
//     },
//     {
//       "name": "eventOwners",
//       "type": "array<text>",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "tags",
//       "type": "array<text>",
//       "features": [
//         "filter",
//         "lexical_search"
//       ]
//     },
//     {
//       "name": "categories",
//       "type": "array<text>",
//       "features": [
//         "filter",
//         "lexical_search"
//       ]
//     },
//     {
//       "name": "eventSourceId",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "eventSourceType",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "name",
//       "type": "text",
//       "features": [
//         "lexical_search"
//       ]
//     },
//     {
//       "name": "description",
//       "type": "text",
//       "features": [
//         "lexical_search"
//       ]
//     },
//     {
//       "name": "startTime",
//       "type": "long",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "endTime",
//       "type": "long",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "recurrenceRule",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "hasRegistrationFields",
//       "type": "bool",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "hasPurchasable",
//       "type": "bool",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "startingPrice",
//       "type": "int",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "currency",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "payeeId",
//       "type": "text",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "imageUrl",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "lat",
//       "type": "double",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "long",
//       "type": "double",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "timezone",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "address",
//       "type": "text",
//       "features": [
//         "lexical_search",
//         "filter"
//       ]
//     },
//     {
//       "name": "sourceUrl",
//       "type": "text",
//       "features": []
//     },
//     {
//       "name": "createdAt",
//       "type": "long",
//       "features": []
//     },
//     {
//       "name": "updatedAt",
//       "type": "long",
//       "features": []
//     },
//     {
//       "name": "updatedBy",
//       "type": "text",
//       "features": []
//     }
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

func ConvertEventsToDocuments(events []types.Event, hasIds bool) (documents []interface{}) {
	for _, event := range events {
		var _uuid string
		if !hasIds {
			_uuid = uuid.NewString()
		} else {
			_uuid = event.Id
		}

		document := map[string]interface{}{
			"_id":         _uuid,
			"eventOwners": event.EventOwners,
			"name":        event.Name,
			"description": event.Description,
			"startTime":   int64(event.StartTime),
			"address":     event.Address,
			"lat":         float64(event.Lat),
			"long":        float64(event.Long),
			"timezone":    event.Timezone,
		}

		// Add optional fields only if they are not nil
		if event.EndTime != 0 {
			document["endTime"] = int64(event.EndTime)
		}
		if event.StartingPrice != 0 {
			document["startingPrice"] = int32(event.StartingPrice)
		}
		if event.Currency != "" {
			document["currency"] = string(event.Currency)
		}
		if event.PayeeId != "" {
			document["payeeId"] = string(event.PayeeId)
		}
		if event.HasRegistrationFields != false {
			document["hasRegistrationFields"] = bool(event.HasRegistrationFields)
		}
		if event.HasPurchasable != false {
			document["hasPurchasable"] = bool(event.HasPurchasable)
		}
		if event.ImageUrl != "" {
			document["imageUrl"] = string(event.ImageUrl)
		}
		if len(event.Categories) > 0 {
			document["categories"] = []string(event.Categories)
		}
		if len(event.Tags) > 0 {
			document["categories"] = []string(event.Tags)
		}
		if event.CreatedAt != 0 {
			document["createdAt"] = int64(event.CreatedAt)
		}
		if event.UpdatedAt != 0 {
			document["updatedAt"] = int64(event.UpdatedAt)
		}
		if event.UpdatedBy != "" {
			document["updatedBy"] = string(event.UpdatedBy)
		}

		documents = append(documents, document)
	}

	return documents
}

func BulkUpsertEventToMarqo(client *marqo.Client, events []types.Event, hasIds bool) (*marqo.UpsertDocumentsResponse, error) {
	// Bulk upsert multiple events
	documents := ConvertEventsToDocuments(events, hasIds)
	indexName := GetMarqoIndexName()
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
func SearchMarqoEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64, startTime, endTime int64, ownerIds []string, categories string) (types.EventSearchResponse, error) {
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
	if query != "" {
		query = "keywords: { " + query + " }"
	}

	if categories != "" {
		query = query + " {show matches for these categories(" + categories + ")}"
	}

	filter := fmt.Sprintf("%s startTime:[%v TO %v] AND long:[* TO %f] AND long:[%f TO *] AND lat:[* TO %f] AND lat:[%f TO *]", ownerFilter, startTime, endTime, maxLong, minLong, maxLat, minLat)
	indexName := GetMarqoIndexName()
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
		return types.EventSearchResponse{
			Query:  query,
			Filter: filter,
			Events: []types.Event{},
		}, err
	}
	// Extract the events from the search response
	var events []types.Event
	for _, doc := range searchResp.Hits {
		event := NormalizeMarqoDocOrSearchRes(doc)
		if event != nil {
			events = append(events, *event)
		}
	}

	return types.EventSearchResponse{
		Query: query,
		Filter: filter,
		Events: events,
	}, nil
}

func BulkGetMarqoEventByID(client *marqo.Client, docIds []string) ([]*types.Event, error) {
	indexName := GetMarqoIndexName()
	getDocumentsReq := &marqo.GetDocumentsRequest{
		IndexName: indexName,
		DocumentIDs: docIds,
	}
	res, err := client.GetDocuments(getDocumentsReq)
	if err != nil {
		log.Printf("Failed to get documents: %v", err)
		return nil, err
	}

	// Check if no documents were found
	if len(res.Results) == 1 && res.Results[0]["_found"] == false {
		log.Printf("No documents found for the given IDs")
		return []*types.Event{}, nil
	}

	var events []*types.Event

	for _, result := range res.Results {
		event := NormalizeMarqoDocOrSearchRes(result, )
		events = append(events, event)
	}
	return events, nil
}

func GetMarqoEventByID(client *marqo.Client, docId string) (*types.Event, error) {
	docIds := []string{docId}
	events, err := BulkGetMarqoEventByID(client, docIds)
	if err != nil {
		log.Printf("Error getting event by id: %v", err)
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no event found with id: %s", docId)
	}
	return events[0], nil
}

func BulkUpdateMarqoEventByID(client *marqo.Client, events []types.Event) (*marqo.UpsertDocumentsResponse, error) {
	// Validate that each event has an ID
	for i, event := range events {
		if event.Id == "" {
			return nil, fmt.Errorf("event at index %d is missing an ID", i)
		}
	}

	// If all events have IDs, proceed with the bulk upsert
	return BulkUpsertEventToMarqo(client, events, true)
}

func NormalizeMarqoDocOrSearchRes (doc map[string]interface{}) (event *types.Event) {
	// NOTE: seems to be a bug in Go that instantiates these `int64` values as
	// `float64` when they are parsed / marshalled
	startTimeFloat := getValue[float64](doc, "startTime")
	startTimeInt := int64(startTimeFloat)

	event = &types.Event{
		Id:          getValue[string](doc, "_id"),
		EventOwners: getStringSlice(doc, "eventOwners"),
		Name:        getValue[string](doc, "name"),
		Description: getValue[string](doc, "description"),
		StartTime:   startTimeInt,
		Address:     getValue[string](doc, "address"),
		Lat:         getValue[float64](doc, "lat"),
		Long:        getValue[float64](doc, "long"),
		Timezone:    getValue[string](doc, "timezone"),
		Categories:  getStringSlice(doc, "categories"),
		Tags: 			 getStringSlice(doc, "tags"),
	}

  // Handle optional fields
	optionalFields := []struct {
		key      string
		setField func()
		}{
			{"endTime", func() {
				if v := getValue[*float64](doc, "endTime"); v != nil {
						endTime := int64(*v)
						event.EndTime = endTime
				}
		}},
		{"startingPrice", func() {
				if v := getValue[float64](doc, "startingPrice"); v != 0 {
						startingPrice := int32(v)
						event.StartingPrice = startingPrice
				}
		}},
		{"currency", func() {
			if v := getValue[string](doc, "currency"); v != "" {
					event.Currency = v
			}
		}},
		{"payeeId", func() {
				if v := getValue[string](doc, "payeeId"); v != "" {
						event.PayeeId = v
				}
		}},
		{"hasRegistrationFields", func() {
				if v := getValue[bool](doc, "hasRegistrationFields"); v != false {
						event.HasRegistrationFields = v
				}
		}},
		{"hasPurchasable", func() {
				if v := getValue[bool](doc, "hasPurchasable"); v != false {
						event.HasPurchasable = v
				}
		}},
		{"imageUrl", func() {
				if v := getValue[string](doc, "imageUrl"); v != "" {
						event.ImageUrl = v
				}
		}},
		{"categories", func() {
			if v := getValue[[]string](doc, "categories"); v != nil {
					event.Categories = v
			}
		}},
		{"tags", func() {
			if v := getValue[[]string](doc, "tags"); v != nil {
					event.Tags = v
			}
		}},
		{"createdAt", func() {
				if v := getValue[float64](doc, "createdAt"); v != 0 {
						createdAt := int64(v)
						event.CreatedAt = createdAt
				}
		}},
		{"updatedAt", func() {
				if v := getValue[float64](doc, "updatedAt"); v != 0 {
						updatedAt := int64(v)
						event.UpdatedAt = updatedAt
				}
		}},
		{"updatedBy", func() {
				if v := getValue[string](doc, "updatedBy"); v != "" {
						event.UpdatedBy = v
				}
		}},
	}

	for _, field := range optionalFields {
		if value, ok := doc[field.key]; ok && value != nil {
			field.setField()
		}
	}

	// NOTE: this is a hack for Adalo, always sending `refUrl` for a link out
	// to the platform event
	event.RefUrl = os.Getenv("APEX_URL") + "/events/" + event.Id

	// NOTE: this is also a hack for Adalo
	if event.ImageUrl == "" {
		event.ImageUrl = helpers.GetImgUrlFromHash(*event)
	}

	return event
}

// miToLat converts miles to latitude offset
func miToLat(mi float64) float64 {
	return (mi * milesPerKm) / earthRadiusKm * (180 / math.Pi)
}

// miToLong converts kilometers to longitude
func miToLong(mi float64, lat float64) float64 {
	return (mi * milesPerKm) / (earthRadiusKm * math.Cos(lat*math.Pi/180)) * (180 / math.Pi)
}

func getValue[T string | *string | []string | *[]string | float64 | *float64 | int64 | *int64 | int32 | *int32 | bool | *bool](doc map[string]interface{}, key string) T {
	if value, ok := doc[key]; ok && value != nil {
			switch any((*new(T))).(type) {
			case string:
					if str, ok := value.(string); ok {
							return any(str).(T)
					}
			case *string:
					if str, ok := value.(string); ok {
							return any(&str).(T)
					}
			case []string:
				if slice, ok := value.([]interface{}); ok {
						result := make([]string, 0, len(slice))
						for _, item := range slice {
								if str, ok := item.(string); ok {
										result = append(result, str)
								}
						}
						return any(result).(T)
				}
			case *[]string:
				if slice, ok := value.([]interface{}); ok {
						result := make([]string, 0, len(slice))
						if slice == nil {
							return any(result).(T)
						}
						for _, item := range slice {
								if str, ok := item.(string); ok {
										result = append(result, str)
								}
						}
						return any(result).(T)
				}
			case float64:
					if f, ok := value.(float64); ok {
							return any(f).(T)
					}
			case *float64:
					if f, ok := value.(float64); ok {
							return any(&f).(T)
					}
			case int64:
					if i, ok := value.(float64); ok {
							return any(int64(i)).(T)
					}
			case *int64:
					if i, ok := value.(float64); ok {
							i64 := int64(i)
							return any(&i64).(T)
					}
			case int32:
					if i, ok := value.(float64); ok {
							return any(int32(i)).(T)
					}
			case *int32:
					if i, ok := value.(float64); ok {
							i32 := int32(i)
							return any(&i32).(T)
					}
			case bool:
					if b, ok := value.(bool); ok {
							return any(b).(T)
					}
			case *bool:
					if b, ok := value.(bool); ok {
							return any(&b).(T)
					}
			}
			log.Printf("key: %s, Unexpected Type: %T, Value: %v", key, value, value)
	}
	var zero T
	return zero
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

