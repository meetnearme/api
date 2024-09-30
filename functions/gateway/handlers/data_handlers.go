package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/handlers/dynamodb_handlers"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
)

var validate *validator.Validate = validator.New()

type MarqoHandler struct {
    MarqoService services.MarqoServiceInterface
}

func NewMarqoHandler(marqoService services.MarqoServiceInterface) *MarqoHandler {
    return &MarqoHandler{MarqoService: marqoService}
}

// Create a new struct for raw JSON operations
type rawEventData struct {
    Id              string   `json:"id"`
    EventOwners     []string `json:"eventOwners" validate:"required,min=1"`
		EventOwnerName  string   `json:"eventOwnerName" validate:"required"`
    Name            string   `json:"name" validate:"required"`
    Description     string   `json:"description"`
    Address         string   `json:"address"`
    Lat             float64   `json:"lat"`
    Long            float64   `json:"long"`
		Timezone        string    `json:"timezone"`
}

type rawEvent struct {
    rawEventData
    StartTime interface{} `json:"startTime" validate:"required"`
    EndTime   interface{} `json:"endTime,omitempty"`
    StartingPrice   *int32    `json:"startingPrice,omitempty"`
    Currency        *string     `json:"currency,omitempty"`
    PayeeId         *string     `json:"payeeId,omitempty"`
		HasRegistrationFields *bool `json:"hasRegistrationFields,omitempty"`
		HasPurchasable *bool  `json:"hasPurchasable,omitempty"`
		ImageUrl      *string `json:"imageUrl,omitempty"`
		Categories    *[]string `json:"categories,omitempty"`
		Tags      		*[]string `json:"tags,omitempty"`
		CreatedAt     *int64 `json:"createdAt,omitempty"`
		UpdatedAt     *int64 `json:"updatedAt,omitempty"`
		UpdatedBy     *string `json:"updatedBy,omitempty"`
}

func ConvertRawEventToEvent(raw rawEvent, requireId bool) (types.Event, error) {
    event := types.Event{
        Id:          raw.Id,
        EventOwners: raw.EventOwners,
				EventOwnerName: raw.EventOwnerName,
        Name:        raw.Name,
        Description: raw.Description,
        Address:     raw.Address,
        Lat:         raw.Lat,
        Long:        raw.Long,
				Timezone:    raw.Timezone,
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


    if raw.StartTime == nil {
        return types.Event{}, fmt.Errorf("startTime is required")
    }
    startTime, err := helpers.UtcOrUnixToUnix64(raw.StartTime)
    if err != nil || startTime == 0 {
        return types.Event{}, fmt.Errorf("invalid StartTime: %w", err)
    }
    event.StartTime = startTime

    if raw.EndTime != nil {
        endTime, err := helpers.UtcOrUnixToUnix64(raw.EndTime)
        if err != nil || endTime == 0 {
            return types.Event{}, fmt.Errorf("invalid EndTime: %w", err)
        }
    }
    if raw.PayeeId != nil || raw.StartingPrice != nil || raw.Currency != nil {

        if (raw.PayeeId == nil || raw.StartingPrice == nil || raw.Currency == nil) {
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

    event, err = ConvertRawEventToEvent(raw, requireId)
    if err != nil {
        return types.Event{}, http.StatusBadRequest, fmt.Errorf("failed to convert raw event: %w", err)
    }

    err = validate.Struct(&event)
    if err != nil {
        return types.Event{}, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err)
    }

    return event, status, nil
}


func (h *MarqoHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
    createEvent, status, err := ValidateSingleEventPaylod(w, r, false)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    createEvents := []types.Event{createEvent}

    res, err := services.BulkUpsertEventToMarqo(marqoClient, createEvents, false)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostEvent(w, r)
    }
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
        if requireIds && rawEvent.Id == "" {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d has no id", i)
        }
        if len(rawEvent.EventOwners) == 0 {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d is missing eventOwners", i)
        }

				if rawEvent.EventOwnerName == "" {
					return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d is missing eventOwnerName", i)
			}

				if rawEvent.Timezone == "" {
					return nil, http.StatusBadRequest, fmt.Errorf("invalid body: Event at index %d is missing timezone", i)
				}
        event, err := ConvertRawEventToEvent(rawEvent, requireIds)
        if err != nil {
            return nil, http.StatusBadRequest, fmt.Errorf("invalid event at index %d: %s", i, err.Error())
        }
        events[i] = event
    }

    return events, http.StatusOK, nil
}

func (h *MarqoHandler) PostBatchEvents(w http.ResponseWriter, r *http.Request) {
    events, status, err := HandleBatchEventValidation(w, r, false)

    if err != nil {
        transport.SendServerRes(w, []byte(err.Error()), status, err)
        return
    }

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    res, err := services.BulkUpsertEventToMarqo(marqoClient, events, false)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.PostBatchEvents(w, r)
    }
}

func (h *MarqoHandler) GetOneEvent(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }
    eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
    var event *types.Event
    event, err = services.GetMarqoEventByID(marqoClient, eventId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.GetOneEvent(w, r)
    }
}

func (h *MarqoHandler) BulkUpdateEvents(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    events, status, err := HandleBatchEventValidation(w, r, true)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to extract event from payload: "+err.Error()), status, err)
        return
    }


    res, err := services.BulkUpdateMarqoEventByID(marqoClient, events)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
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

func (h *MarqoHandler) UpdateOneEvent(w http.ResponseWriter, r *http.Request) {
    marqoClient, err := services.GetMarqoClient()
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

    res, err := services.BulkUpdateMarqoEventByID(marqoClient, updateEvents)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo event: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.UpdateOneEvent(w, r)
    }
}

func (h *MarqoHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
    // Extract parameter values from the request query parameters
    q, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address := GetSearchParamsFromReq(r)

    marqoClient, err := services.GetMarqoClient()
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    var res types.EventSearchResponse
    res, err = services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to search marqo events: "+err.Error()), http.StatusInternalServerError, err)
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
    marqoService := services.NewMarqoService()
    handler := NewMarqoHandler(marqoService)
    return func(w http.ResponseWriter, r *http.Request) {
        handler.SearchEvents(w, r)
    }
}

func CreateCheckoutSession(w http.ResponseWriter, r *http.Request) (err error) {
    ctx := r.Context()
		vars := mux.Vars(r)
		eventId := vars["event_id"]
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
        transport.SendServerRes(w, []byte("Missing user ID"), http.StatusBadRequest, nil)
        return
    }

    // Create an empty struct
    var createPurchase internal_types.PurchaseInsert

    body, err := io.ReadAll(r.Body)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusBadRequest, err)
        return
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
    currentTime := time.Now()
    createPurchase.CreatedAt = currentTime
    createPurchase.UpdatedAt = currentTime

		purchasableService := dynamodb_service.NewPurchasableService()
		h := dynamodb_handlers.NewPurchasableHandler(purchasableService)

    db := transport.GetDB()
    purchasable, err := h.PurchasableService.GetPurchasablesByEventID(r.Context(), db, eventId)
    if err != nil {
        transport.SendServerRes(w, []byte("Failed to get purchasables for event id: "+eventId+ " "+err.Error()), http.StatusInternalServerError, err)
        return
    }

    // Validate inventory
    var purchasableMap = map[string]internal_types.PurchasableItemInsert{}
    if purchasableMap, err = validateInventory(purchasable, createPurchase); err != nil {
        transport.SendServerRes(w, []byte("Failed to validate inventory for event id: "+eventId+ " + "+err.Error()), http.StatusBadRequest, err)
        return
    }

    // After validating inventory
    inventoryUpdates := make([]internal_types.PurchasableInventoryUpdate, len(createPurchase.PurchasedItems))
    for i, item := range createPurchase.PurchasedItems {
        inventoryUpdates[i] = internal_types.PurchasableInventoryUpdate{
            Name:     item.Name,
            Quantity: purchasableMap[item.Name].Inventory - item.Quantity,
            PurchasableIndex: i,
        }
    }

		// TODO: delete this log
		log.Printf("\n\ncreatePurchase: %+v", createPurchase)

    // this boolean gets toggled in the scenario where stripe
    // checkout instantiation or other unrelated checkout steps
    // AFTER the inventory is officially "held" + optimistically
    // decremented
    var needsRevert bool

    err = h.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventId, inventoryUpdates, purchasableMap)
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
            revertErr := h.PurchasableService.UpdatePurchasableInventory(r.Context(), db, eventId, revertUpdates, purchasableMap)
            if revertErr != nil {
                log.Printf("ERR: Failed to revert inventory changes: %v", revertErr)
            }
        }
    }()

		_, stripePrivKey := services.GetStripeKeyPair()
		stripe.Key = stripePrivKey

		lineItems := make([]*stripe.CheckoutSessionLineItemParams, len(createPurchase.PurchasedItems))

		for i, item := range createPurchase.PurchasedItems {
				lineItems[i] = &stripe.CheckoutSessionLineItemParams{
						Quantity: stripe.Int64(int64(item.Quantity)),
						PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
								Currency: stripe.String("USD"),
								UnitAmount: stripe.Int64(int64(item.Cost)), // Convert to cents
								ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
										Name: stripe.String(item.Name + " (" + createPurchase.EventName + ")"),
										Metadata: map[string]string{
												"EventId": eventId,
												"ItemType": item.ItemType,
												"DonationRatio": fmt.Sprint(item.DonationRatio),
										},
								},
						},
				}
		}

		unixNow := time.Now().Unix()
		referenceId := "event-"+eventId+"-user-"+userId+"-time-"+time.Unix(unixNow, 0).UTC().Format(time.RFC3339)
		params := &stripe.CheckoutSessionParams{
			ClientReferenceID: stripe.String(referenceId), // Store purchase
			SuccessURL: stripe.String( os.Getenv("APEX_URL") + "/events/" + eventId + "?checkout=success"),
			CancelURL:  stripe.String( os.Getenv("APEX_URL") + "/events/" + eventId + "?checkout=cancel"),
			LineItems:  lineItems,
			// TODO: `mode` needs to be "subscription" if there's a subscription / recurring item,
			// use `add_invoice_item` to then append the one-time payment items:
			// https://stackoverflow.com/questions/64011643/how-to-combine-a-subscription-and-single-payments-in-one-charge-stripe-ap
			Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		}

		stripeCheckoutResult, err := session.New(params);

		if err != nil {
			needsRevert = true
			var errMsg = []byte("ERR: Failed to create Stripe checkout session: "+err.Error())
			log.Println(string(errMsg))
			transport.SendServerRes(w, errMsg, http.StatusInternalServerError, err)
			return
		}

		createPurchase.StripeSessionId = stripeCheckoutResult.ID

		// Now that the checks are in place, we defer to the
		defer func() {
			purchaseService := dynamodb_service.NewPurchaseService()
			h := dynamodb_handlers.NewPurchaseHandler(purchaseService)

			db := transport.GetDB()
			_, err := h.PurchaseService.InsertPurchase(r.Context(), db, createPurchase)
			if err != nil {
				log.Printf("ERR: failed to insert purchase into purchases database for stripe session ID %+v", stripeCheckoutResult.ID)
			}
		}()


		log.Printf("\nstripe result: %+v", stripeCheckoutResult)

    // Create a new struct that includes the createPurchase fields and the Stripe checkout URL
    type PurchaseResponse struct {
			internal_types.PurchaseInsert
			StripeCheckoutURL string `json:"stripe_checkout_url"`
		}

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


    // TODO: we need to

    // 1) check inventory in the `Purchasables` table where it is tracked
    // 2) if not available, return "out of stock" error for that item
    // 3) if available, decrement the `Purchasables` table items
    // 4) grab email from context (pull from token) and check for user in stripe customer id
    // 5) create stripe customer Id if not present already
    // 6) Create a Stripe checkout session
    // 7) submit the transaction as PENDING with stripe `sessionId` and `customerNumber` (add to `Purchases` table)
    // 8) Handoff session to stripe
    // 9) Listen to Stripe webhook to mark transaction SETTLED
    // 10) If Stripe webhook misses, poll the stripe API for the Session ID status
    // 11) Need an SNS queue to do polling, Lambda isn't guaranteed to be there




}

func CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        CreateCheckoutSession(w, r)
    }
}

func validateInventory(purchasable *internal_types.Purchasable, createPurchase internal_types.PurchaseInsert) (purchasableItems map[string]internal_types.PurchasableItemInsert, err error) {
    purchases := make([]*internal_types.PurchasedItem, len(purchasable.PurchasableItems))
    for i, item := range createPurchase.PurchasedItems {
        purchases[i] = &internal_types.PurchasedItem{
            // Assuming PurchasableItemInsert has similar fields to Purchasable
            Name:  item.Name,
            Quantity: item.Quantity,
            // ... other fields ...
        }
    }
    // Create a map for quick lookup of purchasable items
    purchasableMap := make(map[string]internal_types.PurchasableItemInsert)
    for i, p := range purchasable.PurchasableItems {
        purchasableMap[p.Name] = internal_types.PurchasableItemInsert{
            Inventory: p.Inventory,
            PurchasableIndex: i,
        }

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
