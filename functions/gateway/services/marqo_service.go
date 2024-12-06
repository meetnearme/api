package services

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

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

func GetMarqoIndexName() string {
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
//       "dependentFields": {
//         "name": 0.3,
//         "eventOwnerName": 0.1,
//         "description": 0.5,
//         "address": 0.2
//       },
//       "type": "multimodal_combination"
//     },
//     {
//       "name": "eventOwners",
//       "type": "array<text>",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "eventOwnerName",
//       "type": "text",
//       "features": [
//         "lexical_search"
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
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "eventSourceType",
//       "type": "text",
//       "features": [
//         "filter"
//       ]
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
//         "filter",
//         "score_modifier"
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
//       "name": "hideCrossPromo",
//       "type": "bool",
//       "features": []
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
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "createdAt",
//       "type": "long",
//       "features": [
//         "filter"
//       ]
//     },
//     {
//       "name": "updatedAt",
//       "type": "long",
//       "features": [
//         "filter"
//       ]
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
	now := time.Now().Unix()
	createdAt := now
	updatedAt := now
	for _, event := range events {
		var _uuid string
		if !hasIds {
			_uuid = uuid.NewString()
		} else {
			_uuid = event.Id
		}
		if event.CreatedAt > 0 {
			createdAt = event.CreatedAt
		}
		document := map[string]interface{}{
			"_id":            _uuid,
			"eventOwners":    event.EventOwners,
			"eventOwnerName": event.EventOwnerName,
			"name":           event.Name,
			"description":    event.Description,
			"startTime":      int64(event.StartTime),
			"address":        event.Address,
			"lat":            float64(event.Lat),
			"long":           float64(event.Long),
			"timezone":       event.Timezone,
			"createdAt":      createdAt,
			"updatedAt":      updatedAt,
		}

		// Add optional fields only if they are not nil
		if event.EventSourceType != "" {
			document["eventSourceType"] = string(event.EventSourceType)
		}
		if event.EventSourceId != "" {
			document["eventSourceId"] = string(event.EventSourceId)
		}
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
		if event.HideCrossPromo {
			document["hideCrossPromo"] = bool(event.HideCrossPromo)
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
		Mappings: map[string]interface{}{
			"name_description_address": map[string]interface{}{
				"type": "multimodal_combination",
				"weights": map[string]float64{
					"description":    0.5,
					"address":        0.2,
					"name":           0.3,
					"eventOwnerName": 0.1,
				},
			},
		},
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
func SearchMarqoEvents(client *marqo.Client, query string, userLocation []float64, maxDistance float64, startTime, endTime int64, ownerIds []string, categories string, address string, parseDates string, eventSourceTypes []string, eventSourceIds []string) (types.EventSearchResponse, error) {
	// Calculate the maximum and minimum latitude and longitude based on the user's location and maximum distance
	maxLat := userLocation[0] + miToLat(maxDistance)
	maxLong := userLocation[1] + miToLong(maxDistance, userLocation[0])
	minLat := userLocation[0] - miToLat(maxDistance)
	minLong := userLocation[1] - miToLong(maxDistance, userLocation[0])

	// Search for events based on the query
	searchMethod := "HYBRID"
	// TODO: parameterize this
	limit := 100

	var ownerFilter string
	if len(ownerIds) > 0 {
		ownerFilter = fmt.Sprintf("eventOwners IN (%s) AND ", strings.Join(ownerIds, ","))
	}

	var addressFilter string
	if address != "" {
		addressFilter = fmt.Sprintf("address:(%s) AND ", address)
	}

	if query != "" {
		query = "keywords: { " + query + " }"
	}

	if categories != "" {
		query = query + " {show matches for these categories(" + categories + ")}"
	}

	var eventSourceTypeFilter string
	var eventSourceIdFilter string

	// Build eventSourceType filter using Marqo IN operator
	if len(eventSourceTypes) > 0 {
		eventSourceTypeFilter = fmt.Sprintf("eventSourceType IN (%s) AND ", strings.Join(eventSourceTypes, ", "))
	} else {
		eventSourceTypeFilter = fmt.Sprintf("eventSourceType IN (%s) AND ", strings.Join(helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES, ", "))
	}

	// Build eventSourceId filter using Marqo IN operator
	if len(eventSourceIds) > 0 {
		eventSourceIdFilter = fmt.Sprintf("eventSourceId IN (%s) AND ", strings.Join(eventSourceIds, ","))
	}

	// Update the filter string construction to include the new filters
	filter := fmt.Sprintf("%s %s %s %s startTime:[%v TO %v] AND long:[* TO %f] AND long:[%f TO *] AND lat:[* TO %f] AND lat:[%f TO *]",
		addressFilter,
		ownerFilter,
		eventSourceTypeFilter,
		eventSourceIdFilter,
		startTime,
		endTime,
		maxLong,
		minLong,
		maxLat,
		minLat,
	)

	indexName := GetMarqoIndexName()

	searchRequest := marqo.SearchRequest{
		IndexName:    indexName,
		Q:            &query,
		SearchMethod: &searchMethod,
		Filter:       &filter,
		Limit:        &limit,
		// TODO: this is missing from the marqo Go client, we should add

		// SearchableAttributesTensor: []string{
		//   "eventOwnerName",
		//   "name",
		//   "description",
		//   "address",
		//   "categories",
		//   "tags",
		// },
		HybridParameters: &marqo.HybridParameters{
			RetrievalMethod: "disjunction",
			RankingMethod:   "rrf",
			// NOTE: none of these seemed to have much influence in
			// testing around the time of initial launch, should be
			// revisited and better understood

			// ScoreModifiersLexical: &marqo.ScoreModifiers{
			//	"multiply_score_by": []marqo.ScoreModifier{
			// 			{
			// 					FieldName: "startTime",
			// 					Weight:    0.8,
			// 			},
			// 			{
			// 					FieldName: "createdAt",
			// 					Weight:    0.0,
			// 			},
			// 			{
			// 					FieldName: "updatedAt",
			// 					Weight:    0.0,
			// 			},
			// 	},
			//   "add_to_score": []marqo.ScoreModifier{
			//       {
			//         FieldName: "startTime",
			//         Weight:    0.9,
			//       },
			// 			{
			//         FieldName: "name_description_address",
			//         Weight:    -0.8,
			//     },
			// 	},
			// },
			// ScoreModifiersTensor: &marqo.ScoreModifiers{
			//   "add_to_score": []marqo.ScoreModifier{
			//       {
			//         FieldName: "startTime",
			//         Weight:    0.00001,
			//       },
			// 	},
			// },
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

	// Group and sort the search results
	groupedEvents := groupAndSortEvents(searchResp.Hits)

	// Interleave the grouped events
	interleavedEvents := interleaveEvents(groupedEvents)

	// Extract the events from the search response
	var events []types.Event
	for _, doc := range interleavedEvents {

		event := NormalizeMarqoDocOrSearchRes(doc)
		if event != nil {
			if parseDates == "1" {
				localizedTime, localizedDate := helpers.GetLocalDateAndTime(event.StartTime, event.Timezone)
				event.LocalizedStartTime = localizedTime
				event.LocalizedStartDate = localizedDate
			}

			events = append(events, *event)
		}

	}

	return types.EventSearchResponse{
		Query:  query,
		Filter: filter,
		Events: events,
	}, nil
}

func groupAndSortEvents(hits []map[string]interface{}) map[string][]map[string]interface{} {
	groupA := []map[string]interface{}{}
	groupB := make(map[float64][]map[string]interface{})

	for _, doc := range hits {
		score, ok := doc["_tensor_score"].(float64)
		if !ok {
			groupA = append(groupA, doc)
			continue
		}

		grouped := false
		for baseScore := range groupB {
			if math.Abs(score-baseScore) <= 0.002 {
				groupB[baseScore] = append(groupB[baseScore], doc)
				grouped = true
				break
			}
		}

		if !grouped {
			if len(groupB) == 0 || math.Abs(score-getClosestBaseScore(groupB, score)) > 0.002 {
				groupB[score] = []map[string]interface{}{doc}
			} else {
				groupA = append(groupA, doc)
			}
		}
	}

	// Sort each group in groupB by startTime
	for _, group := range groupB {
		sort.Slice(group, func(i, j int) bool {
			timeI, _ := group[i]["startTime"].(float64)
			timeJ, _ := group[j]["startTime"].(float64)
			return timeI < timeJ
		})
	}

	return map[string][]map[string]interface{}{
		"A": groupA,
		"B": flattenGroupB(groupB),
	}
}

func getClosestBaseScore(groupB map[float64][]map[string]interface{}, score float64) float64 {
	var closest float64
	minDiff := math.Inf(1)
	for baseScore := range groupB {
		diff := math.Abs(score - baseScore)
		if diff < minDiff {
			minDiff = diff
			closest = baseScore
		}
	}
	return closest
}

func flattenGroupB(groupB map[float64][]map[string]interface{}) []map[string]interface{} {
	var flattened []map[string]interface{}
	for _, group := range groupB {
		flattened = append(flattened, group...)
	}
	return flattened
}

func interleaveEvents(groupedEvents map[string][]map[string]interface{}) []map[string]interface{} {
	groupA := groupedEvents["A"]
	groupB := groupedEvents["B"]

	result := make([]map[string]interface{}, 0, len(groupA)+len(groupB))

	i, j := 0, 0
	for i < len(groupA) || j < len(groupB) {
		if i < len(groupA) {
			result = append(result, groupA[i])
			i++
		}

		if j < len(groupB) {
			insertIndex := len(result) * (j + 1) / (len(groupB) + 1)
			result = append(result, nil)
			copy(result[insertIndex+1:], result[insertIndex:])
			result[insertIndex] = groupB[j]
			j++
		}
	}

	return result
}

func BulkGetMarqoEventByID(client *marqo.Client, docIds []string, parseDates string) ([]*types.Event, error) {
	indexName := GetMarqoIndexName()
	getDocumentsReq := &marqo.GetDocumentsRequest{
		IndexName:   indexName,
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
		event := NormalizeMarqoDocOrSearchRes(result)
		if parseDates == "1" {
			localizedTime, localizedDate := helpers.GetLocalDateAndTime(event.StartTime, event.Timezone)
			event.LocalizedStartTime = localizedTime
			event.LocalizedStartDate = localizedDate
		}
		events = append(events, event)
	}
	return events, nil
}

func GetMarqoEventByID(client *marqo.Client, docId string, parseDates string) (*types.Event, error) {
	docIds := []string{docId}
	events, err := BulkGetMarqoEventByID(client, docIds, parseDates)
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

func NormalizeMarqoDocOrSearchRes(doc map[string]interface{}) (event *types.Event) {
	// NOTE: seems to be a bug in Go that instantiates these `int64` values as
	// `float64` when they are parsed / marshalled
	startTimeFloat := getValue[float64](doc, "startTime")
	startTimeInt := int64(startTimeFloat)

	event = &types.Event{
		Id:              getValue[string](doc, "_id"),
		EventOwners:     getStringSlice(doc, "eventOwners"),
		EventOwnerName:  getValue[string](doc, "eventOwnerName"),
		EventSourceId:   getValue[string](doc, "eventSourceId"),
		EventSourceType: getValue[string](doc, "eventSourceType"),
		Name:            getValue[string](doc, "name"),
		Description:     getValue[string](doc, "description"),
		StartTime:       startTimeInt,
		Address:         getValue[string](doc, "address"),
		Lat:             getValue[float64](doc, "lat"),
		Long:            getValue[float64](doc, "long"),
		Timezone:        getValue[string](doc, "timezone"),
		Categories:      getStringSlice(doc, "categories"),
		Tags:            getStringSlice(doc, "tags"),
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
		{"eventSourceType", func() {
			if v := getValue[string](doc, "eventSourceType"); v != "" {
				event.EventSourceType = v
			}
		}},
		{"eventSourceId", func() {
			if v := getValue[string](doc, "eventSourceId"); v != "" {
				event.EventSourceId = v
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
			if v := getValue[bool](doc, "hasRegistrationFields"); v {
				event.HasRegistrationFields = v
			}
		}},
		{"hasPurchasable", func() {
			if v := getValue[bool](doc, "hasPurchasable"); v {
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
		{"hideCrossPromo", func() {
			if v := getValue[bool](doc, "hideCrossPromo"); v {
				event.HideCrossPromo = v
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
	event.RefUrl = os.Getenv("APEX_URL") + "/event/" + event.Id

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
