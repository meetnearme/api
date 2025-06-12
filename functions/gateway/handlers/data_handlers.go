package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/handlers/dynamodb_handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/checkout/session"
	"github.com/stripe/stripe-go/v80/webhook"
)

var validate *validator.Validate = validator.New()

type WeaviateHandler struct {
	WeaviateService services.WeaviateServiceInterface
}

func NewWeaviateHandler(marqoService services.WeaviateServiceInterface) *WeaviateHandler {
	return &WeaviateHandler{WeaviateService: marqoService}
}

type PurchasableWebhookHandler struct {
	PurchasableService internal_types.PurchasableServiceInterface
	PurchaseService    internal_types.PurchaseServiceInterface
}

func NewPurchasableWebhookHandler(purchasableService internal_types.PurchasableServiceInterface, purchaseService internal_types.PurchaseServiceInterface) *PurchasableWebhookHandler {
	return &PurchasableWebhookHandler{PurchasableService: purchasableService, PurchaseService: purchaseService}
}

// Create a new struct for raw JSON operations
type rawEventData struct {
	Id              string   `json:"id"`
	EventOwners     []string `json:"eventOwners" validate:"required,min=1"`
	EventOwnerName  string   `json:"eventOwnerName" validate:"required"`
	EventSourceType string   `json:"eventSourceType" validate:"required"`
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description" validate:"required"`
	Address         string   `json:"address" validate:"required"`
	Lat             float64  `json:"lat" validate:"required"`
	Long            float64  `json:"long" validate:"required"`
	Timezone        string   `json:"timezone" validate:"required"`
}

type rawEvent struct {
	rawEventData
	EventSourceId         *string     `json:"eventSourceId,omitempty"`
	StartTime             interface{} `json:"startTime" validate:"required"`
	EndTime               interface{} `json:"endTime,omitempty"`
	StartingPrice         *int32      `json:"startingPrice,omitempty"`
	Currency              *string     `json:"currency,omitempty"`
	PayeeId               *string     `json:"payeeId,omitempty"`
	CompetitionConfigId   *string     `json:"competitionConfigId,omitempty"`
	HasRegistrationFields *bool       `json:"hasRegistrationFields,omitempty"`
	HasPurchasable        *bool       `json:"hasPurchasable,omitempty"`
	ImageUrl              *string     `json:"imageUrl,omitempty"`
	Categories            *[]string   `json:"categories,omitempty"`
	Tags                  *[]string   `json:"tags,omitempty"`
	CreatedAt             *int64      `json:"createdAt,omitempty"`
	UpdatedAt             *int64      `json:"updatedAt,omitempty"`
	UpdatedBy             *string     `json:"updatedBy,omitempty"`
	HideCrossPromo        *bool       `json:"hideCrossPromo,omitempty"`
}

// Create a new struct that includes the createPurchase fields and the Stripe checkout URL
type PurchaseResponse struct {
	internal_types.PurchaseInsert
	StripeCheckoutURL string `json:"stripe_checkout_url"`
}

type BulkDeleteEventsPayload struct {
	Events []string `json:"events" validate:"required,min=1"`
}

type userMetaResult struct {
	id      string
	members []string
	err     error
}

type searchResult struct {
	foundUsers []internal_types.UserSearchResultDangerous
	err        error
}

func ConvertRawEventToEvent(raw rawEvent, requireId bool) (types.Event, error) {
	loc, err := time.LoadLocation(raw.Timezone)
	if err != nil {
		return types.Event{}, fmt.Errorf("invalid timezone: %w", err)
	}
	event := types.Event{
		Id:              raw.Id,
		EventOwners:     raw.EventOwners,
		EventOwnerName:  raw.EventOwnerName,
		EventSourceType: raw.EventSourceType,
		Name:            raw.Name,
		Description:     raw.Description,
		Address:         raw.Address,
		Lat:             raw.Lat,
		Long:            raw.Long,
		Timezone:        *loc,
	}

	// Safely assign pointer values
	if raw.StartingPrice != nil {
		event.StartingPrice = *raw.StartingPrice
	}
	if raw.Currency != nil {
		event.Currency = *raw.Currency
	}
	if raw.PayeeId != nil {
		event.PayeeId = *raw.PayeeId
	}
	if raw.HasRegistrationFields != nil {
		event.HasRegistrationFields = *raw.HasRegistrationFields
	}
	if raw.HasPurchasable != nil {
		event.HasPurchasable = *raw.HasPurchasable
	}
	if raw.ImageUrl != nil {
		event.ImageUrl = *raw.ImageUrl
	}
	if raw.Categories != nil {
		event.Categories = *raw.Categories
	}
	if raw.Tags != nil {
		event.Tags = *raw.Tags
	}
	if raw.CreatedAt != nil {
		event.CreatedAt = *raw.CreatedAt
	}
	if raw.UpdatedAt != nil {
		event.UpdatedAt = *raw.UpdatedAt
	}
	if raw.UpdatedBy != nil {
		event.UpdatedBy = *raw.UpdatedBy
	}
	if raw.HideCrossPromo != nil {
		event.HideCrossPromo = *raw.HideCrossPromo
	}
	if raw.EventSourceId != nil {
		event.EventSourceId = *raw.EventSourceId
	}
	if raw.CompetitionConfigId != nil {
		event.CompetitionConfigId = *raw.CompetitionConfigId
	}
	if raw.StartTime == nil {
		return types.Event{}, fmt.Errorf("startTime is required")
	}
	startTime, err := helpers.UtcToUnix64(raw.StartTime, loc)
	if err != nil || startTime == 0 {
		return types.Event{}, fmt.Errorf("invalid StartTime: %w", err)
	}
	event.StartTime = startTime

	if raw.EndTime != nil {
		endTime, err := helpers.UtcToUnix64(raw.EndTime, loc)
		if err != nil || endTime == 0 {
			return types.Event{}, fmt.Errorf("invalid EndTime: %w", err)
		}
		event.EndTime = endTime
	}
	if raw.PayeeId != nil || raw.StartingPrice != nil || raw.Currency != nil {

		if raw.PayeeId == nil || raw.StartingPrice == nil || raw.Currency == nil {
			return types.Event{}, fmt.Errorf("all of 'PayeeId', 'StartingPrice', and 'Currency' are required if any are present")
		}

		if raw.PayeeId != nil {
			event.PayeeId = *raw.PayeeId
		}
		if raw.Currency != nil {
			event.Currency = *raw.Currency
		}
		if raw.StartingPrice != nil {
			event.StartingPrice = *raw.StartingPrice
		}
	}
	return event, nil
}

func ValidateSingleEventPaylod(w http.ResponseWriter, r *http.Request, requireId bool) (event types.Event, status int, err error) {
	var raw rawEvent

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err)
	}

	err = json.Unmarshal(body, &raw)
	if err != nil {
		return types.Event{}, http.StatusUnprocessableEntity, fmt.Errorf("invalid JSON payload: %w", err)
	}

	event, status, err = HandleSingleEventValidation(raw, requireId)
	if err != nil {
		return types.Event{}, status, fmt.Errorf("invalid body: %w", err)
	}

	return event, status, nil
}

func (h *WeaviateHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
	createEvent, status, err := ValidateSingleEventPaylod(w, r, false)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
		return
	}

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	createEvents := []types.Event{createEvent}
	ctx := r.Context()
	res, err := services.BulkUpsertEventsToWeaviate(ctx, weaviateClient, createEvents)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to upsert event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	json, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, json, http.StatusCreated, nil)
}

func PostEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PostEvent(w, r)
	}
}

func HandleSingleEventValidation(rawEvent rawEvent, requireId bool) (types.Event, int, error) {
	if err := validate.Struct(rawEvent); err != nil {
		// Type assert to get validation errors
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Get just the first validation error
			firstErr := validationErrors[0]
			// Extract just the field name and error
			return types.Event{}, http.StatusBadRequest,
				fmt.Errorf("Field validation for '%s' failed on the '%s' tag",
					firstErr.Field(), firstErr.Tag())
		}
		return types.Event{}, http.StatusBadRequest, err
	}
	if requireId && rawEvent.Id == "" {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event has no id")
	}
	if len(rawEvent.EventOwners) == 0 {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event is missing eventOwners")
	}
	if requireId && rawEvent.Id == "" {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event has no id")
	}
	if len(rawEvent.EventOwners) == 0 {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event is missing eventOwners")
	}

	if rawEvent.EventOwnerName == "" {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event is missing eventOwnerName")
	}

	if rawEvent.Timezone == "" {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("event is missing timezone")
	}
	if helpers.ArrFindFirst([]string{rawEvent.EventSourceType}, helpers.ALL_EVENT_SOURCE_TYPES) == "" {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("invalid eventSourceType: %s", rawEvent.EventSourceType)
	}
	event, err := ConvertRawEventToEvent(rawEvent, requireId)
	if err != nil {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("invalid event : %s", err.Error())
	}
	return event, http.StatusOK, nil
}

func HandleBatchEventValidation(w http.ResponseWriter, r *http.Request, requireIds bool) ([]types.Event, int, error) {
	var payload struct {
		Events []rawEvent `json:"events"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err)
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, fmt.Errorf("invalid JSON payload: %w", err)
	}

	err = validate.Struct(&payload)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err)
	}

	events := make([]types.Event, len(payload.Events))
	for i, rawEvent := range payload.Events {
		event, statusCode, err := HandleSingleEventValidation(rawEvent, requireIds)
		if err != nil {
			return nil, statusCode, fmt.Errorf("invalid body: invalid event at index %d: %w", i, err)
		}
		events[i] = event
	}

	return events, http.StatusOK, nil
}

func (h *WeaviateHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
	events, status, err := HandleBatchEventValidation(w, r, false)

	if err != nil {
		transport.SendServerRes(w, []byte(err.Error()), status, err)
		return
	}

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	res, err := services.BulkUpsertEventsToWeaviate(r.Context(), weaviateClient, events)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to upsert events: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	json, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}
	transport.SendServerRes(w, json, http.StatusCreated, nil)
}

func PostBatchEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PostBatchEvents(w, r)
	}
}

func (h *WeaviateHandler) GetOneEvent(w http.ResponseWriter, r *http.Request) {
	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}
	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	parseDates := r.URL.Query().Get("parse_dates")
	var event *types.Event
	event, err = services.GetWeaviateEventByID(r.Context(), weaviateClient, eventId, parseDates)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	json, err := json.Marshal(event)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}
	transport.SendServerRes(w, json, http.StatusOK, nil)
}

func GetOneEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.GetOneEvent(w, r)
	}
}

func (h *WeaviateHandler) BulkUpdateEvents(w http.ResponseWriter, r *http.Request) {
	marqoClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	events, status, err := HandleBatchEventValidation(w, r, true)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
		return
	}
	res, err := services.BulkUpdateWeaviateEventsByID(r.Context(), marqoClient, events)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to upsert event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	json, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}
	transport.SendServerRes(w, json, http.StatusOK, nil)
}

func BulkUpdateEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	marqoService := services.NewWeaviateService()
	handler := NewWeaviateHandler(marqoService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.BulkUpdateEvents(w, r)
	}
}

func SearchLocationsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		// URL decode the query
		decodedQuery, err := url.QueryUnescape(query)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to decode query"), http.StatusBadRequest, err)
			return
		}

		// Search for matching cities
		query = strings.ToLower(decodedQuery)
		matches := helpers.SearchCitiesIndexed(query)

		// Prepare the response
		var jsonResponse []byte

		if len(matches) < 1 {
			jsonResponse = []byte("[]")
		} else {
			jsonResponse, err = json.Marshal(matches)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to create JSON response"), http.StatusInternalServerError, err)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
	}
}

func GetUsersHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the ids parameter
		idsParam := r.URL.Query().Get("ids")
		if idsParam == "" {
			transport.SendServerRes(w, []byte("Missing required 'ids' parameter"), http.StatusBadRequest, nil)
			return
		}

		throwOnMissing := r.URL.Query().Get("throw") == "1"

		// Parse the comma-separated ids
		ids := strings.Split(idsParam, ",")

		// Validate each ID
		for _, id := range ids {
			if strings.HasPrefix(id, "tm_") {
				err := helpers.ValidateTeamUUID(id)
				if err != nil {
					transport.SendServerRes(w,
						[]byte(fmt.Sprintf(err.Error())),
						http.StatusBadRequest,
						nil)
					return
				}
				continue // Skip the numeric validation below for tm_ prefixed IDs
			}
			// Check if ID is exactly 18 characters
			if len(id) != helpers.ZITADEL_USER_ID_LEN {
				transport.SendServerRes(w,
					[]byte(fmt.Sprintf("Invalid ID length: %s. Must be exactly 18 characters", id)),
					http.StatusBadRequest,
					nil)
				return
			}

			// Check if ID contains only numeric characters
			if !regexp.MustCompile(`^[0-9]+$`).MatchString(id) {
				transport.SendServerRes(w,
					[]byte(fmt.Sprintf("Invalid ID format: %s. Must contain only numbers", id)),
					http.StatusBadRequest,
					nil)
				return
			}
		}

		metaChan := make(chan userMetaResult)
		searchChan := make(chan searchResult)

		// Launch goroutine for SearchUsersByIDs
		go func() {
			matches, err := helpers.SearchUsersByIDs(ids, false)
			searchChan <- searchResult{foundUsers: matches, err: err}
		}()

		// Launch goroutines for each team ID
		activeRequests := 0
		for _, id := range ids {
			if strings.HasPrefix(id, "tm_") {
				activeRequests++
				go func(id string) {
					membersString, err := helpers.GetOtherUserMetaByID(id, "members")
					if throwOnMissing && err != nil {
						metaChan <- userMetaResult{id: id, members: nil, err: err}
						return
					}
					members := []string{}
					if membersString != "" {
						members = strings.Split(membersString, ",")
					}
					metaChan <- userMetaResult{id: id, members: members, err: nil}
				}(id)
			}
		}

		// Collect results
		allUserMeta := make(map[string][]string)
		var foundUsers []internal_types.UserSearchResultDangerous
		// Handle all responses
		for i := 0; i <= activeRequests; i++ {
			select {
			case metaRes := <-metaChan:
				if metaRes.err != nil {
					if throwOnMissing {
						transport.SendServerRes(w, []byte("Failed to get user meta: "+metaRes.err.Error()), http.StatusInternalServerError, metaRes.err)
						return
					}
					allUserMeta[metaRes.id] = []string{}
				}
				allUserMeta[metaRes.id] = metaRes.members
			case res := <-searchChan:
				if res.err != nil {
					transport.SendServerRes(w, []byte("Failed to search users: "+res.err.Error()), http.StatusInternalServerError, res.err)
					return
				}
				foundUsers = res.foundUsers
			}
		}

		// Merge the metadata with foundUsers
		for i, user := range foundUsers {
			if members, exists := allUserMeta[user.UserID]; exists {
				// Initialize the Metadata map if it's nil
				if foundUsers[i].Metadata == nil {
					foundUsers[i].Metadata = make(map[string]interface{})
				}
				// Add the members to the metadata as a comma-separated string
				foundUsers[i].Metadata["members"] = members
			}
		}

		var jsonResponse []byte
		if len(foundUsers) < 1 {
			jsonResponse = []byte("[]")
		} else {
			_jsonResponse, err := json.Marshal(foundUsers)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to create JSON response"), http.StatusInternalServerError, err)
				return
			}
			jsonResponse = _jsonResponse
		}

		w.Header().Set("Content-Type", "application/json")
		transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
	}
}

func SearchUsersHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")

		// URL decode the query
		decodedQuery, err := url.QueryUnescape(query)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to decode query"), http.StatusBadRequest, err)
			return
		}

		// Search for matching users
		query = strings.ToLower(decodedQuery)
		matches, err := helpers.SearchUserByEmailOrName(query)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to search users: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		var jsonResponse []byte
		if len(matches) < 1 {
			jsonResponse = []byte("[]")
		} else {
			jsonResponse, err = json.Marshal(matches)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to create JSON response"), http.StatusInternalServerError, err)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
	}
}

func (h *WeaviateHandler) UpdateOneEvent(w http.ResponseWriter, r *http.Request) {
	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	if eventId == "" {
		transport.SendServerRes(w, []byte("Event must have an id "), http.StatusInternalServerError, err)
		return
	}

	updateEvent, status, err := ValidateSingleEventPaylod(w, r, false)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
		return
	}

	updateEvent.Id = eventId
	updateEvents := []types.Event{updateEvent}

	res, err := services.BulkUpdateWeaviateEventsByID(r.Context(), weaviateClient, updateEvents)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to upsert event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	json, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}
	transport.SendServerRes(w, json, http.StatusOK, nil)
}

func UpdateOneEventHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.UpdateOneEvent(w, r)
	}
}

func (h *WeaviateHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	// Extract parameter values from the request query parameters
	q, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	marqoClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	var res types.EventSearchResponse
	res, err = services.SearchWeaviateEvents(r.Context(), marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to search events: "+err.Error()), http.StatusInternalServerError, err)
		return
	}
	json, err := json.Marshal(res)
	if err != nil {
		transport.SendServerRes(w, []byte("Error marshaling JSON"), http.StatusInternalServerError, err)
		return
	}
	transport.SendServerRes(w, json, http.StatusOK, nil)
}

func SearchEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.SearchEvents(w, r)
	}
}

func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	vars := mux.Vars(r)
	eventId := vars[helpers.EVENT_ID_KEY]
	eventSourceId := r.URL.Query().Get("event_source_id")
	eventSourceType := r.URL.Query().Get("event_source_type")
	if eventSourceId != "" && eventSourceType == helpers.ES_EVENT_SERIES {
		eventId = eventSourceId
	}
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	userId := userInfo.Sub
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusUnauthorized, nil)
		return
	}

	// Create an empty struct
	var createPurchase internal_types.PurchaseInsert

	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	// all purchases are pending and a client passing this status should be overridden
	if createPurchase.Status == "" {
		createPurchase.Status = "PENDING"
	}

	err = json.Unmarshal(body, &createPurchase)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	// Set the EventID and UserID after unmarshaling
	createPurchase.EventID = eventId
	createPurchase.UserID = userId

	// Set CreatedAt and UpdatedAt to current time
	now := time.Now()
	createPurchase.CreatedAt = now.Unix()
	createPurchase.UpdatedAt = now.Unix()

	_createdAt := now.Unix()
	createdAtString := fmt.Sprintf("%020d", _createdAt) // Pad with zeros to a fixed width of 20 digits

	createPurchase.CreatedAtString = createdAtString
	referenceId := "event-" + eventId + "-user-" + userId + "-time-" + createPurchase.CreatedAtString

	// Create the composite key
	compositeKey := fmt.Sprintf("%s_%s_%s", createPurchase.EventID, createPurchase.UserID, createPurchase.CreatedAtString)

	// Add the composite key and createdAt to the purchase object
	createPurchase.CompositeKey = compositeKey

	// Create the purchase record immediately instead of deferring it
	purchaseService := dynamodb_service.NewPurchaseService()
	purchaseHandler := dynamodb_handlers.NewPurchaseHandler(purchaseService)

	// when there are no purchased items, we treat this as an "RSVP" or "INTERESTED" status that shows
	// in the users purchase / registration history. The empty PurchasedItems array signals that this
	// is an event that does not have `RegistrationFields` or `PurchasableItems`
	if len(createPurchase.PurchasedItems) == 0 {
		db := transport.GetDB()
		_, err := purchaseHandler.PurchaseService.InsertPurchase(r.Context(), db, createPurchase)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to insert free purchase into database: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}

		// Create the response object
		response := PurchaseResponse{
			PurchaseInsert:    createPurchase,
			StripeCheckoutURL: "", // Empty URL for free items
		}

		// Marshal and send the response
		purchaseJSON, err := json.Marshal(response)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to marshal purchase response: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}
		transport.SendServerRes(w, purchaseJSON, http.StatusOK, nil)
		return nil
	}

	purchasableService := dynamodb_service.NewPurchasableService()
	purchasableHandler := dynamodb_handlers.NewPurchasableHandler(purchasableService)

	db := transport.GetDB()
	purchasable, err := purchasableHandler.PurchasableService.GetPurchasablesByEventID(r.Context(), db, eventId)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get purchasables for event id: "+eventId+" "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	// Validate inventory
	var purchasableMap = map[string]internal_types.PurchasableItemInsert{}
	if purchasableMap, err = validatePurchase(purchasable, createPurchase); err != nil {
		transport.SendServerRes(w, []byte("Failed to validate inventory for event id: "+eventId+": "+err.Error()), http.StatusBadRequest, err)
		return
	}

	// After validating inventory
	inventoryUpdates := make([]internal_types.PurchasableInventoryUpdate, len(createPurchase.PurchasedItems))
	for i, item := range createPurchase.PurchasedItems {
		inventoryUpdates[i] = internal_types.PurchasableInventoryUpdate{
			Name:             item.Name,
			Quantity:         purchasableMap[item.Name].Inventory - item.Quantity,
			PurchasableIndex: i,
		}
	}

	// this boolean gets toggled in the scenario where stripe checkout fails to complete or other
	// unrelated checkout failures AFTER the inventory is officially "held" + optimistically decremented
	var needsRevert bool

	err = purchasableHandler.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventId, inventoryUpdates, purchasableMap)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to update inventory: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	defer func() {
		if needsRevert {
			// Revert inventory changes if there's an error
			revertUpdates := make([]internal_types.PurchasableInventoryUpdate, len(inventoryUpdates))
			for i, update := range inventoryUpdates {
				revertUpdates[i] = internal_types.PurchasableInventoryUpdate{
					Name:             update.Name,
					Quantity:         purchasableMap[update.Name].Inventory, // Restore original inventory
					PurchasableIndex: update.PurchasableIndex,
				}
			}
			revertErr := purchasableHandler.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventId, revertUpdates, purchasableMap)
			if revertErr != nil {
				log.Printf("ERR: Failed to revert inventory changes: %v", revertErr)
			}
		}
	}()

	// Handle for free item purchases. These still need to track inventory and update the database, though we don't
	// need to create a Stripe checkout session
	if createPurchase.Total == 0 {
		// Skip Stripe checkout for free items
		createPurchase.Status = helpers.PurchaseStatus.Registered // Mark as registered immediately since it's free

		db := transport.GetDB()
		_, err := purchaseHandler.PurchaseService.InsertPurchase(r.Context(), db, createPurchase)
		if err != nil {
			needsRevert = true
			transport.SendServerRes(w, []byte("Failed to insert free purchase into database: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}

		// Create the response object
		response := PurchaseResponse{
			PurchaseInsert:    createPurchase,
			StripeCheckoutURL: "", // Empty URL for free items
		}

		// Marshal and send the response
		purchaseJSON, err := json.Marshal(response)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to marshal purchase response: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}
		transport.SendServerRes(w, purchaseJSON, http.StatusOK, nil)
		return nil
	}

	// Continue with existing Stripe checkout logic for paid items
	_, stripePrivKey := services.GetStripeKeyPair()
	stripe.Key = stripePrivKey

	lineItems := make([]*stripe.CheckoutSessionLineItemParams, len(createPurchase.PurchasedItems))

	for i, item := range createPurchase.PurchasedItems {
		lineItems[i] = &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(int64(item.Quantity)),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency:   stripe.String("USD"),
				UnitAmount: stripe.Int64(int64(item.Cost)), // Convert to cents
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.Name + " (" + createPurchase.EventName + ")"),
					Metadata: map[string]string{
						"EventId":       eventId,
						"ItemType":      item.ItemType,
						"DonationRatio": fmt.Sprint(item.DonationRatio),
					},
				},
			},
		}
	}

	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceId), // Store purchase
		SuccessURL:        stripe.String(os.Getenv("APEX_URL") + "/admin/profile?new_purch_key=" + createPurchase.CompositeKey),
		CancelURL:         stripe.String(os.Getenv("APEX_URL") + "/event/" + eventId + "?checkout=cancel"),
		LineItems:         lineItems,
		// NOTE: `mode` needs to be "subscription" if there's a subscription / recurring item,
		// use `add_invoice_item` to then append the one-time payment items:
		// https://stackoverflow.com/questions/64011643/how-to-combine-a-subscription-and-single-payments-in-one-charge-stripe-ap
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
		ExpiresAt: stripe.Int64(time.Now().Add(30 * time.Minute).Unix()),
	}

	stripeCheckoutResult, err := session.New(params)

	if err != nil {
		needsRevert = true
		var errMsg = []byte("ERR: Failed to create Stripe checkout session: " + err.Error())
		log.Println(string(errMsg))
		transport.SendServerRes(w, errMsg, http.StatusInternalServerError, err)
		return
	}

	createPurchase.StripeSessionId = stripeCheckoutResult.ID

	// Now that the checks are in place, we defer the transaction creation in the database
	// to respond to the client as quickly as possible
	defer func() {
		purchaseService := dynamodb_service.NewPurchaseService()
		h := dynamodb_handlers.NewPurchaseHandler(purchaseService)
		createPurchase.Status = helpers.PurchaseStatus.Pending

		// Create the composite key
		compositeKey := fmt.Sprintf("%s_%s_%s", createPurchase.EventID, createPurchase.UserID, createPurchase.CreatedAtString)

		// Add the composite key and createdAt to the purchase object
		createPurchase.CompositeKey = compositeKey

		db := transport.GetDB()
		_, err := h.PurchaseService.InsertPurchase(r.Context(), db, createPurchase)
		if err != nil {
			log.Printf("ERR: failed to insert purchase into purchases database for stripe session ID: %+v, err: %+v", stripeCheckoutResult.ID, err)
		}
	}()

	// Create the response object
	response := PurchaseResponse{
		PurchaseInsert:    createPurchase,
		StripeCheckoutURL: stripeCheckoutResult.URL,
	}

	// Marshal the response directly
	purchaseJSON, err := json.Marshal(response)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to marshal purchase response: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	// Send the response
	transport.SendServerRes(w, purchaseJSON, http.StatusOK, nil)
	return nil

	// ✅ 1) check inventory in the `Purchasables` table where it is tracked
	// ✅ 2) if not available, return "out of stock" error for that item
	// ✅ 3) if available, decrement the `Purchasables` table items
	// ❌ (not doing) 4) grab email from context (pull from token) and check for user in stripe customer id
	// ❌ (not doing) 5) create stripe customer Id if not present already
	// ✅ 6) Create a Stripe checkout session
	// ✅ 7) submit the transaction as PENDING with stripe `sessionId` and `customerNumber` (add to `Purchases` table)
	// ✅ 8) Handoff session to stripe
	// ✅ 9) Listen to Stripe webhook to mark transaction SETTLED
	// ❌ 10) If Stripe webhook misses, poll the stripe API for the Session ID status
	// ❌ 11) Need an SNS queue to do polling, Lambda isn't guaranteed to be there

}

func CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CreateCheckoutSession(w, r)
	}
}

// Function to transform Purchase to PurchaseUpdate
func TransformPurchaseToUpdate(purchase internal_types.Purchase) internal_types.PurchaseUpdate {
	return internal_types.PurchaseUpdate{
		UserID:              purchase.UserID,
		EventID:             purchase.EventID,
		CompositeKey:        purchase.CompositeKey,
		EventName:           purchase.EventName,
		Status:              purchase.Status,
		UpdatedAt:           time.Now().Unix(),
		StripeTransactionId: purchase.StripeTransactionId,
		StripeSessionId:     purchase.StripeSessionId,
	}
}

func (h *PurchasableWebhookHandler) HandleCheckoutWebhook(w http.ResponseWriter, r *http.Request) (err error) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("ERR: Error reading request body: %v\n", err)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, nil)
		return
	}
	// If you are testing your webhook locally with the Stripe CLI you
	// can find the endpoint's secret by running `stripe listen`
	// Otherwise, find your endpoint's secret in your webhook settings
	// in the Developer Dashboard

	endpointSecret := services.GetStripeCheckoutWebhookSecret()
	ctx := r.Context()
	apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)
	stripeHeader := apiGwV2Req.Headers["stripe-signature"]
	event, err := webhook.ConstructEvent(payload, stripeHeader,
		endpointSecret)
	if err != nil {
		msg := fmt.Sprintf("ERR: Error verifying webhook signature: %v\n", err)
		transport.SendServerRes(w, []byte(msg), http.StatusBadRequest, nil)
		return err
	}
	switch event.Type {
	case "checkout.session.completed":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		clientReferenceID := checkoutSession.ClientReferenceID

		db := transport.GetDB()
		re := regexp.MustCompile(`event-(.+?)-user-(.+?)-time-(.+)`)
		matches := re.FindStringSubmatch(clientReferenceID)
		eventID := ""
		userID := ""
		createdAt := ""
		if len(matches) > 3 {
			eventID = matches[1]
			userID = matches[2]
			createdAt = matches[3]
		}
		purchase, err := h.PurchaseService.GetPurchaseByPk(r.Context(), db, eventID, userID, createdAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get purchases for event id: "+eventID+" by clientReferenceID: "+clientReferenceID+" | error: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}
		purchaseUpdate := TransformPurchaseToUpdate(*purchase)
		purchaseUpdate.Status = helpers.PurchaseStatus.Settled
		if checkoutSession.PaymentIntent != nil {
			purchaseUpdate.StripeTransactionId = checkoutSession.PaymentIntent.ID
		}

		_, err = h.PurchaseService.UpdatePurchase(r.Context(), db, eventID, userID, createdAt, purchaseUpdate)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to update purchase status to SETTLED: "), http.StatusInternalServerError, err)
			return err
		}
		msg := fmt.Sprintf("Checkout session marked as SETTLED for stripe clientReferenceID: %s", clientReferenceID)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, err)
		return err

	case "checkout.session.expired":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		clientReferenceID := checkoutSession.ClientReferenceID
		log.Printf("Checkout session expired: client reference ID: %s", clientReferenceID)

		re := regexp.MustCompile(`event-(.+?)-user-(.+?)-time-(.+)`)
		matches := re.FindStringSubmatch(clientReferenceID)
		eventID := ""
		userID := ""
		createdAt := ""
		if len(matches) > 3 {
			eventID = matches[1]
			userID = matches[2]
			createdAt = matches[3]
		}
		db := transport.GetDB()

		purchasable, err := h.PurchasableService.GetPurchasablesByEventID(r.Context(), db, eventID)
		if err != nil {
			msg := fmt.Sprintf("ERR: Failed to get purchasables for event id: %s, err: %v", eventID, err.Error())
			log.Println(msg)
			transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, err)
			return err
		}
		// Create a map for quick lookup of purchasable items
		purchasableItems := make(map[string]internal_types.PurchasableItemInsert)
		for i, p := range purchasable.PurchasableItems {
			purchasableItems[p.Name] = internal_types.PurchasableItemInsert{
				Name:             p.Name,
				Inventory:        p.Inventory,
				StartingQuantity: p.StartingQuantity,
				PurchasableIndex: i,
			}
		}
		purchase, err := h.PurchaseService.GetPurchaseByPk(r.Context(), db, eventID, userID, createdAt)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get purchase for event id: "+eventID+" by clientReferenceID: "+clientReferenceID+" | error: "+err.Error()), http.StatusInternalServerError, err)
			return err
		}
		log.Printf("purchase: %+v", purchase)
		// Create a map of updates to restore the previously decremented inventory
		incrementUpdates := make([]internal_types.PurchasableInventoryUpdate, len(purchase.PurchasedItems))
		for i, item := range purchase.PurchasedItems {
			newQty := purchasableItems[item.Name].Inventory + item.Quantity
			if newQty > purchasableItems[item.Name].StartingQuantity {
				newQty = purchasableItems[item.Name].StartingQuantity
				msg := fmt.Sprintf("ERR: Inventory for item '%s' attempts to increment by %d above starting quantity: %d", item.Name, item.Quantity, newQty)
				log.Println(msg)
			}
			incrementUpdates[i] = internal_types.PurchasableInventoryUpdate{
				Name:             item.Name,
				Quantity:         newQty, // Increment inventory
				PurchasableIndex: purchasableItems[item.Name].PurchasableIndex,
			}
		}

		err = h.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventID, incrementUpdates, purchasableItems)
		if err != nil {
			msg := fmt.Sprintf("ERR: Failed to restore inventory changes to eventID: %s, err: %v", eventID, err)
			log.Println(msg)
			transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, err)
			return err
		}
		purchaseUpdate := TransformPurchaseToUpdate(*purchase)
		purchaseUpdate.Status = helpers.PurchaseStatus.Canceled

		_, err = h.PurchaseService.UpdatePurchase(r.Context(), db, eventID, userID, createdAt, purchaseUpdate)
		if err != nil {
			msg := fmt.Sprintf("ERR: Failed to update purchase status to CANCELED: %v", err)
			transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, err)
			return err
		}
		err = h.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventID, incrementUpdates, purchasableItems)
		if err != nil {
			msg := fmt.Sprintf("ERR: Failed to restore inventory changes to eventID: %s, err: %v", eventID, err)
			log.Println(msg)
			transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, err)
			return err
		}
		msg := fmt.Sprintf("Purchase status updated to CANCELED for compositeKey: %s", purchaseUpdate.CompositeKey)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	default:
		log.Printf("Unhandled event type: %s\n", event.Type)
	}

	transport.SendServerRes(w, []byte(event.Data.Raw), http.StatusOK, nil)
	return
}

func HandleCheckoutWebhookHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	purchasableService := dynamodb_service.NewPurchasableService()
	purchaseService := dynamodb_service.NewPurchaseService()
	handler := NewPurchasableWebhookHandler(purchasableService, purchaseService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.HandleCheckoutWebhook(w, r)
	}
}

func validatePurchase(purchasable *internal_types.Purchasable, createPurchase internal_types.PurchaseInsert) (purchasableItems map[string]internal_types.PurchasableItemInsert, err error) {
	purchases := make([]*internal_types.PurchasedItem, len(purchasable.PurchasableItems))

	// Create a map for quick lookup of purchasable items
	purchasableMap := make(map[string]internal_types.PurchasableItemInsert)
	for i, p := range purchasable.PurchasableItems {
		purchasableMap[p.Name] = internal_types.PurchasableItemInsert{
			Name:             p.Name,
			Inventory:        p.Inventory,
			Cost:             p.Cost,
			PurchasableIndex: i,
			ExpiresOn:        p.ExpiresOn,
		}
	}

	total := 0
	for i, item := range createPurchase.PurchasedItems {
		// Security check, users should not be able to modify the frontend `cost` field
		// so we validate that the cost matches the cost fetched from the database in `purchasableMap`
		if purchasableMap[item.Name].Cost != item.Cost {
			return purchasableMap, fmt.Errorf("item '%s' has incorrect cost", item.Name)
		}
		if purchasableMap[item.Name].ExpiresOn != nil && time.Now().After(*purchasableMap[item.Name].ExpiresOn) {
			return purchasableMap, fmt.Errorf("item '%s' has expired", item.Name)
		}
		total += int(item.Quantity) * int(item.Cost)
		purchases[i] = &internal_types.PurchasedItem{
			Name:     item.Name,
			Quantity: item.Quantity,
		}
	}

	// Security check, users should not be able to modify the frontend `total` field
	// so we validate that the total matches the sum of the purchased items
	if createPurchase.Total != int32(total) {
		return purchasableMap, fmt.Errorf("total cost does not match: expected %d, got %d", createPurchase.Total, total)
	}

	// Validate each purchased item
	for _, purchasedItem := range createPurchase.PurchasedItems {
		purchasableItem, exists := purchasableMap[purchasedItem.Name]
		if !exists {
			return purchasableMap, fmt.Errorf("item '%s' is not available for purchase", purchasedItem.Name)
		}

		if purchasedItem.Quantity > purchasableItem.Inventory {
			return purchasableMap, fmt.Errorf("insufficient inventory for item '%s': requested %d, available %d",
				purchasedItem.Name, purchasedItem.Quantity, purchasableItem.Inventory)
		}
	}
	return purchasableMap, nil
}

type UpdateEventRegPurchPayload struct {
	Events                   []rawEvent                              `json:"events" validate:"required"`
	RegistrationFieldsUpdate internal_types.RegistrationFieldsUpdate `json:"registrationFieldsUpdate"`
	PurchasableUpdate        internal_types.PurchasableUpdate        `json:"purchasableUpdate"`
	Rounds                   []internal_types.CompetitionRoundUpdate `json:"rounds"`
}

func UpdateEventRegPurchHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		eventId := vars[helpers.EVENT_ID_KEY]

		userInfo := helpers.UserInfo{}
		if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(helpers.UserInfo)
		}
		roleClaims := []helpers.RoleClaim{}
		if claims, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
			roleClaims = claims
		}

		validRoles := []string{"superAdmin", "eventAdmin"}
		userId := userInfo.Sub
		if userId == "" {
			transport.SendServerRes(w, []byte("Missing user ID"), http.StatusUnauthorized, nil)
			return
		}
		if !helpers.HasRequiredRole(roleClaims, validRoles) {
			err := errors.New("only event editors can add or edit events")
			transport.SendServerRes(w, []byte(err.Error()), http.StatusForbidden, err)
			return
		}

		var updateEventRegPurchPayload UpdateEventRegPurchPayload

		body, err := io.ReadAll(r.Body)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
			return
		}

		err = json.Unmarshal(body, &updateEventRegPurchPayload)
		if err != nil {
			transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
			return
		}

		// we should use goroutines to parallelize the three distinct database update operations here
		db := transport.GetDB()

		var createdAt int64
		updatedAt := time.Now().Unix()
		if updateEventRegPurchPayload.Events[0].CreatedAt != nil {
			createdAt = *updateEventRegPurchPayload.Events[0].CreatedAt
		} else {
			createdAt = time.Now().Unix()
		}

		if eventId == "" {
			eventId = uuid.NewString()
			updateEventRegPurchPayload.Events[0].Id = eventId
			updateEventRegPurchPayload.RegistrationFieldsUpdate.EventId = eventId
			updateEventRegPurchPayload.PurchasableUpdate.EventId = eventId
			if helpers.ES_SERIES_PARENT == updateEventRegPurchPayload.Events[0].EventSourceType {
				updateEventRegPurchPayload.Events[0].EventSourceId = nil
				for i, event := range updateEventRegPurchPayload.Events {
					if i == 0 {
						event.EventSourceId = nil
						updateEventRegPurchPayload.Events[i] = event
					} else {
						event.EventSourceId = &eventId
						event.EventSourceType = helpers.ES_SINGLE_EVENT
					}
				}
			}
		}

		// Call patch on rounds for eventId only
		if len(updateEventRegPurchPayload.Rounds) > 0 {
			// Define which fields to update (excluding eventId)
			keysToUpdate := []string{
				"eventId",
			}

			service := dynamodb_service.NewCompetitionRoundService()
			err = service.BatchPatchCompetitionRounds(ctx, db, updateEventRegPurchPayload.Rounds, keysToUpdate)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to update existing competition rounds: "+err.Error()), http.StatusInternalServerError, err)
				return
			}
		}

		// Update purchasable
		if updateEventRegPurchPayload.PurchasableUpdate.CreatedAt.IsZero() {
			updateEventRegPurchPayload.PurchasableUpdate.CreatedAt = time.Unix(createdAt, 0)
		}
		updateEventRegPurchPayload.PurchasableUpdate.UpdatedAt = time.Unix(updatedAt, 0)

		purchService := dynamodb_service.NewPurchasableService()
		purchHandler := dynamodb_handlers.NewPurchasableHandler(purchService)
		purchRes, err := purchHandler.PurchasableService.UpdatePurchasable(r.Context(), db, updateEventRegPurchPayload.PurchasableUpdate)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to update purchasable: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		// Update registration fields
		updateEventRegPurchPayload.RegistrationFieldsUpdate.UpdatedBy = userId
		if updateEventRegPurchPayload.RegistrationFieldsUpdate.CreatedAt.IsZero() {
			updateEventRegPurchPayload.RegistrationFieldsUpdate.CreatedAt = time.Unix(createdAt, 0)
		}
		updateEventRegPurchPayload.RegistrationFieldsUpdate.UpdatedAt = time.Unix(updatedAt, 0)
		regFieldsService := dynamodb_service.NewRegistrationFieldsService()
		regFieldsHandler := dynamodb_handlers.NewRegistrationFieldsHandler(regFieldsService)
		regFieldsRes, err := regFieldsHandler.RegistrationFieldsService.UpdateRegistrationFields(r.Context(), db, eventId, updateEventRegPurchPayload.RegistrationFieldsUpdate)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to update registration fields: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		// Update events
		weaviateClient, err := services.GetWeaviateClient()
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		events := make([]types.Event, len(updateEventRegPurchPayload.Events))
		if updateEventRegPurchPayload.Events[0].Id == "" {
			events[0].Id = eventId
		}
		for i, rawEvent := range updateEventRegPurchPayload.Events {
			if rawEvent.CreatedAt == nil {
				rawEvent.CreatedAt = &createdAt
			}
			rawEvent.UpdatedAt = &updatedAt
			rawEvent.Description = updateEventRegPurchPayload.Events[0].Description
			if updateEventRegPurchPayload.Events[0].EventSourceType == helpers.ES_SERIES_PARENT {
				rawEvent.EventSourceId = &eventId
			}

			event, statusCode, err := HandleSingleEventValidation(rawEvent, false)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to validate events: "+err.Error()), statusCode, err)
				return
			}
			events[i] = event
		}

		// Before pushing the new events, check for outdated child events, we need to search them prior to
		// the new event upsert because the new events will have unkown IDs and would get deleted by the
		// delete "sweeper" we do in the `defer` function below

		farFutureTime, _ := time.Parse(time.RFC3339, "2099-05-01T12:00:00Z")
		childEventsToDelete, err := services.SearchWeaviateEvents(ctx, weaviateClient, "", []float64{0, 0}, 1000000, 0, farFutureTime.Unix(), []string{}, "", "", "", []string{helpers.ES_EVENT_SERIES, helpers.ES_EVENT_SERIES_UNPUB}, []string{eventId})
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to search for existing child events: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		// If events being upserted have an ID, they are known and we deny-list them here so they
		// are NOT deleted by the delete "sweeper" we do in the `defer` function below
		deleteDenyList := make(map[string]bool)
		for _, event := range events {
			deleteDenyList[event.Id] = true
		}

		// Filter out events we want to keep
		var eventsToDelete []types.Event
		for _, event := range childEventsToDelete.Events {
			if !deleteDenyList[event.Id] {
				eventsToDelete = append(eventsToDelete, event)
			}
		}

		// After pushing the new events, check for outdated child events, we need to search them
		defer func() {
			if len(eventsToDelete) > 0 {
				deleteEventsArr := make([]string, len(eventsToDelete))
				for i, event := range eventsToDelete {
					deleteEventsArr[i] = event.Id
				}

				_, err = services.BulkDeleteEventsFromWeaviate(ctx, weaviateClient, deleteEventsArr)
				if err != nil {
					transport.SendServerRes(w, []byte("Failed to delete old child events: "+err.Error()), http.StatusInternalServerError, err)
					return
				}
			}
		}()

		eventsRes, err := services.BulkUpsertEventsToWeaviate(ctx, weaviateClient, events)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to upsert events to marqo: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		var parentEventData types.Event
		if len(events) > 0 {
			parentEventData = events[0]
		} else {
			transport.SendServerRes(w, []byte("No event data was processed."), http.StatusInternalServerError, errors.New("no event data processed"))
		}

		log.Printf("parentEventData: %v", parentEventData)

		// Create response object
		response := map[string]interface{}{
			"status":  "success",
			"message": "Event(s), registration fields, and purchasable(s) updated successfully",
			"data": map[string]interface{}{
				"parentEvent": parentEventData,
				"events":      eventsRes,
				"regFields":   regFieldsRes,
				"purchasable": purchRes,
			},
		}

		// Marshal the response
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			transport.SendServerRes(w, []byte(`{"error": "Failed to create response"}`), http.StatusInternalServerError, err)
			return
		}

		transport.SendServerRes(w, jsonResponse, http.StatusOK, nil)
	}
}

func BulkDeleteEventsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		BulkDeleteEvents(w, r)
	}
}

func BulkDeleteEvents(w http.ResponseWriter, r *http.Request) {
	var bulkDeleteEventsPayload BulkDeleteEventsPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	err = json.Unmarshal(body, &bulkDeleteEventsPayload)
	if err != nil {
		transport.SendServerRes(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusUnprocessableEntity, err)
		return
	}

	marqoClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	// TODO: check that the event user has permission to delete via `eventOwners` array

	_, err = services.BulkDeleteEventsFromWeaviate(r.Context(), marqoClient, bulkDeleteEventsPayload.Events)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete events from marqo: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Events deleted successfully"), http.StatusOK, nil)
}
