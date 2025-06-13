package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	// "github.com/go-openapi/strfmt"
	// "github.com/google/uuid"
	"github.com/go-openapi/strfmt"
	"github.com/golang/geo/s2"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"

	// "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object"

	// "github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate/entities/models"
)

const (
	earthRadiusKm = 6371.0
	milesPerKm    = 0.621371
)

const vectorizer = "text2vec-transformers"
const eventClassName = "EventStrict" //

func GetWeaviateClient() (*weaviate.Client, error) {
	weaviateHost := os.Getenv("WEAVIATE_HOST")
	weaviateScheme := os.Getenv("WEAVIATE_SCHEME")
	weaviatePort := os.Getenv("WEAVIATE_PORT")
	weaviateApiKey := os.Getenv("WEAVIATE_API_KEY_ALLOWED_KEYS")

	if weaviateHost == "" {
		weaviateHost = "localhost"
	}

	if weaviateScheme == "" {
		weaviateScheme = "http"
	}

	if weaviatePort == "" {
		weaviatePort = "8080"
	}

	if weaviateApiKey == "" {
		log.Printf("Please add a weaviate API Key")
	}

	weaviateHostPort := weaviateHost + ":" + weaviatePort

	cfg := weaviate.Config{
		Host:       weaviateHostPort,
		Scheme:     weaviateScheme,
		AuthConfig: auth.ApiKey{Value: weaviateApiKey},
		Headers:    nil,
		// May need AuthConfig, need to look at Marqo impl
		// what do we want our time out to be
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating Weaviate client: %w", err)
	}

	return client, nil
}

type WeaviateServiceInterface interface {
	UpsertEventToWeaviate(client *weaviate.Client, event types.Event) (*models.Object, error)
}

type WeaviateService struct{}

func NewWeaviateService() *WeaviateService {
	return &WeaviateService{}
}

func (e *WeaviateService) UpsertEventToWeaviate(client *weaviate.Client, event types.Event) (*models.Object, error) {
	// This is seems to be a placeholder for compilation checks and not ever used, modelled after the marqo service
	return nil, nil
}

func DefineWeaviateSchema(ctx context.Context, client *weaviate.Client) error {
	// Delete class if exists (same as before)
	exists, err := client.Schema().ClassExistenceChecker().WithClassName(eventClassName).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed checking class existence: %w", err)
	}
	if exists {
		log.Printf("WARN: Class '%s' exists. Deleting.", eventClassName)
		if err = client.Schema().ClassDeleter().WithClassName(eventClassName).Do(ctx); err != nil {
			return fmt.Errorf("failed deleting class '%s': %w", eventClassName, err)
		}
		log.Printf("INFO: Class '%s' deleted.", eventClassName)
	}

	// Define class structure using models.Property and string data types
	eventClass := &models.Class{
		Class:       eventClassName,
		Description: "Stores event information using the strict Go struct definition",
		Vectorizer:  vectorizer,
		ModuleConfig: map[string]interface{}{
			vectorizer: map[string]interface{}{"vectorizeClassName": false},
		},
		VectorIndexType: "hnsw",
		VectorIndexConfig: map[string]interface{}{
			"efConstruction": 128,
			"maxConnections": 32,
			"distance":       "cosine",
		},
		Properties: []*models.Property{ // Use []*models.Property directly
			// --- Define properties for ALL fields using string data types ---
			// Vectorized fields: name, description
			{Name: "name", DataType: []string{"text"}, Description: "Event name (vectorized)",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": false, "vectorizePropertyName": false}},
			},
			{Name: "description", DataType: []string{"text"}, Description: "Event description (vectorized)",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": false, "vectorizePropertyName": false}},
			},
			{Name: "address", DataType: []string{"text"}, Description: "Venue address",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": false, "vectorizePropertyName": false}},
			},
			// Other fields (NOT vectorized by default)
			{Name: "eventOwners", DataType: []string{"text[]"}, Description: "List of owner IDs", // Use "text[]" for string arrays
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "eventOwnerName", DataType: []string{"text"}, Description: "Primary owner name",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "eventSourceType", DataType: []string{"text"}, Description: "Source system type",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "startTime", DataType: []string{"int"}, Description: "Event start timestamp (Unix epoch)", // Use "int" for int64
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "endTime", DataType: []string{"int"}, Description: "Event end timestamp (Unix epoch)",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "lat", DataType: []string{"number"}, Description: "Latitude", // Use "number" for float64
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "long", DataType: []string{"number"}, Description: "Longitude",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "eventSourceId", DataType: []string{"text"}, Description: "Optional source system ID",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "startingPrice", DataType: []string{"number"}, Description: "Optional starting price", // Using "number" for the int32 -> float64 conversion in ToMap
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "currency", DataType: []string{"text"}, Description: "Optional currency code",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "payeeId", DataType: []string{"text"}, Description: "Optional payee ID",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "hasRegistrationFields", DataType: []string{"boolean"}, Description: "Flag for registration fields", // Use "boolean" for bool
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "hasPurchasable", DataType: []string{"boolean"}, Description: "Flag for purchasable items",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "imageUrl", DataType: []string{"text"}, Description: "Optional image URL",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "timezone", DataType: []string{"text"}, Description: "Timezone name (e.g., America/Denver)", // Timezone name stored as text
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "categories", DataType: []string{"text[]"}, Description: "Optional list of categories",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "tags", DataType: []string{"text[]"}, Description: "Optional list of tags",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "createdAt", DataType: []string{"int"}, Description: "Creation timestamp (Unix epoch)",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "updatedAt", DataType: []string{"int"}, Description: "Last update timestamp (Unix epoch)",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "updatedBy", DataType: []string{"text"}, Description: "User ID of last updater",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "refUrl", DataType: []string{"text"}, Description: "Optional reference URL",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "hideCrossPromo", DataType: []string{"boolean"}, Description: "Flag to hide cross-promotion",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "competitionConfigId", DataType: []string{"text"}, Description: "Optional competition config ID",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "localStartDate", DataType: []string{"text"}, Description: "UI field: Localized start date string", // UI fields as text
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
			{Name: "localStartTime", DataType: []string{"text"}, Description: "UI field: Localized start time string",
				ModuleConfig: map[string]interface{}{vectorizer: map[string]interface{}{"skip": true}},
			},
		},
	}

	// Create the class (same as before)
	err = client.Schema().ClassCreator().WithClass(eventClass).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create class '%s': %w", eventClassName, err)
	}
	return nil
}

// ToMap converts the Event struct to map[string]interface{} for Weaviate.
// It handles the time.Location by storing its name string.
func EventStructToMap(e types.Event) map[string]interface{} {
	props := map[string]interface{}{
		// Required fields mapping directly
		"eventOwners":     e.EventOwners,
		"eventOwnerName":  e.EventOwnerName,
		"eventSourceType": e.EventSourceType,
		"name":            e.Name,
		"description":     e.Description,
		"startTime":       e.StartTime,
		"address":         e.Address,
		"lat":             e.Lat,
		"long":            e.Long,
		// Handle time.Location: Store the location name string
		// Weaviate schema property must be 'text' type.
		"timezone":              e.Timezone.String(), // Use the name like "UTC" or "America/Denver"
		"hasRegistrationFields": e.HasRegistrationFields,
		"hasPurchasable":        e.HasPurchasable,
		"hideCrossPromo":        e.HideCrossPromo,
		// UI fields stored as text
		"localStartDate": e.LocalizedStartDate,
		"localStartTime": e.LocalizedStartTime,
	}

	// Add optional fields only if they have non-zero/non-empty values
	if e.EndTime != 0 {
		props["endTime"] = e.EndTime
	}
	if e.EventSourceId != "" {
		props["eventSourceId"] = e.EventSourceId
	}
	// Weaviate 'int' type maps to Go's int64, 'number' maps to float64.
	// Store int32 as number for flexibility or define schema property as int if suitable.
	// Let's use 'number' in schema and cast here.
	if e.StartingPrice != 0 {
		props["startingPrice"] = float64(e.StartingPrice)
	}
	if e.Currency != "" {
		props["currency"] = e.Currency
	}
	if e.PayeeId != "" {
		props["payeeId"] = e.PayeeId
	}
	if e.ImageUrl != "" {
		props["imageUrl"] = e.ImageUrl
	}
	if len(e.Categories) > 0 {
		props["categories"] = e.Categories
	}
	if len(e.Tags) > 0 {
		props["tags"] = e.Tags
	}
	if e.CreatedAt != 0 {
		props["createdAt"] = e.CreatedAt
	}
	if e.UpdatedAt != 0 {
		props["updatedAt"] = e.UpdatedAt
	}
	if e.UpdatedBy != "" {
		props["updatedBy"] = e.UpdatedBy
	}
	if e.RefUrl != "" {
		props["refUrl"] = e.RefUrl
	}
	if e.CompetitionConfigId != "" {
		props["competitionConfigId"] = e.CompetitionConfigId
	}

	return props
}

func BulkUpsertEventsToWeaviate(ctx context.Context, client *weaviate.Client, events []types.Event) ([]models.ObjectsGetResponse, error) {
	className := eventClassName
	batchSize := 50

	batcher := client.Batch().ObjectsBatcher()
	var objectsInCurrentBatch int = 0
	var weaviateResp []models.ObjectsGetResponse

	for i, event := range events {
		obj := &models.Object{
			Class:      className,
			Properties: EventStructToMap(event),
		}

		if event.Id != "" {
			if _, err := uuid.Parse(event.Id); err == nil {
				obj.ID = strfmt.UUID(event.Id)
			} else {
				log.Printf("WARN: Provided Event.Id '%s' is not a valid UUID. Weaviate will generate one.", event.Id)
			}
		}

		batcher.WithObjects(obj)
		objectsInCurrentBatch++

		if objectsInCurrentBatch >= batchSize || i == len(events)-1 {
			log.Printf("Flushing batch. Current batch size: %d. Total events processed so far: %d", objectsInCurrentBatch, i+1)
			weaviateResp, err := batcher.WithConsistencyLevel(replication.ConsistencyLevel.ONE).Do(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to execute batch for events around index %d: %w", i, err)
			}

			var batchErrors []string
			for _, res := range weaviateResp {
				if res.Result != nil && res.Result.Status != nil && *res.Result.Status == models.ObjectsGetResponseAO2ResultStatusFAILED {
					errMsg := fmt.Sprintf("object with ID %s failed: %v", res.ID, res.Result.Errors.Error)
					log.Printf("ERROR: Weaviate batch object error: %s", errMsg)
					batchErrors = append(batchErrors, errMsg)
				}
				if len(batchErrors) > 0 {
					return nil, fmt.Errorf("encountered %d errors in batch execution: %v", len(batchErrors), batchErrors)
				}
				log.Printf("Successfully flushed batch. %d objects processed in this batch.", objectsInCurrentBatch)
				// Reset counter for the next batch. The batcher itself is reset by the .Do() call.
				objectsInCurrentBatch = 0
			}
		}
	}

	return weaviateResp, nil
}

func SearchWeaviateEvents(
	ctx context.Context,
	client *weaviate.Client,
	query string,
	userLocation []float64,
	maxDistance float64,
	startTime, endTime int64,
	ownerIds []string,
	categories string,
	address string,
	parseDates string,
	eventSourceTypes []string,
	eventSourceIds []string,
) (types.EventSearchResponse, error) {
	className := eventClassName
	limit := 100
	var gqlQueryStringForResponse string
	var whereFilterForResponse string

	var hybridArgument *graphql.HybridArgumentBuilder
	fullTextQueryParts := []string{}
	if query != "" {
		fullTextQueryParts = append(fullTextQueryParts, query)
	}

	if categories != "" {
		fullTextQueryParts = append(fullTextQueryParts, categories)
	}

	if address != "" {
		fullTextQueryParts = append(fullTextQueryParts, address)
	}

	finalHybridQuery := strings.Join(fullTextQueryParts, " ")
	gqlQueryStringForResponse = finalHybridQuery

	if finalHybridQuery != "" {
		hybridArgument = client.GraphQL().HybridArgumentBuilder().
			WithQuery(finalHybridQuery).
			WithAlpha(0.75)
	}

	whereOperands := []*filters.WhereBuilder{}

	now := time.Now().Unix()
	loc, _ := time.LoadLocation("America/New_York")
	endOfTime, _ := helpers.UtcToUnix64("2099-12-31T11:59:59Z", loc)
	timeCondition1 := (&filters.WhereBuilder{}).
		WithOperator(filters.And).
		WithOperands([]*filters.WhereBuilder{
			(&filters.WhereBuilder{}).WithPath([]string{"startTime"}).WithOperator(filters.GreaterThanEqual).WithValueInt(startTime),
			(&filters.WhereBuilder{}).WithPath([]string{"startTime"}).WithOperator(filters.LessThanEqual).WithValueInt(endTime),
		})
	timeCondition2 := (&filters.WhereBuilder{}).
		WithOperator(filters.And).
		WithOperands([]*filters.WhereBuilder{
			(&filters.WhereBuilder{}).WithPath([]string{"startTime"}).WithOperator(filters.GreaterThanEqual).WithValueInt(now),
			(&filters.WhereBuilder{}).WithPath([]string{"startTime"}).WithOperator(filters.LessThanEqual).WithValueInt(endOfTime),
		})
	whereOperands = append(whereOperands, (&filters.WhereBuilder{}).WithOperator(filters.Or).WithOperands([]*filters.WhereBuilder{timeCondition1, timeCondition2}))

	if len(userLocation) == 2 && maxDistance > 0 {
		minLat, maxLat, minLong1, maxLong1, minLong2, maxLong2, needsSplit := calculateSearchBounds(userLocation, maxDistance)

		latFilter := (&filters.WhereBuilder{}).
			WithOperator(filters.And).
			WithOperands([]*filters.WhereBuilder{
				(&filters.WhereBuilder{}).WithPath([]string{"lat"}).WithOperator(filters.GreaterThanEqual).WithValueNumber(minLat),
				(&filters.WhereBuilder{}).WithPath([]string{"lat"}).WithOperator(filters.LessThanEqual).WithValueNumber(maxLat),
			})
		whereOperands = append(whereOperands, latFilter)

		var longFilter *filters.WhereBuilder
		if needsSplit {
			longCondition1 := (&filters.WhereBuilder{}).
				WithOperator(filters.And).
				WithOperands([]*filters.WhereBuilder{
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.GreaterThanEqual).WithValueNumber(minLong1),
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.LessThanEqual).WithValueNumber(maxLong1),
				})
			longCondition2 := (&filters.WhereBuilder{}).
				WithOperator(filters.And).
				WithOperands([]*filters.WhereBuilder{
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.GreaterThanEqual).WithValueNumber(minLong2),
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.LessThanEqual).WithValueNumber(maxLong2),
				})
			longFilter = (&filters.WhereBuilder{}).WithOperator(filters.Or).WithOperands([]*filters.WhereBuilder{longCondition1, longCondition2})
		} else {
			longFilter = (&filters.WhereBuilder{}).
				WithOperator(filters.And).
				WithOperands([]*filters.WhereBuilder{
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.GreaterThanEqual).WithValueNumber(minLong1),
					(&filters.WhereBuilder{}).WithPath([]string{"long"}).WithOperator(filters.LessThanEqual).WithValueNumber(maxLong1),
				})
		}
		whereOperands = append(whereOperands, longFilter)
	}

	if len(ownerIds) > 0 {
		whereOperands = append(whereOperands, (&filters.WhereBuilder{}).
			WithPath([]string{"eventOwners"}).
			WithOperator(filters.ContainsAny).
			WithValueText(ownerIds...))
	}

	typesToSearch := eventSourceTypes
	if len(typesToSearch) == 0 && len(helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES) > 0 {
		typesToSearch = helpers.DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES
	}
	if len(typesToSearch) > 0 {
		whereOperands = append(whereOperands, (&filters.WhereBuilder{}).
			WithPath([]string{"eventSourceType"}).
			WithOperator(filters.ContainsAny).
			WithValueText(typesToSearch...))
	}

	if len(eventSourceIds) > 0 {
		whereOperands = append(whereOperands, (&filters.WhereBuilder{}).
			WithPath([]string{"eventSourceId"}).
			WithOperator(filters.ContainsAny).
			WithValueText(eventSourceIds...))
	}

	var finalWhereFilter *filters.WhereBuilder
	if len(whereOperands) > 0 {
		finalWhereFilter = (&filters.WhereBuilder{}).WithOperator(filters.And).WithOperands(whereOperands)
		filterBytes, _ := json.Marshal(finalWhereFilter)
		whereFilterForResponse = string(filterBytes)
	}

	fields := []graphql.Field{
		{Name: "name"}, {Name: "description"}, {Name: "eventOwners"}, {Name: "eventOwnerName"},
		{Name: "eventSourceType"}, {Name: "startTime"}, {Name: "endTime"}, {Name: "address"},
		{Name: "lat"}, {Name: "long"}, {Name: "eventSourceId"}, //... and all your other fields ...
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "score"}, // Populated if hybrid search is used
		}},
	}

	// --- Step 4: Construct and Execute Query ---
	queryBuilder := client.GraphQL().Get().
		WithClassName(className).
		WithFields(fields...).
		WithLimit(limit)

	if hybridArgument != nil {
		queryBuilder.WithHybrid(hybridArgument)
	}
	if finalWhereFilter != nil {
		queryBuilder.WithWhere(finalWhereFilter)
	}

	searchResult, err := queryBuilder.Do(ctx)
	if err != nil {
		log.Printf("Error searching documents: %v", err)
		return types.EventSearchResponse{
			Query:  query,
			Filter: whereFilterForResponse,
			Events: []types.Event{},
		}, err
	}

	var events []types.Event

	if searchResult.Data == nil {
	}

	getMap, ok := searchResult.Data["Get"].(map[string]interface{})
	if !ok {
	}

	classData, ok := getMap[className].([]interface{})
	if !ok {
	}

	rawHits := make([]map[string]interface{}, 0, len(classData))
	for _, uncastedObj := range classData {
		if objMap, ok := uncastedObj.(map[string]interface{}); ok {
			rawHits = append(rawHits, objMap)
		}
	}
	groupedEvents := groupAndSortEvents(rawHits)

	// Interleave the grouped events
	interleavedEvents := interleaveEvents(groupedEvents)

	// (Your full normalization and date parsing logic goes here)
	for _, doc := range interleavedEvents {

		event, err := NormalizeWeaviateResultToEvent(doc)
		if err != nil {
			log.Printf("Warning: Could not normalize Weaviate result: %v", err)
			continue
		}
		if parseDates == "1" && event.Timezone.String() != "" {
			localizedTime, localizedDate := helpers.GetLocalDateAndTime(event.StartTime, event.Timezone)
			event.LocalizedStartTime = localizedTime
			event.LocalizedStartDate = localizedDate
		}

		events = append(events, *event)
	}

	return types.EventSearchResponse{
		Query:  gqlQueryStringForResponse,
		Filter: whereFilterForResponse,
		Events: events,
	}, nil
}

func BulkDeleteEventsFromWeaviate(ctx context.Context, client *weaviate.Client, eventIds []string) (*models.BatchDeleteResponse, error) {
	if len(eventIds) == 0 {
		log.Println("BulkDeleteEventsFromWeaviate called with no event IDs. Returning Success.")
		return &models.BatchDeleteResponse{
			Results: &models.BatchDeleteResponseResults{
				Matches:    0,
				Successful: 0,
				Failed:     0,
			},
		}, nil
	}

	className := eventClassName

	whereFilter := (&filters.WhereBuilder{}).
		WithPath([]string{"id"}).
		WithOperator(filters.ContainsAny).
		WithValueText(eventIds...)

	log.Printf("Attempting to bulk delete %d events from Weaviate class '%s'", len(eventIds), className)

	resp, err := client.Batch().ObjectsBatchDeleter().
		WithClassName(className).
		WithWhere(whereFilter).
		Do(ctx)

	if err != nil {
		log.Printf("ERROR: Failed to execute batch delete from Weaviate: %v", err)
		return nil, fmt.Errorf("failed to execute Weaviate batch delete: %w", err)
	}

	if resp != nil && resp.Results != nil {
		log.Printf("Weaviate bulk delete completed. Matched: %d, Succeeded: %d, Failed: %d",
			resp.Results.Matches, resp.Results.Successful, resp.Results.Failed)
	}

	return resp, nil
}

func GetWeaviateEventByID(ctx context.Context, client *weaviate.Client, docId string, parseDates string) (*types.Event, error) {
	if docId == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	docIds := []string{docId}

	events, err := BulkGetWeaviateEventByID(ctx, client, docIds, parseDates)
	if err != nil {
		log.Printf("Error getting event by id: %v", err)
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no event found with id: %s", docId)
	}
	return events[0], nil
}

func BulkGetWeaviateEventByID(ctx context.Context, client *weaviate.Client, docIds []string, parseDates string) ([]*types.Event, error) {
	if len(docIds) == 0 {
		return []*types.Event{}, nil
	}

	whereFilter := (&filters.WhereBuilder{}).
		WithPath([]string{"id"}).
		WithOperator(filters.ContainsAny).
		WithValueText(docIds...)

	fields := []graphql.Field{
		{Name: "name"}, {Name: "description"}, {Name: "eventOwners"}, {Name: "eventOwnerName"},
		{Name: "eventSourceType"}, {Name: "startTime"}, {Name: "endTime"}, {Name: "address"},
		{Name: "lat"}, {Name: "long"}, {Name: "eventSourceId"}, {Name: "startingPrice"},
		{Name: "currency"}, {Name: "payeeId"}, {Name: "hasRegistrationFields"}, {Name: "hasPurchasable"},
		{Name: "imageUrl"}, {Name: "timezone"}, {Name: "categories"}, {Name: "tags"},
		{Name: "createdAt"}, {Name: "updatedAt"}, {Name: "updatedBy"}, {Name: "refUrl"},
		{Name: "hideCrossPromo"}, {Name: "competitionConfigId"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"}, // We always need the ID
		}},
	}

	result, err := client.GraphQL().Get().
		WithClassName(eventClassName).
		WithWhere(whereFilter).
		WithFields(fields...).
		WithLimit(len(docIds)).
		Do(ctx)

	if err != nil {
		log.Printf("Failed to get documents by ID from Weaviate: %v", err)
		return nil, err
	}

	var events []*types.Event
	getMap, ok := result.Data["Get"].(map[string]interface{})
	if !ok || getMap == nil {
		log.Println("Weaviate 'Get by ID' query returned invalid data structure.")
		return []*types.Event{}, nil
	}
	classData, ok := getMap[eventClassName].([]interface{})
	if !ok {
		log.Printf("Weaviate 'Get by ID' query for class '%s' returned no results.", eventClassName)
		return []*types.Event{}, nil // Return empty, not an error
	}

	for _, uncastedObj := range classData {
		objMap, ok := uncastedObj.(map[string]interface{})
		if !ok {
			continue
		}

		log.Printf("ObjMap: %+v", objMap)
		event, normalizeErr := NormalizeWeaviateResultToEvent(objMap)
		if normalizeErr != nil {
			log.Printf("Warning: Could not normalize Weaviate result: %v", normalizeErr)
			continue
		}

		if parseDates == "1" {
			localizedTime, localizedDate := helpers.GetLocalDateAndTime(event.StartTime, event.Timezone)
			event.LocalizedStartTime = localizedTime
			event.LocalizedStartDate = localizedDate
		}
		events = append(events, event)
	}

	return events, nil
}

func BulkUpdateWeaviateEventsByID(ctx context.Context, client *weaviate.Client, events []types.Event) ([]models.ObjectsGetResponse, error) {
	for i, event := range events {
		if event.Id == "" {
			return nil, fmt.Errorf("event at index %d is missing an ID, cannot perform bulk update", i)
		}
	}

	log.Printf("All %d events have IDs. Proceeding with Weaviate bulk update.", len(events))
	return BulkUpsertEventsToWeaviate(ctx, client, events)
}

// HELPERS

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

func groupAndSortEvents(hits []map[string]interface{}) map[string][]map[string]interface{} {
	groupA := []map[string]interface{}{}
	groupB := make(map[float64][]map[string]interface{})

	for _, doc := range hits {
		var score float64
		var scoreFound bool

		if additional, ok := doc["_additional"].(map[string]interface{}); ok {
			// The score can come back as a string or float64 from GraphQL/JSON unmarshaling
			if scoreVal, ok := additional["score"]; ok && scoreVal != nil {
				if scoreStr, ok := scoreVal.(string); ok {
					// If it's a string, parse it
					parsedScore, err := strconv.ParseFloat(scoreStr, 64)
					if err == nil {
						score = parsedScore
						scoreFound = true
					}
				} else if scoreFloat, ok := scoreVal.(float64); ok {
					// If it's already a float64, use it
					score = scoreFloat
					scoreFound = true
				}
			}
		}

		if !scoreFound {
			// If no score is found, it goes to Group A, same as before
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

func flattenGroupB(groupB map[float64][]map[string]interface{}) []map[string]interface{} {
	var flattened []map[string]interface{}
	for _, group := range groupB {
		flattened = append(flattened, group...)
	}
	return flattened
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

func NormalizeWeaviateResultToEvent(objMap map[string]interface{}) (*types.Event, error) {
	var event types.Event

	var eventID string
	if additional, ok := objMap["_additional"].(map[string]interface{}); ok {
		if idStr, idOk := additional["id"].(string); idOk {
			eventID = idStr
		}
	}

	var timezoneStr string
	if tz, ok := objMap["timezone"].(string); ok {
		timezoneStr = tz
	}

	// We remove _additional so it doesn't interfere with mapping the actual properties.
	delete(objMap, "_additional")
	delete(objMap, "timezone")

	jsonData, err := json.Marshal(objMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Weaviate properties to JSON: %w", err)
	}

	err = json.Unmarshal(jsonData, &event)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON to types.Event struct: %w", err)
	}

	// Assign the ID we extracted earlier.
	event.Id = eventID

	if timezoneStr != "" {
		loc, loadErr := time.LoadLocation(timezoneStr)
		if loadErr == nil {
			event.Timezone = *loc
		} else {
			log.Printf("NormalizeWeaviateResultToEvent: Could not load location for timezone string '%s': %v. Defaulting to UTC.", timezoneStr, loadErr)
			event.Timezone = *time.UTC
		}
	} else {
		// Ensure a default if the timezone was missing from the result
		log.Printf("NormalizeWeaviateResultToEvent: Timezone string not found or empty. Defaulting to UTC.")
		event.Timezone = *time.UTC
	}

	event.RefUrl = os.Getenv("APEX_URL") + "/event/" + event.Id

	if event.ImageUrl == "" && event.Id != "" {
		event.ImageUrl = helpers.GetImgUrlFromHash(event) // Assuming GetImgUrlFromHash takes types.Event
	}

	return &event, nil
}

// calculateSearchBounds calculates the latitude and longitude bounds for a given location and distance
// Returns minLat, maxLat, minLong1, maxLong1, minLong2, maxLong2, needsSplit
// When needsSplit is true, minLong1/maxLong1 represents the first range and minLong2/maxLong2 represents the second range
func calculateSearchBounds(location []float64, maxDistance float64) (minLat float64, maxLat float64, minLong1 float64, maxLong1 float64, minLong2 float64, maxLong2 float64, needsSplit bool) {
	latOffset := miToLat(maxDistance) * 2
	longOffset := miToLong(maxDistance, location[0]) * 2
	s2Location := s2.LatLngFromDegrees(location[0], location[1])
	s2rect := s2.RectFromCenterSize(s2Location, s2.LatLngFromDegrees(latOffset, longOffset))

	minLat = s2rect.Lo().Lat.Degrees()
	maxLat = s2rect.Hi().Lat.Degrees()

	// If the longitude range wraps around the prime meridian, split into two bounding boxes
	if s2rect.Lo().Lng.Degrees() > s2rect.Hi().Lng.Degrees() {
		needsSplit = true
		minLong1 = float64(s2rect.Lo().Lng.Degrees())
		maxLong1 = 180
		minLong2 = -180
		maxLong2 = float64(s2rect.Hi().Lng.Degrees())
	} else {
		needsSplit = false
		minLong1 = float64(s2rect.Lo().Lng.Degrees())
		maxLong1 = float64(s2rect.Hi().Lng.Degrees())
	}

	return minLat, maxLat, minLong1, maxLong1, minLong2, maxLong2, needsSplit
}

// miToLat converts miles to latitude offset (degrees)
func miToLat(mi float64) float64 {
	// One degree of latitude is approximately 69 miles
	ret := mi / 69.0
	return ret
}

// miToLong converts miles to longitude offset (degrees)
// This varies with latitude - longitude degrees are closer together as you move away from the equator
func miToLong(mi float64, lat float64) float64 {
	// One degree of longitude at given latitude is approximately 69 * cos(latitude) miles
	ret := mi / (69.0 * math.Cos(lat*math.Pi/180))
	return ret
}
