package handlers

import (
	"context"
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

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/handlers/dynamodb_handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/webhook"
)

var validate *validator.Validate = validator.New()

type WeaviateHandler struct {
	WeaviateService services.WeaviateServiceInterface
}

func NewWeaviateHandler(weaviateService services.WeaviateServiceInterface) *WeaviateHandler {
	return &WeaviateHandler{WeaviateService: weaviateService}
}

type PurchasableWebhookHandler struct {
	PurchasableService internal_types.PurchasableServiceInterface
	PurchaseService    internal_types.PurchaseServiceInterface
}

func NewPurchasableWebhookHandler(purchasableService internal_types.PurchasableServiceInterface, purchaseService internal_types.PurchaseServiceInterface) *PurchasableWebhookHandler {
	return &PurchasableWebhookHandler{PurchasableService: purchasableService, PurchaseService: purchaseService}
}

type SubscriptionWebhookHandler struct {
	SubscriptionService interfaces.StripeSubscriptionServiceInterface
}

func NewSubscriptionWebhookHandler(subscriptionService interfaces.StripeSubscriptionServiceInterface) *SubscriptionWebhookHandler {
	return &SubscriptionWebhookHandler{SubscriptionService: subscriptionService}
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

func ValidateSingleEventPaylod(w http.ResponseWriter, r *http.Request, requireId bool) (event types.Event, status int, err error) {
	var raw services.RawEvent

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return types.Event{}, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err)
	}

	err = json.Unmarshal(body, &raw)
	if err != nil {
		return types.Event{}, http.StatusUnprocessableEntity, fmt.Errorf("invalid JSON payload: %w", err)
	}

	event, status, err = services.SingleValidateEvent(raw, requireId)
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
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
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

func HandleBatchEventValidation(w http.ResponseWriter, r *http.Request, requireIds bool) ([]types.Event, int, error) {
	var payload struct {
		Events []services.RawEvent `json:"events" validate:"required,min=1"`
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

	// Additional check with custom message
	if len(payload.Events) == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("events array must contain at least one event")
	}

	events, statusCode, err := services.BulkValidateEvents(payload.Events, requireIds)
	if err != nil {
		return nil, statusCode, fmt.Errorf("invalid body: %w", err)
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
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
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
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}
	eventId := mux.Vars(r)[constants.EVENT_ID_KEY]
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
	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	events, status, err := HandleBatchEventValidation(w, r, true)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
		return
	}
	res, err := services.BulkUpdateWeaviateEventsByID(r.Context(), weaviateClient, events)
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
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.BulkUpdateEvents(w, r)
	}
}

func SearchLocationsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle OPTIONS preflight request
		if r.Method == "OPTIONS" {
			transport.SetCORSHeaders(w, r)
			return
		}

		// Set CORS headers for the response
		transport.SetCORSHeaders(w, r)

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
						// []byte(fmt.Sprintf(err.Error())),
						[]byte(err.Error()),
						http.StatusBadRequest,
						nil)
					return
				}
				continue // Skip the numeric validation below for tm_ prefixed IDs
			}
			// Check if ID is exactly 18 characters
			if len(id) != constants.ZITADEL_USER_ID_LEN {
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
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	eventId := mux.Vars(r)[constants.EVENT_ID_KEY]
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
	q, _, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	var res types.EventSearchResponse
	res, err = services.SearchWeaviateEvents(r.Context(), weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
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

func CreateSubscriptionCheckoutSession(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
	}
	userId := userInfo.Sub
	if userId == "" {
		transport.SendServerRes(w, []byte("Missing user ID"), http.StatusUnauthorized, nil)
		return
	}

	subscriptionPlanID := r.URL.Query().Get("subscription_plan_id")
	growthPlanID := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH")
	seedPlanID := os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED")
	if subscriptionPlanID == "" || (subscriptionPlanID != growthPlanID && subscriptionPlanID != seedPlanID) {
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=checkout_failed", http.StatusSeeOther)
		return nil
	}

	// Check if user already has this subscription by checking roleClaims
	roleClaims := []constants.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
		roleClaims = claims
	}

	// Prevent checkout if user already has this subscription
	if subscriptionPlanID == growthPlanID && helpers.HasRequiredRole(roleClaims, []string{constants.Roles[constants.SubGrowth]}) {
		log.Printf("User %s already has Growth subscription, preventing duplicate checkout", userId)
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=already_subscribed", http.StatusSeeOther)
		return nil
	}
	if subscriptionPlanID == seedPlanID && (helpers.HasRequiredRole(roleClaims, []string{constants.Roles[constants.SubSeed]}) || helpers.HasRequiredRole(roleClaims, []string{constants.Roles[constants.SubGrowth]})) {
		// Prevent Seed checkout if user has Seed OR Growth subscription (Growth includes Seed features)
		log.Printf("User %s already has Seed subscription (or Growth which includes Seed), preventing duplicate checkout", userId)
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=already_subscribed", http.StatusSeeOther)
		return nil
	}

	subscriptionService := services.NewStripeSubscriptionService()
	subscriptions, err := subscriptionService.GetSubscriptionPlans()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get subscription plans: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	var subscriptionPlan *types.SubscriptionPlan
	for _, subscription := range subscriptions {
		if subscription.ID == subscriptionPlanID {
			subscriptionPlan = subscription
			break
		}
	}

	if subscriptionPlan == nil {
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=checkout_failed", http.StatusSeeOther)
		return nil
	}

	// Continue with existing Stripe checkout logic for paid items
	stripeClient := services.GetStripeClient()

	// Search for existing Stripe customer by Zitadel user ID in metadata
	stripeCustomer, searchErr := subscriptionService.SearchCustomerByExternalID(userInfo.Sub)
	if searchErr != nil {
		log.Printf("Error searching for Stripe customer: %v", searchErr)
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=checkout_failed", http.StatusSeeOther)
		return nil
	}

	// If customer doesn't exist, create it
	var createErr error
	if stripeCustomer == nil {
		log.Printf("Stripe customer not found, creating new customer for Zitadel user %s", userInfo.Sub)
		stripeCustomer, createErr = subscriptionService.CreateCustomer(userInfo.Sub, userInfo.Email, userInfo.Name)
		if createErr != nil {
			log.Printf("Error creating Stripe customer: %v", createErr)
			http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=checkout_failed", http.StatusSeeOther)
			return nil
		}
		log.Printf("Created Stripe customer %s for Zitadel user %s", stripeCustomer.ID, userInfo.Sub)
	} else {
		log.Printf("Found existing Stripe customer %s for Zitadel user %s", stripeCustomer.ID, userInfo.Sub)
	}

	lineItems := make([]*stripe.CheckoutSessionCreateLineItemParams, 0)

	lineItems = append(lineItems, &stripe.CheckoutSessionCreateLineItemParams{
		Quantity: stripe.Int64(1),
		Price:    stripe.String(subscriptionPlan.PriceID),
	})

	// Map subscription plan ID to role name
	roleMap := map[string]string{
		os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED"):   constants.Roles[constants.SubSeed],
		os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH"): constants.Roles[constants.SubGrowth],
	}
	roleName := roleMap[subscriptionPlanID]

	params := &stripe.CheckoutSessionCreateParams{
		SuccessURL: stripe.String(constants.CUSTOMER_PORTAL_RETURN_URL_PATH + "?new_role=" + roleName),
		CancelURL:  stripe.String(os.Getenv("APEX_URL") + strings.Replace(constants.SitePages["pricing"].Slug, "{trailingslash:\\/?}", "", 1)),
		Customer:   stripe.String(stripeCustomer.ID),
		LineItems:  lineItems,
		// NOTE: `mode` needs to be "subscription" if there's a subscription / recurring item,
		// use `add_invoice_item` to then append the one-time payment items:
		// https://stackoverflow.com/questions/64011643/how-to-combine-a-subscription-and-single-payments-in-one-charge-stripe-ap
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
	}
	stripeCheckoutResult, err := stripeClient.V1CheckoutSessions.Create(context.Background(), params)
	if err != nil {
		log.Printf("Error creating Stripe checkout session: %v", err)
		http.Redirect(w, r, os.Getenv("APEX_URL")+"/pricing?error=checkout_failed", http.StatusSeeOther)
		return nil
	}

	// Send the response - redirect to Stripe checkout URL
	http.Redirect(w, r, stripeCheckoutResult.URL, http.StatusSeeOther)
	return nil
}

func CreateSubscriptionCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CreateSubscriptionCheckoutSession(w, r)
	}
}

func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) (err error) {
	ctx := r.Context()
	vars := mux.Vars(r)
	eventId := vars[constants.EVENT_ID_KEY]
	eventSourceId := r.URL.Query().Get("event_source_id")
	eventSourceType := r.URL.Query().Get("event_source_type")
	if eventSourceId != "" && eventSourceType == constants.ES_EVENT_SERIES {
		eventId = eventSourceId
	}
	if eventId == "" {
		transport.SendServerRes(w, []byte("Missing event ID"), http.StatusBadRequest, nil)
		return
	}

	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
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
		createPurchase.Status = constants.PurchaseStatus.Registered // Mark as registered immediately since it's free

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
	stripeClient := services.GetStripeClient()

	// Search for existing Stripe customer by Zitadel user ID in metadata
	subscriptionService := services.NewStripeSubscriptionService()
	stripeCustomer, searchErr := subscriptionService.SearchCustomerByExternalID(userInfo.Sub)
	if searchErr != nil {
		log.Printf("Error searching for Stripe customer: %v", searchErr)
		var errMsg = []byte("ERR: Failed to search for Stripe customer: " + searchErr.Error())
		log.Println(string(errMsg))
		transport.SendServerRes(w, errMsg, http.StatusInternalServerError, searchErr)
		return nil
	}

	// If customer doesn't exist, create it
	var createErr error
	if stripeCustomer == nil {
		log.Printf("Stripe customer not found, creating new customer for Zitadel user %s", userInfo.Sub)
		stripeCustomer, createErr = subscriptionService.CreateCustomer(userInfo.Sub, userInfo.Email, userInfo.Name)
		if createErr != nil {
			log.Printf("Error creating Stripe customer: %v", createErr)
			var errMsg = []byte("ERR: Failed to create Stripe customer: " + createErr.Error())
			log.Println(string(errMsg))
			transport.SendServerRes(w, errMsg, http.StatusInternalServerError, createErr)
			return nil
		}
		log.Printf("Created Stripe customer %s for Zitadel user %s", stripeCustomer.ID, userInfo.Sub)
	} else {
		log.Printf("Found existing Stripe customer %s for Zitadel user %s", stripeCustomer.ID, userInfo.Sub)
	}

	lineItems := make([]*stripe.CheckoutSessionCreateLineItemParams, len(createPurchase.PurchasedItems))

	for i, item := range createPurchase.PurchasedItems {
		lineItems[i] = &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(int64(item.Quantity)),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency:   stripe.String("USD"),
				UnitAmount: stripe.Int64(int64(item.Cost)), // Convert to cents
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
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

	params := &stripe.CheckoutSessionCreateParams{
		ClientReferenceID: stripe.String(referenceId), // Store purchase
		SuccessURL:        stripe.String(os.Getenv("APEX_URL") + "/admin?new_purch_key=" + createPurchase.CompositeKey),
		CancelURL:         stripe.String(os.Getenv("APEX_URL") + "/event/" + eventId + "?checkout=cancel"),
		Customer:          stripe.String(stripeCustomer.ID),
		LineItems:         lineItems,
		// NOTE: `mode` needs to be "subscription" if there's a subscription / recurring item,
		// use `add_invoice_item` to then append the one-time payment items:
		// https://stackoverflow.com/questions/64011643/how-to-combine-a-subscription-and-single-payments-in-one-charge-stripe-ap
		Mode:      stripe.String(string(stripe.CheckoutSessionModePayment)),
		ExpiresAt: stripe.Int64(time.Now().Add(30 * time.Minute).Unix()),
	}

	stripeCheckoutResult, err := stripeClient.V1CheckoutSessions.Create(context.Background(), params)

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
		createPurchase.Status = constants.PurchaseStatus.Pending

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
	stripeHeader := r.Header.Get("stripe-signature")
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
		purchaseUpdate.Status = constants.PurchaseStatus.Settled
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
		purchaseUpdate.Status = constants.PurchaseStatus.Canceled

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

func (h *SubscriptionWebhookHandler) HandleSubscriptionWebhook(w http.ResponseWriter, r *http.Request) (err error) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("ERR: Error reading request body: %v\n", err)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusInternalServerError, nil)
		return err
	}

	endpointSecret := services.GetStripeCheckoutWebhookSecret()
	stripeHeader := r.Header.Get("stripe-signature")
	event, err := webhook.ConstructEvent(payload, stripeHeader,
		endpointSecret)
	if err != nil {
		msg := fmt.Sprintf("ERR: Error verifying webhook signature: %v\n", err)
		transport.SendServerRes(w, []byte(msg), http.StatusBadRequest, nil)
		return err
	}

	switch event.Type {
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_CREATED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}

		addedGrowthPlan := false
		addedSeedPlan := false
		for _, item := range subscription.Items.Data {
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH") {
				addedGrowthPlan = true
			}
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED") {
				addedSeedPlan = true
			}
		}

		// Get the Stripe customer
		s := services.GetStripeClient()
		customer, err := s.V1Customers.Retrieve(context.Background(), subscription.Customer.ID, nil)
		if err != nil {
			log.Printf("Error retrieving customer: %v", err)
			transport.SendServerRes(w, []byte("Error retrieving customer"), http.StatusInternalServerError, nil)
			return err
		}

		// Extract the Zitadel user ID from metadata
		var zitadelUserID string
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["zitadel_user_id"]; exists {
				zitadelUserID = extID
			}
		}

		log.Printf("Subscription created for Zitadel user: %s, Customer: %s, Growth: %v, Seed: %v", zitadelUserID, subscription.Customer.ID, addedGrowthPlan, addedSeedPlan)

		roles, err := helpers.GetUserRoles(zitadelUserID)
		if err != nil {
			log.Printf("Error getting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error getting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		if addedGrowthPlan && helpers.ArrFindFirst(roles, []string{constants.Roles[constants.SubGrowth]}) == "" {
			roles = append(roles, constants.Roles[constants.SubGrowth])
		}
		if addedSeedPlan && helpers.ArrFindFirst(roles, []string{constants.Roles[constants.SubSeed]}) == "" {
			roles = append(roles, constants.Roles[constants.SubSeed])
		}

		err = helpers.SetUserRoles(zitadelUserID, roles)
		if err != nil {
			log.Printf("Error setting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error setting user roles"), http.StatusInternalServerError, nil)
			return err
		}
		msg := fmt.Sprintf("Customer subscription: %s created for customer: %s", subscription.ID, subscription.Customer.ID)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_UPDATED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}

		// Determine which plans are active in the subscription
		hasGrowthPlan := false
		hasSeedPlan := false
		for _, item := range subscription.Items.Data {
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH") {
				hasGrowthPlan = true
			}
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED") {
				hasSeedPlan = true
			}
		}

		// Get the Stripe customer
		s := services.GetStripeClient()
		customer, err := s.V1Customers.Retrieve(context.Background(), subscription.Customer.ID, nil)
		if err != nil {
			log.Printf("Error retrieving customer: %v", err)
			transport.SendServerRes(w, []byte("Error retrieving customer"), http.StatusInternalServerError, nil)
			return err
		}

		// Extract the Zitadel user ID from metadata
		var zitadelUserID string
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["zitadel_user_id"]; exists {
				zitadelUserID = extID
			}
		}

		if zitadelUserID == "" {
			log.Printf("Warning: No zitadel_user_id found in customer metadata for subscription %s", subscription.ID)
			msg := fmt.Sprintf("Customer subscription updated: %s (no user ID found)", subscription.ID)
			transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
			return nil
		}

		log.Printf("Subscription updated for Zitadel user: %s, Customer: %s, Subscription: %s, Growth: %v, Seed: %v", zitadelUserID, subscription.Customer.ID, subscription.ID, hasGrowthPlan, hasSeedPlan)

		// Get current user roles
		roles, err := helpers.GetUserRoles(zitadelUserID)
		if err != nil {
			log.Printf("Error getting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error getting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		// Build desired roles list (preserve non-subscription roles)
		var updatedRoles []string
		subGrowthRole := constants.Roles[constants.SubGrowth]
		subSeedRole := constants.Roles[constants.SubSeed]

		// Keep all non-subscription roles
		for _, role := range roles {
			if role != subGrowthRole && role != subSeedRole {
				updatedRoles = append(updatedRoles, role)
			}
		}

		// Add subscription roles based on current subscription items
		if hasGrowthPlan && helpers.ArrFindFirst(updatedRoles, []string{subGrowthRole}) == "" {
			updatedRoles = append(updatedRoles, subGrowthRole)
		}
		if hasSeedPlan && helpers.ArrFindFirst(updatedRoles, []string{subSeedRole}) == "" {
			updatedRoles = append(updatedRoles, subSeedRole)
		}

		// Update roles in Zitadel
		err = helpers.SetUserRoles(zitadelUserID, updatedRoles)
		if err != nil {
			log.Printf("Error setting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error setting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		msg := fmt.Sprintf("Customer subscription updated: %s for customer: %s", subscription.ID, subscription.Customer.ID)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_DELETED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}

		// Determine which plans were in the deleted subscription (from webhook payload)
		deletedGrowthPlan := false
		deletedSeedPlan := false
		for _, item := range subscription.Items.Data {
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH") {
				deletedGrowthPlan = true
			}
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED") {
				deletedSeedPlan = true
			}
		}

		// Get the Stripe customer
		s := services.GetStripeClient()
		customer, err := s.V1Customers.Retrieve(context.Background(), subscription.Customer.ID, nil)
		if err != nil {
			log.Printf("Error retrieving customer: %v", err)
			transport.SendServerRes(w, []byte("Error retrieving customer"), http.StatusInternalServerError, nil)
			return err
		}

		// Extract the Zitadel user ID from metadata
		var zitadelUserID string
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["zitadel_user_id"]; exists {
				zitadelUserID = extID
			}
		}

		if zitadelUserID == "" {
			log.Printf("Warning: No zitadel_user_id found in customer metadata for deleted subscription %s", subscription.ID)
			msg := fmt.Sprintf("Customer subscription deleted: %s (no user ID found)", subscription.ID)
			transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
			return nil
		}

		log.Printf("Subscription deleted for Zitadel user: %s, Customer: %s, Subscription: %s, Deleted Growth: %v, Deleted Seed: %v", zitadelUserID, subscription.Customer.ID, subscription.ID, deletedGrowthPlan, deletedSeedPlan)

		// Since we prevent duplicate subscriptions at checkout, we can simply remove roles
		// Get current user roles
		roles, err := helpers.GetUserRoles(zitadelUserID)
		if err != nil {
			log.Printf("Error getting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error getting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		// Remove subscription roles that were in the deleted subscription (keep all other roles)
		var updatedRoles []string
		subGrowthRole := constants.Roles[constants.SubGrowth]
		subSeedRole := constants.Roles[constants.SubSeed]

		for _, role := range roles {
			// Remove role if it was in the deleted subscription
			shouldKeep := true
			if role == subGrowthRole && deletedGrowthPlan {
				shouldKeep = false
			}
			if role == subSeedRole && deletedSeedPlan {
				shouldKeep = false
			}

			if shouldKeep {
				updatedRoles = append(updatedRoles, role)
			}
		}

		// Update roles in Zitadel (remove subscription roles)
		err = helpers.SetUserRoles(zitadelUserID, updatedRoles)
		if err != nil {
			log.Printf("Error setting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error setting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		log.Printf("Removed subscription roles for Zitadel user: %s after subscription deletion (Growth: %v, Seed: %v)", zitadelUserID, deletedGrowthPlan, deletedSeedPlan)
		msg := fmt.Sprintf("Customer subscription deleted: %s for customer: %s", subscription.ID, subscription.Customer.ID)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_PAUSED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}

		// Determine which plans are in the paused subscription
		pausedGrowthPlan := false
		pausedSeedPlan := false
		for _, item := range subscription.Items.Data {
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH") {
				pausedGrowthPlan = true
			}
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED") {
				pausedSeedPlan = true
			}
		}

		// Get the Stripe customer
		s := services.GetStripeClient()
		customer, err := s.V1Customers.Retrieve(context.Background(), subscription.Customer.ID, nil)
		if err != nil {
			log.Printf("Error retrieving customer: %v", err)
			transport.SendServerRes(w, []byte("Error retrieving customer"), http.StatusInternalServerError, nil)
			return err
		}

		// Extract the Zitadel user ID from metadata
		var zitadelUserID string
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["zitadel_user_id"]; exists {
				zitadelUserID = extID
			}
		}

		if zitadelUserID == "" {
			log.Printf("Warning: No zitadel_user_id found in customer metadata for paused subscription %s", subscription.ID)
			msg := fmt.Sprintf("Customer subscription paused: %s (no user ID found)", subscription.ID)
			transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
			return nil
		}

		log.Printf("Subscription paused for Zitadel user: %s, Customer: %s, Subscription: %s, Growth: %v, Seed: %v", zitadelUserID, subscription.Customer.ID, subscription.ID, pausedGrowthPlan, pausedSeedPlan)

		// Since we prevent duplicate subscriptions at checkout, we can simply remove roles
		// Get current user roles
		roles, err := helpers.GetUserRoles(zitadelUserID)
		if err != nil {
			log.Printf("Error getting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error getting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		// Remove subscription roles from paused subscription (keep all other roles)
		var updatedRoles []string
		subGrowthRole := constants.Roles[constants.SubGrowth]
		subSeedRole := constants.Roles[constants.SubSeed]

		for _, role := range roles {
			// Remove role if it was in the paused subscription
			shouldKeep := true
			if role == subGrowthRole && pausedGrowthPlan {
				shouldKeep = false
			}
			if role == subSeedRole && pausedSeedPlan {
				shouldKeep = false
			}

			if shouldKeep {
				updatedRoles = append(updatedRoles, role)
			}
		}

		// Update roles in Zitadel (remove roles from paused subscription)
		err = helpers.SetUserRoles(zitadelUserID, updatedRoles)
		if err != nil {
			log.Printf("Error setting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error setting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		msg := fmt.Sprintf("Customer subscription paused: %s for customer: %s", subscription.ID, subscription.Customer.ID)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_PENDING_UPDATE_APPLIED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		msg := fmt.Sprintf("Customer subscription pending update applied: %s", subscription.ID)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_PENDING_UPDATE_EXPIRED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		msg := fmt.Sprintf("Customer subscription pending update expired: %s", subscription.ID)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_RESUMED:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}

		// Determine which plans are in the resumed subscription
		resumedGrowthPlan := false
		resumedSeedPlan := false
		for _, item := range subscription.Items.Data {
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_GROWTH") {
				resumedGrowthPlan = true
			}
			if item.Price.Product.ID == os.Getenv("STRIPE_SUBSCRIPTION_PLAN_SEED") {
				resumedSeedPlan = true
			}
		}

		// Get the Stripe customer
		s := services.GetStripeClient()
		customer, err := s.V1Customers.Retrieve(context.Background(), subscription.Customer.ID, nil)
		if err != nil {
			log.Printf("Error retrieving customer: %v", err)
			transport.SendServerRes(w, []byte("Error retrieving customer"), http.StatusInternalServerError, nil)
			return err
		}

		// Extract the Zitadel user ID from metadata
		var zitadelUserID string
		if customer.Metadata != nil {
			if extID, exists := customer.Metadata["zitadel_user_id"]; exists {
				zitadelUserID = extID
			}
		}

		if zitadelUserID == "" {
			log.Printf("Warning: No zitadel_user_id found in customer metadata for resumed subscription %s", subscription.ID)
			msg := fmt.Sprintf("Customer subscription resumed: %s (no user ID found)", subscription.ID)
			transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
			return nil
		}

		log.Printf("Subscription resumed for Zitadel user: %s, Customer: %s, Subscription: %s, Growth: %v, Seed: %v", zitadelUserID, subscription.Customer.ID, subscription.ID, resumedGrowthPlan, resumedSeedPlan)

		// Get current user roles
		roles, err := helpers.GetUserRoles(zitadelUserID)
		if err != nil {
			log.Printf("Error getting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error getting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		// Add subscription roles back (if not already present)
		subGrowthRole := constants.Roles[constants.SubGrowth]
		subSeedRole := constants.Roles[constants.SubSeed]

		if resumedGrowthPlan && helpers.ArrFindFirst(roles, []string{subGrowthRole}) == "" {
			roles = append(roles, subGrowthRole)
		}
		if resumedSeedPlan && helpers.ArrFindFirst(roles, []string{subSeedRole}) == "" {
			roles = append(roles, subSeedRole)
		}

		// Update roles in Zitadel (restore roles from resumed subscription)
		err = helpers.SetUserRoles(zitadelUserID, roles)
		if err != nil {
			log.Printf("Error setting user roles: %v", err)
			transport.SendServerRes(w, []byte("Error setting user roles"), http.StatusInternalServerError, nil)
			return err
		}

		msg := fmt.Sprintf("Customer subscription resumed: %s for customer: %s", subscription.ID, subscription.Customer.ID)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_SUBSCRIPTION_TRIAL_WILL_END:
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		msg := fmt.Sprintf("Customer subscription trial will end: %s", subscription.ID)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	case constants.STRIPE_WEBHOOK_EVENT_CUSTOMER_UPDATED:
		var customer stripe.Customer
		err := json.Unmarshal(event.Data.Raw, &customer)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			transport.SendServerRes(w, []byte("Error parsing webhook JSON"), http.StatusInternalServerError, nil)
			return err
		}
		msg := fmt.Sprintf("Customer updated: %s", customer.ID)
		log.Println(msg)
		transport.SendServerRes(w, []byte(msg), http.StatusOK, nil)
		return nil
	default:
		// log.Printf("Unhandled event type: %s\n", event.Type)
		transport.SendServerRes(w, []byte("Unhandled event type"), http.StatusOK, nil)
		return nil
	}
}

func HandleSubscriptionWebhookHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	subscriptionService := services.NewStripeSubscriptionService()

	handler := NewSubscriptionWebhookHandler(subscriptionService)
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler.HandleSubscriptionWebhook(w, r)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to handle subscription webhook: "+err.Error()), http.StatusInternalServerError, err)
			return
		}
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
	Events                   []services.RawEvent                     `json:"events" validate:"required"`
	RegistrationFieldsUpdate internal_types.RegistrationFieldsUpdate `json:"registrationFieldsUpdate"`
	PurchasableUpdate        internal_types.PurchasableUpdate        `json:"purchasableUpdate"`
	Rounds                   []internal_types.CompetitionRoundUpdate `json:"rounds"`
}

func UpdateEventRegPurchHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		eventId := vars[constants.EVENT_ID_KEY]

		userInfo := constants.UserInfo{}
		if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(constants.UserInfo)
		}
		roleClaims := []constants.RoleClaim{}
		if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
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
		if updateEventRegPurchPayload.PurchasableUpdate.CreatedAt.Unix() > 0 {
			createdAt = updateEventRegPurchPayload.PurchasableUpdate.CreatedAt.Unix()
		} else {
			createdAt = time.Now().Unix()
		}

		if eventId == "" {
			eventId = uuid.NewString()
			updateEventRegPurchPayload.Events[0].Id = eventId
			updateEventRegPurchPayload.RegistrationFieldsUpdate.EventId = eventId
			updateEventRegPurchPayload.PurchasableUpdate.EventId = eventId
			if constants.ES_SERIES_PARENT == updateEventRegPurchPayload.Events[0].EventSourceType {
				updateEventRegPurchPayload.Events[0].EventSourceId = nil
				for i, event := range updateEventRegPurchPayload.Events {
					if i == 0 {
						event.EventSourceId = nil
						updateEventRegPurchPayload.Events[i] = event
					} else {
						event.EventSourceId = &eventId
						event.EventSourceType = constants.ES_SINGLE_EVENT
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
			transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		events := make([]types.Event, len(updateEventRegPurchPayload.Events))
		if updateEventRegPurchPayload.Events[0].Id == "" {
			events[0].Id = eventId
		}
		for i, rawEvent := range updateEventRegPurchPayload.Events {
			rawEvent.Description = updateEventRegPurchPayload.Events[0].Description
			if updateEventRegPurchPayload.Events[0].EventSourceType == constants.ES_SERIES_PARENT {
				rawEvent.EventSourceId = &eventId
			}

			event, statusCode, err := services.SingleValidateEvent(rawEvent, false)
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
		childEventsToDelete, err := services.SearchWeaviateEvents(ctx, weaviateClient, "", []float64{0, 0}, 1000000, 0, farFutureTime.Unix(), []string{}, "", "", "", []string{constants.ES_EVENT_SERIES, constants.ES_EVENT_SERIES_UNPUB}, []string{eventId})
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
			transport.SendServerRes(w, []byte("Failed to upsert events to weaviate: "+err.Error()), http.StatusInternalServerError, err)
			return
		}

		var parentEventData types.Event
		if len(events) > 0 {
			parentEventData = events[0]
		} else {
			transport.SendServerRes(w, []byte("No event data was processed."), http.StatusInternalServerError, errors.New("no event data processed"))
		}

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

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	// TODO: check that the event user has permission to delete via `eventOwners` array

	_, err = services.BulkDeleteEventsFromWeaviate(r.Context(), weaviateClient, bulkDeleteEventsPayload.Events)
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to delete events from weaviate: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Events deleted successfully"), http.StatusOK, nil)
}

func (h *WeaviateHandler) PostReShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
	}

	roleClaims := []constants.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
		roleClaims = claims
	}

	validRoles := []string{string(constants.SubGrowth), string(constants.SubSeed), string(constants.SuperAdmin)}
	if !helpers.HasRequiredRole(roleClaims, validRoles) {
		err := errors.New("only Growth tier subscribers can re share events")
		transport.SendServerRes(w, []byte(err.Error()), http.StatusForbidden, err)
		return
	}

	userId := ""
	if userInfo.Sub != "" {
		userId = userInfo.Sub
	}

	eventId := r.URL.Query().Get("event_id")

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	resp, err := weaviateClient.Data().ObjectsGetter().
		WithID(eventId).
		WithClassName(constants.WeaviateEventClassName).
		Do(ctx)

	var existingShadowOwners []string
	if resp != nil && len(resp) > 0 && resp[0] != nil {
		if props, ok := resp[0].Properties.(map[string]interface{}); ok {
			if shadowOwners, exists := props["shadowOwners"]; exists {
				if shadowOwnersSlice, ok := shadowOwners.([]interface{}); ok {
					for _, owner := range shadowOwnersSlice {
						if ownerStr, ok := owner.(string); ok {
							existingShadowOwners = append(existingShadowOwners, ownerStr)
						}
					}
				}
			}
		}
	}

	// Always append the current userId
	existingShadowOwners = append(existingShadowOwners, userId)

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, owner := range existingShadowOwners {
		if !seen[owner] {
			seen[owner] = true
			unique = append(unique, owner)
		}
	}
	existingShadowOwners = unique

	err = weaviateClient.Data().Updater().
		WithMerge(). // merges properties into the object
		WithID(eventId).
		WithClassName(constants.WeaviateEventClassName).
		WithProperties(map[string]interface{}{
			"shadowOwners": existingShadowOwners, // Only the 'points' property is updated
		}).
		Do(ctx)

	if err != nil {
		transport.SendServerRes(w, []byte("Failed to re share event: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	transport.SendServerRes(w, []byte("Event re-shared successfully"), http.StatusOK, nil)
}

func PostReShareHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	weaviateService := services.NewWeaviateService()
	handler := NewWeaviateHandler(weaviateService)
	return func(w http.ResponseWriter, r *http.Request) {
		handler.PostReShare(w, r)
	}
}

func CheckRole(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userInfo := constants.UserInfo{}
		if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(constants.UserInfo)
		}

		if userInfo.Sub == "" {
			transport.SendServerRes(w, []byte("Unauthorized"), http.StatusUnauthorized, nil)
			return
		}

		expectedRole := r.URL.Query().Get("role")
		if expectedRole == "" {
			transport.SendServerRes(w, []byte("Missing role parameter"), http.StatusBadRequest, nil)
			return
		}

		// Get role claims from context
		roleClaims := []constants.RoleClaim{}
		if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
			roleClaims = claims
		}

		// Check if user has the expected role
		hasRole := false
		for _, roleClaim := range roleClaims {
			if roleClaim.Role == expectedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			jsonResponse := map[string]interface{}{
				"status":  "error",
				"message": constants.ROLE_NOT_FOUND_MESSAGE,
			}
			bytes, err := json.Marshal(jsonResponse)
			if err != nil {
				transport.SendServerRes(w, []byte("Failed to marshal JSON: "+err.Error()), http.StatusInternalServerError, err)
				return
			}
			// Role not found yet, return 404 so client can retry
			transport.SendServerRes(w, bytes, http.StatusNotFound, nil)
			return
		}

		// Role found! Return success response
		jsonResponse := map[string]interface{}{
			"status":  "success",
			"message": constants.ROLE_ACTIVE_MESSAGE,
		}
		bytes, err := json.Marshal(jsonResponse)
		if err != nil {
			transport.SendServerRes(w, []byte("Failed to marshal JSON: "+err.Error()), http.StatusInternalServerError, err)
			return
		}
		transport.SendServerRes(w, bytes, http.StatusOK, nil)
	}
}

func CheckRoleHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		CheckRole(w, r)
	}
}
