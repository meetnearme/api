package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/a-h/templ"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"

	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type GeoLookupInputPayload struct {
	Location string `json:"location" validate:"required"`
}

type GeoThenSeshuPatchInputPayload struct {
	Location string `json:"location" validate:"required"`
	Url      string `json:"source_url" validate:"required"` // URL is the DB key in SeshuSession
}

type SeshuSessionEventsPayload struct {
	Url                     string                          `json:"event_source_url" validate:"required"` // URL is the DB key in SeshuSession
	EventBoolValid          []internal_types.EventBoolValid `json:"eventValidations" validate:"required"`
	EventRecursiveBoolValid []internal_types.EventBoolValid `json:"eventValidationRecursive" validate:"omitempty"`
}

type EventDomPaths struct {
	EventTitle       string `json:"event_title_dom"`
	EventLocation    string `json:"event_location_dom"`
	EventStartTime   string `json:"event_start_time_dom"`
	EventEndTime     string `json:"event_end_time_dom"`
	EventURL         string `json:"event_url"`
	EventDescription string `json:"event_description_dom"`
}

type SetMnmOptionsRequestPayload struct {
	Subdomain    string `json:"subdomain" validate:"required"`
	PrimaryColor string `json:"primaryColor,omitempty"`
	ThemeMode    string `json:"themeMode,omitempty"`
}

type UpdateUserAboutRequestPayload struct {
	About string `json:"about" validate:"required"`
}

type UpdateUserLocationRequestPayload struct {
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
	City      string  `json:"city" validate:"required"`
}

type eventSearchResult struct {
	events []internal_types.Event
	err    error
}

type eventParentResult struct {
	event *internal_types.Event
	err   error
}

func SetMnmOptions(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload SetMnmOptionsRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError)
	}
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userID := userInfo.Sub

	// Call Cloudflare KV store to save the subdomain
	cfMetadataValue := fmt.Sprintf(`userId=%s;--p=%s;themeMode=%s`, userID, inputPayload.PrimaryColor, inputPayload.ThemeMode)
	metadata := map[string]string{"": ""}
	err = helpers.SetCloudflareMnmOptions(inputPayload.Subdomain, userID, metadata, cfMetadataValue)
	if err != nil {
		if err.Error() == constants.ERR_KV_KEY_EXISTS {
			return transport.SendHtmlErrorText([]byte("Subdomain already taken"), http.StatusInternalServerError)
		} else {
			return transport.SendHtmlErrorText([]byte("Failed to set subdomain: "+err.Error()), http.StatusInternalServerError)
		}
	}

	var buf bytes.Buffer
	var successPartial templ.Component
	if r.URL.Query().Has("theme") {
		successPartial = partials.SuccessBannerHTML(`Theme updated successfully`)
	} else {
		successPartial = partials.SuccessHTMLText(`Subdomain set successfully`)
	}

	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func DeleteMnmSubdomain(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userID := userInfo.Sub

	err := helpers.DeleteSubdomainFromDB(userID)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to delete subdomain: "+err.Error()), http.StatusInternalServerError)
	}

	var buf bytes.Buffer
	successPartial := partials.SuccessHTMLText(`Subdomain deleted successfully`)
	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetEventsPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()

	q, _, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
	}

	mnmOptions := helpers.GetMnmOptionsFromContext(ctx)
	mnmUserId := mnmOptions["userId"]

	// we override the `owners` query param here, because subdomains should always show only
	// the owner as declared authoritatively by the subdomain ID lookup in Cloudflare KV
	if mnmUserId != "" {
		ownerIds = []string{mnmUserId}
	}

	res, err := services.SearchWeaviateEvents(ctx, weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
	}

	events := res.Events
	listMode := r.URL.Query().Get("list_mode")

	// Only sort by StartTime if there's no search query (preserve Weaviate's relevance order)
	if q == "" {
		sort.Slice(events, func(i, j int) bool {
			return events[i].StartTime < events[j].StartTime
		})
	}

	roleClaims := []constants.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
		roleClaims = claims
	}

	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
	}
	userId := ""
	if userInfo.Sub != "" {
		userId = userInfo.Sub
	}
	pageUser := &internal_types.UserSearchResult{
		UserID: userId,
	}

	eventListPartial := pages.EventsInner(events, listMode, roleClaims, userId, pageUser, false, "")

	var buf bytes.Buffer
	err = eventListPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	// Set CORS headers for embed support
	transport.SetCORSHeaders(w, r)

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetEmbedHtml(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	// Get userId from query parameter
	userId := r.URL.Query().Get("userId")
	if userId == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SetCORSHeaders(w, r)
			transport.SendServerRes(w, []byte("userId query parameter is required"), http.StatusBadRequest, errors.New("userId query parameter is required"))
		}
	}

	// Get embedBaseUrl from query parameter (optional, defaults to APEX_URL)
	embedBaseUrl := r.URL.Query().Get("embedBaseUrl")
	if embedBaseUrl == "" {
		embedBaseUrl = os.Getenv("APEX_URL")
	}

	// Use DeriveEventsFromRequest to get events (similar to GetHomeOrUserPage)
	// We need to set the userId in the request context or modify the request
	// For now, let's use GetSearchParamsFromReq and search events directly
	originalQueryLat := r.URL.Query().Get("lat")
	originalQueryLong := r.URL.Query().Get("lon")
	originalQueryLocation := r.URL.Query().Get("location")

	// Create a modified request with userId in the path for DeriveEventsFromRequest
	// Actually, DeriveEventsFromRequest gets userId from mux.Vars, so we need to set it differently
	// Let's use the same approach as GetEventsPartial but get all the data we need
	q, city, userLocation, radius, startTimeUnix, endTimeUnix, cfLocation, _, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	// Override ownerIds to use the userId from query parameter
	ownerIds := []string{userId}

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SetCORSHeaders(w, r)
			transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
		}
	}

	// Search for events
	res, err := services.SearchWeaviateEvents(ctx, weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SetCORSHeaders(w, r)
			transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
		}
	}

	events := res.Events

	// Only sort by StartTime if there's no search query (preserve Weaviate's relevance order)
	if q == "" {
		sort.Slice(events, func(i, j int) bool {
			return events[i].StartTime < events[j].StartTime
		})
	}

	// Get user info for pageUser
	var pageUser *types.UserSearchResult
	userResult, err := helpers.GetOtherUserByID(userId)
	if err == nil && userResult.UserID != "" {
		pageUser = &userResult
		pageUser.UserID = userId
	}

	// Render widget component (reuses HomePage with isEmbed=true)
	widget := pages.Widget(
		ctx,
		events,
		pageUser,
		cfLocation,
		city,
		fmt.Sprint(userLocation[0]),
		fmt.Sprint(userLocation[1]),
		fmt.Sprint(originalQueryLat),
		fmt.Sprint(originalQueryLong),
		originalQueryLocation,
		embedBaseUrl,
	)

	var buf bytes.Buffer
	err = widget.Render(ctx, &buf)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			transport.SetCORSHeaders(w, r)
			transport.SendServerRes(w, []byte("Failed to render widget: "+err.Error()), http.StatusInternalServerError, err)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for embed support
		transport.SetCORSHeaders(w, r)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

func GetProfileInterestsPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userMetaClaims := map[string]interface{}{}
	if _, ok := ctx.Value("userMetaClaims").(map[string]interface{}); ok {
		userMetaClaims = ctx.Value("userMetaClaims").(map[string]interface{})
	}
	parsedInterests := helpers.GetUserInterestFromMap(userMetaClaims, constants.INTERESTS_KEY)

	settingsPartial := pages.ProfileInterestsPartial(parsedInterests)

	var buf bytes.Buffer
	err := settingsPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetSubscriptionsPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	// Get user info from context
	userInfo, ok := ctx.Value("userInfo").(constants.UserInfo)
	if !ok || userInfo.Sub == "" {
		return transport.SendHtmlErrorPartial([]byte("Unauthorized: Missing user ID"), http.StatusUnauthorized)
	}

	zitadelUserID := userInfo.Sub

	// Find Stripe customer using Zitadel user ID
	subscriptionService := services.NewStripeSubscriptionService()
	stripeCustomer, err := subscriptionService.SearchCustomerByExternalID(zitadelUserID)
	if err != nil {
		log.Printf("Error searching for Stripe customer: %v", err)
		// If customer doesn't exist, return empty subscriptions list
		subscriptionsPartial := pages.AdminSubscriptionsPartial([]*internal_types.CustomerSubscription{}, []*internal_types.CustomerSubscription{})
		var buf bytes.Buffer
		renderErr := subscriptionsPartial.Render(ctx, &buf)
		if renderErr != nil {
			return transport.SendHtmlRes(w, []byte(renderErr.Error()), http.StatusInternalServerError, "partial", renderErr)
		}
		return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
	}

	// Check if customer was found (SearchCustomerByExternalID returns nil, nil when not found)
	if stripeCustomer == nil {
		// Customer doesn't exist, return empty subscriptions list
		subscriptionsPartial := pages.AdminSubscriptionsPartial([]*internal_types.CustomerSubscription{}, []*internal_types.CustomerSubscription{})
		var buf bytes.Buffer
		renderErr := subscriptionsPartial.Render(ctx, &buf)
		if renderErr != nil {
			return transport.SendHtmlRes(w, []byte(renderErr.Error()), http.StatusInternalServerError, "partial", renderErr)
		}
		return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
	}

	// Fetch subscriptions using Stripe customer ID
	subscriptions, err := subscriptionService.GetCustomerSubscriptions(stripeCustomer.ID)
	if err != nil {
		log.Printf("Error fetching subscriptions: %v", err)
		return transport.SendHtmlErrorPartial([]byte("Failed to fetch subscriptions: "+err.Error()), http.StatusInternalServerError)
	}

	// Group subscriptions by status (active first)
	var activeSubscriptions []*internal_types.CustomerSubscription
	var otherSubscriptions []*internal_types.CustomerSubscription

	for _, sub := range subscriptions {
		if sub.IsActive() {
			activeSubscriptions = append(activeSubscriptions, sub)
		} else {
			otherSubscriptions = append(otherSubscriptions, sub)
		}
	}

	subscriptionsPartial := pages.AdminSubscriptionsPartial(activeSubscriptions, otherSubscriptions)

	var buf bytes.Buffer
	err = subscriptionsPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetEventAdminChildrenPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	q, _, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	radius = constants.DEFAULT_MAX_RADIUS
	farFutureTime, _ := time.Parse(time.RFC3339, "2099-01-01T00:00:00Z")
	endTimeUnix = farFutureTime.Unix()

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
	}

	eventId := mux.Vars(r)[constants.EVENT_ID_KEY]

	// NOTE: we want the children AND the parent event, empty string gets the parent
	eventSourceIds = []string{eventId}
	eventSourceTypes = []string{constants.ES_EVENT_SERIES, constants.ES_EVENT_SERIES_UNPUB}

	// Separate parent and children events
	var eventParent *internal_types.Event
	var eventChildren []internal_types.Event

	parentChan := make(chan eventParentResult)
	searchChan := make(chan eventSearchResult)

	// Launch parent event fetch in goroutine
	go func() {
		parent, err := services.GetWeaviateEventByID(ctx, weaviateClient, eventId, "")
		parentChan <- eventParentResult{event: parent, err: err}
	}()

	// Launch search in parallel
	go func() {
		res, err := services.SearchWeaviateEvents(ctx, weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
		if err != nil {
			searchChan <- eventSearchResult{err: err}
			return
		}
		searchChan <- eventSearchResult{events: res.Events}
	}()

	// Wait for both results
	parentResult := <-parentChan
	if parentResult.err != nil {
		return transport.SendServerRes(w, []byte("Failed to get event: "+parentResult.err.Error()), http.StatusInternalServerError, parentResult.err)
	}
	eventParent = parentResult.event

	searchResult := <-searchChan
	if searchResult.err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+searchResult.err.Error()), http.StatusInternalServerError, searchResult.err)
	}
	eventChildren = searchResult.events

	// Sort eventChildren by StartTime
	sort.Slice(eventChildren, func(i, j int) bool {
		return eventChildren[i].StartTime < eventChildren[j].StartTime
	})

	eventListPartial := partials.EventAdminChildren(*eventParent, eventChildren)

	var buf bytes.Buffer
	err = eventListPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GeoLookup(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var inputPayload GeoLookupInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, "partial", err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Invalid Body: ")+err.Error()), http.StatusBadRequest, "partial", err)
	}

	baseUrl := constants.GEO_BASE_URL
	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error getting geocoordinates: ")+err.Error()), http.StatusInternalServerError, "partial", err)
	}
	log.Println("GeoLookup lat", lat)
	log.Println("GeoLookup lon", lon)
	log.Println("GeoLookup address", address)
	// Convert lat and lon to float64
	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusInternalServerError, "partial", err)
	}
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusInternalServerError, "partial", err)
	}
	geoLookupPartial := partials.GeoLookup(latFloat, lonFloat, address, "form-hidden")

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func CityLookup(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lon")
		w.Header().Set("Content-Type", "application/json")

		if latStr == "" || lonStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Both lat and lon parameters are required"})
			return
		}

		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid latitude format"})
			return
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid longitude format"})
			return
		}

		latAndLonAreValid := lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180

		if !latAndLonAreValid {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Latitude and Longitude are invalid"})
			return
		}

		cityService := services.GetCityService()
		locationQuery := fmt.Sprintf("%.3f+%.3f", lat, lon)
		city, err := cityService.GetCity(locationQuery)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print("ERR:", err)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Error getting city with query: %s", locationQuery)})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"city": city})
	}
}

func GeoThenPatchSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := transport.GetDB()
		GeoThenPatchSeshuSessionHandler(w, r, db)
	}
}

func GeoThenPatchSeshuSessionHandler(w http.ResponseWriter, r *http.Request, db internal_types.DynamoDBAPI) {
	ctx := r.Context()
	var inputPayload GeoThenSeshuPatchInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, "partial", err)(w, r)
		return
	}
	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, "partial", err)(w, r)
		return
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid Body: "+err.Error()), http.StatusBadRequest, "partial", err)(w, r)
		return
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, "partial", err)(w, r)
		return
	}

	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to get geocoordinates: "+err.Error()), http.StatusInternalServerError, "partial", err)(w, r)
		return
	}
	updateSeshuSession := internal_types.SeshuSessionUpdate{
		Url: inputPayload.Url, // Map source_url to Url field
	}

	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, "partial", err)(w, r)
		return
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusUnprocessableEntity, "partial", err)(w, r)
		return
	}

	updateSeshuSession.LocationLatitude = &latFloat
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusUnprocessableEntity, "partial", err)(w, r)
		return
	}
	updateSeshuSession.LocationLongitude = &lonFloat
	updateSeshuSession.LocationAddress = address

	if updateSeshuSession.Url == "" {
		transport.SendHtmlRes(w, []byte("ERR: Invalid body: url is required"), http.StatusBadRequest, "partial", nil)(w, r)
		return
	}
	geoLookupPartial := partials.GeoLookup(latFloat, lonFloat, address, "badge")

	_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to update target URL session"), http.StatusNotFound, "partial", err)(w, r)
		return
	}

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)(w, r)
		return
	}
	transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)(w, r)
}

func SubmitSeshuEvents(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	db := transport.GetDB()

	ctx := r.Context()
	var inputPayload SeshuSessionEventsPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, "partial", err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, "partial", err)
	}

	// Note that only OpenAI can push events as candidates, `eventValidations` is an array of
	// arrays that confirms the subfields, but avoids a scenario where users can push string data
	// that is prone to manipulation
	// check current session payload

	//trim whitespaces for url
	inputPayload.Url = strings.TrimSpace(inputPayload.Url)

	updateSeshuSession = internal_types.SeshuSessionUpdate{
		Url:              inputPayload.Url,
		EventValidations: inputPayload.EventBoolValid,
	}

	seshuService := services.GetSeshuService()
	_, err = seshuService.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
	}

	// Updating the parent and child
	if len(inputPayload.EventRecursiveBoolValid) > 0 {

		parentSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
			Url: inputPayload.Url,
		})
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get parent session"), http.StatusInternalServerError, "partial", err)
		}

		childSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
			Url: parentSession.ChildId,
		})
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get child session"), http.StatusInternalServerError, "partial", err)
		}

		updatedCandidates := parentSession.EventCandidates
		updatedValidations := parentSession.EventValidations

		childCandidate := childSession.EventCandidates[0]
		childValidation := inputPayload.EventRecursiveBoolValid[0]

		updated := false
		for i, parentCandidate := range parentSession.EventCandidates {
			if parentCandidate.EventURL == childCandidate.EventURL {
				updatedCandidates[i] = childCandidate
				updatedValidations[i] = childValidation
				updated = true
				break
			}
		}

		if !updated {
			return transport.SendHtmlRes(w, []byte("Unable to find child session"), http.StatusInternalServerError, "partial", err)
		}

		parentUpdate := internal_types.SeshuSessionUpdate{
			Url:              parentSession.Url,
			EventCandidates:  updatedCandidates,
			EventValidations: updatedValidations,
		}

		_, err = seshuService.UpdateSeshuSession(ctx, db, parentUpdate)

		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update parent session with child data"), http.StatusBadRequest, "partial", err)
		}

		childUpdate := internal_types.SeshuSessionUpdate{
			Url:              parentSession.ChildId,
			EventValidations: []internal_types.EventBoolValid{inputPayload.EventRecursiveBoolValid[0]},
		}

		_, err = seshuService.UpdateSeshuSession(ctx, db, childUpdate)

		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
		}
	}

	successPartial := partials.SuccessBannerHTML(`We've noted the events you've confirmed as accurate`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func isFakeData(val string) bool {
	switch val {
	case services.FakeCity:
		return true
	case services.FakeUrl1:
		return true
	case services.FakeUrl2:
		return true
	case services.FakeEventTitle1:
		return true
	case services.FakeEventTitle2:
		return true
	case services.FakeStartTime1:
		return true
	case services.FakeStartTime2:
		return true
	case services.FakeEndTime1:
		return true
	case services.FakeEndTime2:
		return true
	}
	return false
}

func getValidatedEvents(candidates []internal_types.EventInfo, validations []internal_types.EventBoolValid, hasDefaultLocation bool) []internal_types.EventInfo {
	var validatedEvents []internal_types.EventInfo

	for i := range candidates {

		isValid := true

		// Check if we have a corresponding validation for this candidate
		if i >= len(validations) {
			// If no validation exists for this candidate, skip it
			continue
		}

		// Validate Event Title
		if candidates[i].EventTitle == "" || isFakeData(candidates[i].EventTitle) || !validations[i].EventValidateTitle {
			isValid = false
		}

		// Validate Event Location (Only if no default location is provided)
		if !hasDefaultLocation {
			if candidates[i].EventLocation == "" || isFakeData(candidates[i].EventLocation) || !validations[i].EventValidateLocation {
				isValid = false
			}
		}

		// Validate Event Start Time
		if candidates[i].EventStartTime == "" || isFakeData(candidates[i].EventStartTime) || !validations[i].EventValidateStartTime {
			isValid = false
		}

		// If valid, add event to list
		if isValid {
			validatedEvents = append(validatedEvents, candidates[i])
		}

	}
	return validatedEvents
}

// TODO: I have no idea if this actually works or not, this is provided by ChatGPT 4o
// I'm leaving this unifinished to go work on other more urgent items, please fix
// change, or remove this as needed

// func findTextSubstring(doc *goquery.Document, substring string) (string, bool) {
// 	var path string
// 	found := false

// 	// Recursive function to traverse nodes and build the path
// 	var traverse func(*goquery.Selection, string)
// 	traverse = func(s *goquery.Selection, currentPath string) {
// 		if found {
// 			return
// 		}

// 		s.Contents().Each(func(i int, node *goquery.Selection) {
// 			if found {
// 				return
// 			}

// 			nodeText := node.Text()
// 			if strings.Contains(nodeText, substring) {
// 				path = currentPath
// 				found = true
// 				return
// 			}

// 			// Build path for the current node
// 			nodeTag := goquery.NodeName(node)
// 			// TODO: this definitely looks like incorrect LLM output
// 			nodePath := fmt.Sprintf("%s > %s:nth-child(%d)", currentPath, nodeTag, i+1)

// 			traverse(node, nodePath)
// 		})
// 	}

// 	// Start traversing from the document root
// 	traverse(doc.Selection, "html")

// 	return path, found
// }

func SubmitSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	dDb := transport.GetDB()
	// natsService, _ := services.GetNatsService(ctx)

	var inputPayload SeshuSessionEventsPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	var payload SeshuSessionEventsPayload
	err = json.Unmarshal([]byte(body), &payload)
	if err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userId := userInfo.Sub
	if userId == "" {
		return transport.SendHtmlRes(w, []byte("You must be logged in to submit an event source"), http.StatusUnauthorized, "partial", err)
	}

	var seshuSessionGet internal_types.SeshuSessionGet
	//validate url trim whitespaces
	payload.Url = strings.TrimSpace(payload.Url)
	seshuSessionGet.OwnerId = userId
	seshuSessionGet.Url = payload.Url
	seshuService := services.GetSeshuService()

	session, err := seshuService.GetSeshuSession(ctx, dDb, seshuSessionGet)
	if err != nil {
		log.Println("Failed to get SeshuSession. ID: ", session, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	inputPayload.EventBoolValid = session.EventValidations

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, "partial", err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, "partial", err)
	}

	ctx = r.Context()
	postgresDB, err := services.GetPostgresService(ctx)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to initialize services"), http.StatusInternalServerError, "partial", err)
	}

	// set context value for url
	ctx = context.WithValue(ctx, "targetUrl", inputPayload.Url)

	jobs, err := postgresDB.GetSeshuJobs(ctx)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to get SeshuJobs"), http.StatusInternalServerError, "partial", err)
	}
	jobAborted := false
	if len(jobs) > 0 {
		jobAborted = true
		return transport.SendHtmlRes(w, []byte("This event source URL already exists"), http.StatusConflict, "partial", err)

	}

	log.Printf("INFO: Submitting SeshuJob for URL: %s", inputPayload.Url)

	defer func() {
		if jobAborted {
			return
		}
		// check for valid latitude / longitude that is NOT equal to `constants.INITIAL_EMPTY_LAT_LONG`
		// which is an intentionally invalid placeholder

		hasDefaultLat := false
		latMatch, err := regexp.MatchString(services.LatitudeRegex, fmt.Sprint(session.LocationLatitude))
		if session.LocationLatitude == constants.INITIAL_EMPTY_LAT_LONG {
			hasDefaultLat = false
		} else if err != nil || !latMatch {
			hasDefaultLat = true
		}

		hasDefaultLon := false
		lonMatch, err := regexp.MatchString(services.LongitudeRegex, fmt.Sprint(session.LocationLongitude))
		if session.LocationLongitude == constants.INITIAL_EMPTY_LAT_LONG {
			hasDefaultLon = false
		} else if err != nil || !lonMatch || session.LocationLongitude == constants.INITIAL_EMPTY_LAT_LONG {
			hasDefaultLon = true
		}

		validatedEvents := getValidatedEvents(session.EventCandidates, session.EventValidations, hasDefaultLat && hasDefaultLon)

		// TODO: search `session.Html` for the items in the `validatedEvents` array
		if len(validatedEvents) == 0 {
			log.Println("No validated events found, aborting job creation")
			return
		}

		// parentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
		// 	if err != nil {
		// 		log.Println("Error parsing parent HTML:", err)
		// }

		// // Optionally parse child HTML if child session exists
		// var childDoc *goquery.Document
		// if session.ChildId != "" {
		// 	childSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
		// 		Url: session.ChildId,
		// 	})
		// 	if err != nil {
		// 		log.Println("Could not retrieve child session:", err)
		// 	} else {
		// 		childDoc, err = goquery.NewDocumentFromReader(strings.NewReader(childSession.Html))
		// 		if err != nil {
		// 			log.Println("Failed to parse child HTML:", err)
		// 		}
		// 	}
		// }

		//URL as a key --

		// TODO: [0] is just a placeholder, should be a loop over `validatedEvents` array and search for each
		// or maybe once it finds the first one that's good enough? Walking a long array might be wasted compute
		// if the first one is good enough

		// Find the path to the text substring
		// TODO: this is commented out because it's not verified and I don't want to introduce regression,
		// uncomment this and figure out if it this approach works for traversing the DOM

		// NOTE: I think searching the DOM string for `>{ validatedEvents[0].EventTitle }<` and then backtracing
		// to get a full DOM Querystring path is a better approach because `validatedEvents[0].EventTitle` can appear
		// in HTML attributes in my testing, we want to find in where it's the opening of an HTML tag

		// doc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
		// if err != nil {
		// 	log.Println("Failed to parse HTML document: ", err)
		// }

		// substring := validatedEvents[0].EventTitle
		// path, found := findTextSubstring(doc, substring)
		// if found {
		// 	fmt.Printf("Text '%s' found at path: %s\n", substring, path)
		// } else {
		// 	fmt.Printf("Text '%s' not found\n", substring)
		// }

		// TODO: delete this `SeshuSession` once the handoff to the `SeshuJobs` table is complete

		// Assumming any event is the one to use for DOM path finding, make sure that it is not rs
		scrapeSource := "unknown"
		var docToUse *goquery.Document
		geoService := services.GetGeoService()

		var anchorEvent *internal_types.EventInfo
		var childEvent *internal_types.EventInfo
		var seshuJob internal_types.SeshuJob
		var scrapeType string

		for i := range validatedEvents {
			e := &validatedEvents[i]
			if e.ScrapeMode != "rs" && anchorEvent == nil {
				anchorEvent = e
			} else if e.ScrapeMode == "rs" {
				childEvent = e
			}
		}

		var scheduledHour = (time.Now().UTC().Hour() + 23) % 24

		if anchorEvent != nil {
			// Find the DOM paths for the main event
			var location string
			var titleTag string
			var locationTag string
			var startTag string
			var endTag string
			var descriptionTag string
			var eventURLTag string

			parentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
			if err != nil {
				log.Println("Error parsing parent HTML:", err)
			}

			docToUse = parentDoc
			// Find the DOM paths for the main event
			//normalise url for consistency
			normalizedUrl, err := helpers.NormalizeURL(session.Url)
			if err != nil {
				log.Println("Error normalizing URL:", err)
			}

			location = session.LocationAddress

			baseUrl, err := helpers.ExtractBaseDomain(session.Url)
			if err != nil {
				log.Println("Error extracting base domain:", err)
			}

			// Infer known scrape source from base domain
			if strings.Contains(baseUrl, "facebook.com") {
				scrapeSource = constants.SESHU_KNOWN_SOURCE_FB
			}

			switch scrapeSource {
			case constants.SESHU_KNOWN_SOURCE_FB:
				titleTag = "_BYPASS_"
				locationTag = "_BYPASS_"
				startTag = "_BYPASS_"
				endTag = "_BYPASS_"
				descriptionTag = "_BYPASS_"
				eventURLTag = "_BYPASS_"
			default:
				titleTag = findTagByExactText(docToUse, anchorEvent.EventTitle)
				locationTag = findTagByExactText(docToUse, anchorEvent.EventLocation)
				startTag = findTagByExactText(docToUse, anchorEvent.EventStartTime)

				if anchorEvent.EventEndTime == "" {
					endTag = ""
				} else {
					endTag = findTagByPartialText(docToUse, anchorEvent.EventEndTime)
				}

				if anchorEvent.EventDescription == "" {
					descriptionTag = ""
				} else {
					// Use half the length of the description to find a partial match
					descriptionTag = findTagByPartialText(docToUse, anchorEvent.EventDescription[:utf8.RuneCountInString(anchorEvent.EventDescription)/2])
				}

				if anchorEvent.EventURL == "" {
					eventURLTag = ""
				} else {
					eventURLTag = findTagByExactText(docToUse, anchorEvent.EventURL)
				}
			}

			var anchorLatFloat, anchorLonFloat float64
			if session.LocationLatitude == constants.INITIAL_EMPTY_LAT_LONG || session.LocationLongitude == constants.INITIAL_EMPTY_LAT_LONG {
				lat, lon, _, err := geoService.GetGeo(location, constants.GEO_BASE_URL)
				if err != nil {
					log.Println("Error getting geocoordinates for session:", err)
				}
				anchorLatFloat, err = strconv.ParseFloat(lat, 64)
				if err != nil {
					log.Println("Invalid latitude value for session:", err)
				}
				anchorLonFloat, err = strconv.ParseFloat(lon, 64)
				if err != nil {
					log.Println("Invalid longitude value for session:", err)
				}
			}

			locationTimezone := services.DeriveTimezoneFromCoordinates(anchorLatFloat, anchorLonFloat)

			seshuJob = internal_types.SeshuJob{
				NormalizedUrlKey:         normalizedUrl,
				LocationLatitude:         session.LocationLatitude,
				LocationLongitude:        session.LocationLongitude,
				LocationAddress:          location,
				LocationTimezone:         locationTimezone,
				ScheduledHour:            scheduledHour,
				TargetNameCSSPath:        titleTag,
				TargetLocationCSSPath:    locationTag,
				TargetStartTimeCSSPath:   startTag,
				TargetEndTimeCSSPath:     endTag,         // optional
				TargetDescriptionCSSPath: descriptionTag, // optional
				TargetHrefCSSPath:        eventURLTag,
				Status:                   "HEALTHY", // assume healthy if parse succeeded
				IsRecursive:              false,
				LastScrapeSuccess:        time.Now().Unix(),
				LastScrapeFailure:        0,
				LastScrapeFailureCount:   0,
				OwnerID:                  session.OwnerId, // ideally from auth context
				KnownScrapeSource:        scrapeSource,    // or infer from URL pattern/domain
			}

			//if rs exist
			if childEvent != nil {
				var childDoc *goquery.Document
				var titleTag string
				var locationTag string
				var startTag string
				var endTag string
				var descriptionTag string

				normalizedChildURL, err := helpers.NormalizeURL(childEvent.EventURL)
				if err != nil || normalizedChildURL == "" {
					log.Println("Error normalizing URL, falling back to raw:", err)
					normalizedChildURL = childEvent.EventURL
				}

				childSession, err := seshuService.GetSeshuSession(ctx, dDb, internal_types.SeshuSessionGet{
					Url: normalizedChildURL,
				})
				if err != nil || childSession == nil {
					log.Println("Could not retrieve child session:", err)
				}
				childDoc, err = goquery.NewDocumentFromReader(strings.NewReader(childSession.Html))
				if err != nil {
					log.Println("Failed to parse child HTML:", err)
				}

				// Infer known scrape source from base domain
				if strings.Contains(baseUrl, "facebook.com") {
					scrapeSource = constants.SESHU_KNOWN_SOURCE_FB
				}

				switch scrapeSource {
				case constants.SESHU_KNOWN_SOURCE_FB:
					titleTag = "_BYPASS_"
					locationTag = "_BYPASS_"
					startTag = "_BYPASS_"
					endTag = "_BYPASS_"
					descriptionTag = "_BYPASS_"
					eventURLTag = "_BYPASS_"
					titleTag = "_BYPASS_"
					locationTag = "_BYPASS_"
					startTag = "_BYPASS_"
					endTag = "_BYPASS_"
					descriptionTag = "_BYPASS_"
				default:
					titleTag = findTagByExactText(childDoc, childEvent.EventTitle)
					locationTag = findTagByExactText(childDoc, childEvent.EventLocation)
					startTag = findTagByExactText(childDoc, childEvent.EventStartTime)

					if childEvent.EventEndTime == "" {
						endTag = ""
					} else {
						endTag = findTagByPartialText(childDoc, childEvent.EventEndTime)
					}

					if childEvent.EventDescription == "" {
						descriptionTag = ""
					} else {
						// Use half the length of the description to find a partial match
						descriptionTag = findTagByPartialText(childDoc, childEvent.EventDescription[:utf8.RuneCountInString(childEvent.EventDescription)/2])
					}

					if childEvent.EventURL == "" {
						eventURLTag = ""
					} else {
						eventURLTag = findTagByPartialText(childDoc, childEvent.EventURL)
					}

					// Seshu children DOM Path
					seshuJob.TargetChildNameCSSPath = titleTag
					seshuJob.TargetChildLocationCSSPath = locationTag
					seshuJob.TargetChildStartTimeCSSPath = startTag
					seshuJob.TargetChildEndTimeCSSPath = endTag
					seshuJob.TargetChildDescriptionCSSPath = descriptionTag
				}
			}

			scrapeType = "init" // initial scrape + optional rs

		} else if childEvent != nil && anchorEvent == nil {
			var childDoc *goquery.Document
			var titleTag string
			var locationTag string
			var startTag string
			var endTag string
			var descriptionTag string
			var eventURLTag string

			normalizedChildURL, err := helpers.NormalizeURL(childEvent.EventURL)
			if err != nil || normalizedChildURL == "" {
				log.Println("Error normalizing URL, falling back to raw:", err)
				normalizedChildURL = childEvent.EventURL
			}

			childSession, err := seshuService.GetSeshuSession(ctx, dDb, internal_types.SeshuSessionGet{
				Url: normalizedChildURL,
			})
			if err != nil || childSession == nil {
				log.Println("Could not retrieve child session:", err)
			}
			childDoc, err = goquery.NewDocumentFromReader(strings.NewReader(childSession.Html))
			if err != nil {
				log.Println("Failed to parse child HTML:", err)
			}

			// Infer known scrape source from base domain
			if strings.Contains(childEvent.EventURL, "facebook.com") {
				scrapeSource = constants.SESHU_KNOWN_SOURCE_FB
			}

			switch scrapeSource {
			case constants.SESHU_KNOWN_SOURCE_FB:
				titleTag = "_BYPASS_"
				locationTag = "_BYPASS_"
				startTag = "_BYPASS_"
				endTag = "_BYPASS_"
				descriptionTag = "_BYPASS_"
				eventURLTag = "_BYPASS_"
			default:
				titleTag = findTagByExactText(childDoc, childEvent.EventTitle)
				locationTag = findTagByExactText(childDoc, childEvent.EventLocation)
				startTag = findTagByExactText(childDoc, childEvent.EventStartTime)

				if childEvent.EventEndTime == "" {
					endTag = ""
				} else {
					endTag = findTagByPartialText(childDoc, childEvent.EventEndTime)
				}

				if childEvent.EventDescription == "" {
					descriptionTag = ""
				} else {
					// Use half the length of the description to find a partial match
					descriptionTag = findTagByPartialText(childDoc, childEvent.EventDescription[:utf8.RuneCountInString(childEvent.EventDescription)/2])
				}

				if childEvent.EventURL == "" {
					eventURLTag = ""
				} else {
					eventURLTag = findTagByPartialText(childDoc, childEvent.EventURL)
				}

				var anchorLatFloat, anchorLonFloat float64
				if childSession.LocationLatitude == constants.INITIAL_EMPTY_LAT_LONG || childSession.LocationLongitude == constants.INITIAL_EMPTY_LAT_LONG {
					lat, lon, _, err := geoService.GetGeo(childEvent.EventLocation, constants.GEO_BASE_URL)
					if err != nil {
						log.Println("Error getting geocoordinates for session:", err)
					}
					anchorLatFloat, err = strconv.ParseFloat(lat, 64)
					if err != nil {
						log.Println("Invalid latitude value for session:", err)
					}
					anchorLonFloat, err = strconv.ParseFloat(lon, 64)
					if err != nil {
						log.Println("Invalid longitude value for session:", err)
					}
				}

				locationTimezone := services.DeriveTimezoneFromCoordinates(anchorLatFloat, anchorLonFloat)

				seshuJob = internal_types.SeshuJob{
					NormalizedUrlKey:         normalizedChildURL,
					LocationLatitude:         childSession.LocationLatitude,
					LocationLongitude:        childSession.LocationLongitude,
					LocationAddress:          childEvent.EventLocation,
					LocationTimezone:         locationTimezone,
					ScheduledHour:            scheduledHour,
					TargetNameCSSPath:        titleTag,
					TargetLocationCSSPath:    locationTag,
					TargetStartTimeCSSPath:   startTag,
					TargetEndTimeCSSPath:     endTag,         // optional
					TargetDescriptionCSSPath: descriptionTag, // optional
					TargetHrefCSSPath:        eventURLTag,
					Status:                   "HEALTHY", // assume healthy if parse succeeded
					IsRecursive:              true,
					LastScrapeSuccess:        time.Now().Unix(),
					LastScrapeFailure:        0,
					LastScrapeFailureCount:   0,
					OwnerID:                  session.OwnerId, // ideally from auth context
					KnownScrapeSource:        scrapeSource,    // or infer from URL pattern/domain
				}

				scrapeType = "rs" // rs only
			}
		}

		pgDb, _ := services.GetPostgresService(ctx)

		err = validate.Struct(seshuJob)
		if err != nil {
			log.Println("Error validating SeshuJob:", err)
			return
		}

		err = pgDb.CreateSeshuJob(ctx, seshuJob)
		if err != nil {
			log.Println("Error creating SeshuJob:", err)
			return
		}

		// NOTE: `natsService.PublishMsg` would put the job in the queue, but we don't yet
		// have a priority queue so we're instead writing directly to the DB
		// in the go func below

		// err = natsService.PublishMsg(ctx, seshuJob)
		// if err != nil {
		// 	log.Println("Failed to publish seshuJob to NATS:", err)
		// }

		go func() {

			if jobAborted {
				return
			}

			extractedEvents, _, err := services.ExtractEventsFromHTML(seshuJob, constants.SESHU_MODE_SCRAPE, scrapeType, &services.RealScrapingService{})
			if err != nil {
				log.Printf("Failed to extract events from %s: %v", seshuJob.NormalizedUrlKey, err)
			}

			if len(extractedEvents) == 0 {
				log.Printf("No events extracted from %s", seshuJob.NormalizedUrlKey)
			} else {
				log.Printf("Extracted %d events from %s", len(extractedEvents), seshuJob.NormalizedUrlKey)
			}

			err = services.PushExtractedEventsToDB(extractedEvents, seshuJob, make(map[string]string))
			if err != nil {
				log.Println("Error pushing ingested events to DB:", err)
			}

		}()

	}()

	updateSeshuSession.Url = inputPayload.Url
	updateSeshuSession.Status = "submitted"

	_, err = services.UpdateSeshuSession(ctx, dDb, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
	}

	if session.ChildId != "" {
		updateSeshuSession.Url = session.ChildId
		updateSeshuSession.Status = "submitted"

		_, err = services.UpdateSeshuSession(ctx, dDb, updateSeshuSession)
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
		}
	}

	successPartial := partials.SuccessBannerHTML(`Your Event Source has been added. We will put it in the queue and let you know when it's imported.`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func UpdateUserInterests(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	r.ParseForm()
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userID := userInfo.Sub

	// TODO: pretty sure the Form approach here makes it so that you can't submit this multiple
	// times in succession when using the profile settings page
	categories := r.Form

	// Use a map to track unique elements
	flattenedCategories := []string{}

	// Flatten and split by "|", then add to the map to remove duplicates
	for key, values := range categories {
		if strings.HasSuffix(key, "category") || strings.HasSuffix(key, "subCategory") {
			for _, value := range values {
				// Split by comma and trim spaces, in case there are multiple values
				for _, item := range strings.Split(value, ",") {
					trimmedItem := strings.TrimSpace(item)
					if trimmedItem != "" {
						flattenedCategories = append(flattenedCategories, trimmedItem)
					}
				}
			}
		}
	}

	flattenedCategoriesString := strings.Join(flattenedCategories, "|")

	err := helpers.UpdateUserMetadataKey(userID, constants.INTERESTS_KEY, flattenedCategoriesString)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to save interests: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML(`Your interests have been updated successfully.`)
	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func UpdateUserAbout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload UpdateUserAboutRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError)
	}
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userID := userInfo.Sub
	err = helpers.UpdateUserMetadataKey(userID, constants.META_ABOUT_KEY, inputPayload.About)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to update 'about' field: "+err.Error()), http.StatusInternalServerError)
	}

	var buf bytes.Buffer
	successPartial := partials.SuccessBannerHTML(`About section successfully saved`)

	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func UpdateUserLocation(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload UpdateUserLocationRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	// Validate struct fields
	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid Body: "+err.Error()), http.StatusBadRequest)
	}

	if strings.TrimSpace(inputPayload.City) == "" {
		return transport.SendHtmlErrorPartial([]byte("City field is required"), http.StatusBadRequest)
	}

	if inputPayload.Latitude < -90 || inputPayload.Latitude > 90 {
		return transport.SendHtmlErrorPartial([]byte("Latitude must be between -90 and 90"), http.StatusBadRequest)
	}

	if inputPayload.Longitude < -180 || inputPayload.Longitude > 180 {
		return transport.SendHtmlErrorPartial([]byte("Longitude must be between -180 and 180"), http.StatusBadRequest)
	}

	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userID := userInfo.Sub

	userLocation := fmt.Sprintf("%s;%.2f;%.2f", inputPayload.City, inputPayload.Latitude, inputPayload.Longitude)

	err = helpers.UpdateUserMetadataKey(userID, constants.META_LOC_KEY, userLocation)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to update location: "+err.Error()), http.StatusInternalServerError)
	}

	// This can be used to see what actually gets saved in Zitadel. But it's an extra api call and log so I'm leaving it commented out.
	// loc, _ := helpers.GetUserMetadataByKey(userID, constants.META_LOC_KEY)
	// locStr, _ := base64.StdEncoding.DecodeString(loc)
	// log.Printf("Location info successfully saved: %s", locStr)

	var buf bytes.Buffer
	// this doesn't get displayed in UI, it's just for the tests to verify success
	buf.WriteString("Location info successfully saved.")
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func getFullDomPath(element *goquery.Selection) string {
	var path []string

	// Traverse up the parent elements
	for node := element; node.Length() > 0; node = node.Parent() {
		tag := goquery.NodeName(node)

		// Get unique identifiers (ID or first class)
		id, existsID := node.Attr("id")
		if existsID {
			path = append([]string{fmt.Sprintf("%s#%s", tag, id)}, path...)
			break // IDs are unique, stop traversal
		}

		class, existsClass := node.Attr("class")
		if existsClass {
			classes := strings.Fields(class)
			if len(classes) > 0 {
				path = append([]string{fmt.Sprintf("%s.%s", tag, classes[0])}, path...)
				continue
			}
		}

		// If no ID or class, just append the tag
		path = append([]string{tag}, path...)
	}

	return strings.Join(path, " > ") // Return full selector path
}

func findTagByExactText(doc *goquery.Document, targetText string) string {
	var exactMatch *goquery.Selection

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Get only the element's own text (exclude children)
		nodeText := strings.TrimSpace(s.Clone().Children().Remove().End().Text())

		if nodeText == targetText {
			exactMatch = s
		}
	})

	if exactMatch != nil {
		return getFullDomPath(exactMatch)
	}
	return ""
}

func findTagByPartialText(doc *goquery.Document, targetSubstring string) string {
	var bestMatch *goquery.Selection

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Normalize and preserve space between children
		text := strings.Join(s.Contents().Map(func(i int, c *goquery.Selection) string {
			return strings.TrimSpace(c.Text()) // → "Wednesday7pm"
		}), " ") // → "Wednesday 7pm"

		if strings.Contains(text, targetSubstring) {
			hasChildMatch := false

			s.Children().Each(func(i int, child *goquery.Selection) {
				childText := strings.Join(child.Contents().Map(func(i int, c *goquery.Selection) string {
					return strings.TrimSpace(c.Text())
				}), " ")
				if strings.Contains(childText, targetSubstring) {
					hasChildMatch = true
				}
			})

			if !hasChildMatch {
				bestMatch = s
			}
		}
	})

	if bestMatch != nil {
		return getFullDomPath(bestMatch)
	}
	return ""
}

// GetEmbedScript returns the embed script that loads the widget HTML
func GetEmbedScript(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Get STATIC_BASE_URL from environment for base URL detection
	staticBaseUrl := os.Getenv("STATIC_BASE_URL")
	if staticBaseUrl == "" {
		staticBaseUrl = "http://localhost:8001"
	}

	// Build JavaScript code with Steps 1, 2, and 3
	script := fmt.Sprintf(`(function() {
		'use strict';

		// Step 1: Container Setup
		// Find or create container element
		var containerId = 'mnm-embed-container';
		var container = document.getElementById(containerId);

		// Check for data-mnm-container attribute on script tag as alternative
		var scripts = document.getElementsByTagName('script');
		for (var i = 0; i < scripts.length; i++) {
			var customContainerId = scripts[i].getAttribute('data-mnm-container');
			if (customContainerId) {
				containerId = customContainerId;
				container = document.getElementById(containerId);
				break;
			}
		}

		// Create container if it doesn't exist
		if (!container) {
			container = document.createElement('div');
			container.id = containerId;
			document.body.appendChild(container);
			console.log('MeetNearMe Embed: Created container element with id: ' + containerId);
		} else {
			console.log('MeetNearMe Embed: Found existing container element with id: ' + containerId);
		}

		// Step 2: User ID Detection
		var userId = null;

		// Try to get userId from data-user-id attribute on script tag (primary method)
		for (var i = 0; i < scripts.length; i++) {
			var scriptUserId = scripts[i].getAttribute('data-user-id');
			if (scriptUserId) {
				userId = scriptUserId;
				console.log('MeetNearMe Embed: Found userId from data-user-id attribute: ' + userId);
				break;
			}
		}

		// Fallback: Try to get userId from query parameter in script URL
		if (!userId) {
			var currentScript = scripts[scripts.length - 1]; // Get the last script (should be this one)
			if (currentScript && currentScript.src) {
				var url = new URL(currentScript.src);
				var urlUserId = url.searchParams.get('userId');
				if (urlUserId) {
					userId = urlUserId;
					console.log('MeetNearMe Embed: Found userId from query parameter: ' + userId);
				}
			}
		}

		// Show error if userId is missing
		if (!userId) {
			var errorMsg = '<div style="padding: 1rem; background-color: #fee; border: 1px solid #fcc; border-radius: 0.5rem; color: #c33;">MeetNearMe Embed Error: userId is required. Please add data-user-id="YOUR_USER_ID" to the script tag or include ?userId=YOUR_USER_ID in the script URL.</div>';
			container.innerHTML = errorMsg;
			console.error('MeetNearMe Embed: userId is required but not found. Add data-user-id="YOUR_USER_ID" to the script tag.');
			return;
		}

		// Step 3: Base URL Detection
		var staticBaseUrlFromEnv = '%s';
		var staticBaseUrl;
		var baseUrl;

		// Determine static base URL (for CSS/assets)
		if (staticBaseUrlFromEnv && staticBaseUrlFromEnv !== '') {
			staticBaseUrl = staticBaseUrlFromEnv;
			console.log('MeetNearMe Embed: Using STATIC_BASE_URL from environment: ' + staticBaseUrl);
		} else {
			// Default to localhost:8001 for local development
			staticBaseUrl = 'http://localhost:8001';
			console.log('MeetNearMe Embed: Using default static base URL for local development: ' + staticBaseUrl);
		}

		// Determine API base URL (for API calls) - use current script's origin
		var currentScript = scripts[scripts.length - 1];
		if (currentScript && currentScript.src) {
			var scriptUrl = new URL(currentScript.src);
			baseUrl = scriptUrl.origin;
			console.log('MeetNearMe Embed: Using script origin as API base URL: ' + baseUrl);
		} else {
			// Fallback to window.location.origin
			baseUrl = window.location.origin;
			console.log('MeetNearMe Embed: Using window.location.origin as API base URL: ' + baseUrl);
		}

		console.log('MeetNearMe Embed: API Base URL: ' + baseUrl + ', Static Base URL: ' + staticBaseUrl);

		// Step 4: Dependency Loading
		// Check which dependencies are already loaded
		var dependencies = {
			alpine: false,
			htmx: false,
			tailwind: false,
			mainCss: false,
			fonts: false,
			focusPlugin: false
		};

		// Check for Alpine.js
		if (window.Alpine) {
			dependencies.alpine = true;
			console.log('MeetNearMe Embed: Alpine.js already loaded');
		} else {
			console.log('MeetNearMe Embed: Alpine.js not found, will load from CDN');
		}

		// Check for HTMX
		if (window.htmx) {
			dependencies.htmx = true;
			console.log('MeetNearMe Embed: HTMX already loaded');
		} else {
			console.log('MeetNearMe Embed: HTMX not found, will load from CDN');
		}

		// Check for Tailwind CSS (check if Tailwind CDN script exists or if Tailwind classes work)
		var tailwindScript = document.querySelector('script[src*="tailwindcss.com"]');
		if (tailwindScript || (window.tailwind && window.tailwind.config)) {
			dependencies.tailwind = true;
			console.log('MeetNearMe Embed: Tailwind CSS already loaded');
		} else {
			console.log('MeetNearMe Embed: Tailwind CSS not found, will load from CDN');
		}

		// Check for main CSS (styles.css or hashed version)
		var mainCssLink = document.querySelector('link[href*="styles"]');
		if (mainCssLink) {
			dependencies.mainCss = true;
			console.log('MeetNearMe Embed: Main CSS already loaded');
		} else {
			console.log('MeetNearMe Embed: Main CSS not found, will load from static server');
		}

		// Check for Google Fonts
		var fontsLink = document.querySelector('link[href*="fonts.googleapis.com"]');
		if (fontsLink) {
			dependencies.fonts = true;
			console.log('MeetNearMe Embed: Google Fonts already loaded');
		} else {
			console.log('MeetNearMe Embed: Google Fonts not found, will load from CDN');
		}

		// Check for Alpine.js Focus plugin
		var focusPluginScript = document.querySelector('script[src*="@alpinejs/focus"]');
		if (focusPluginScript) {
			dependencies.focusPlugin = true;
			console.log('MeetNearMe Embed: Alpine.js Focus plugin already loaded');
		} else {
			console.log('MeetNearMe Embed: Alpine.js Focus plugin not found, will load from CDN');
		}

		// Function to load a script and return a promise
		function loadScript(src, name) {
			return new Promise(function(resolve, reject) {
				// Check if already loaded
				var existing = document.querySelector('script[src="' + src + '"]');
				if (existing) {
					console.log('MeetNearMe Embed: ' + name + ' already loaded from ' + src);
					resolve();
					return;
				}

				console.log('MeetNearMe Embed: Loading ' + name + ' from ' + src);
				var script = document.createElement('script');
				script.src = src;
				script.crossOrigin = 'anonymous';
				script.onload = function() {
					console.log('MeetNearMe Embed: Successfully loaded ' + name);
					resolve();
				};
				script.onerror = function(error) {
					console.error('MeetNearMe Embed: Failed to load ' + name + ' from ' + src, error);
					reject(new Error('Failed to load ' + name + ' from ' + src));
				};
				document.head.appendChild(script);
			});
		}

		// Function to load a stylesheet and return a promise
		function loadStylesheet(href, name) {
			return new Promise(function(resolve, reject) {
				// Check if already loaded
				var existing = document.querySelector('link[href="' + href + '"]');
				if (existing) {
					console.log('MeetNearMe Embed: ' + name + ' already loaded from ' + href);
					resolve();
					return;
				}

				console.log('MeetNearMe Embed: Loading ' + name + ' from ' + href);
				var link = document.createElement('link');
				link.rel = 'stylesheet';
				link.href = href;
				link.crossOrigin = 'anonymous';
				link.onload = function() {
					console.log('MeetNearMe Embed: Successfully loaded ' + name);
					resolve();
				};
				link.onerror = function(error) {
					console.error('MeetNearMe Embed: Failed to load ' + name + ' from ' + href, error);
					reject(new Error('Failed to load ' + name + ' from ' + href));
				};
				document.head.appendChild(link);
			});
		}

		// Function to check if Tailwind is ready (polls until Tailwind processes classes)
		function waitForTailwind(callback, maxAttempts) {
			maxAttempts = maxAttempts || 30;
			var attempts = 0;

			function check() {
				attempts++;
				var test = document.createElement('div');
				test.className = 'bg-blue-500';
				test.style.position = 'absolute';
				test.style.visibility = 'hidden';
				test.style.top = '-9999px';
				document.body.appendChild(test);

				var style = window.getComputedStyle(test);
				var hasColor = style.backgroundColor &&
					style.backgroundColor !== 'rgba(0, 0, 0, 0)' &&
					style.backgroundColor !== 'transparent' &&
					style.backgroundColor !== 'rgb(0, 0, 0)';

				document.body.removeChild(test);

				if (hasColor || attempts >= maxAttempts) {
					callback();
				} else {
					setTimeout(check, 100);
				}
			}
			check();
		}

		// Load dependencies in order
		var loadPromises = [];
		var failedDependencies = [];

		// Load Google Fonts if needed (load first for better performance)
		if (!dependencies.fonts) {
			loadPromises.push(
				loadStylesheet('https://fonts.googleapis.com/css2?family=Outfit:wght@400&family=Ubuntu+Mono:ital,wght@0,400;0,700;1,400;1,700&family=Anton&family=Unbounded:wght@900&display=swap', 'Google Fonts').catch(function(error) {
					failedDependencies.push('Google Fonts');
					// Don't throw - fonts are nice to have but not critical
					console.warn('MeetNearMe Embed: Google Fonts failed to load, continuing anyway');
				})
			);
		}

		// Load main CSS from static server if needed
		if (!dependencies.mainCss) {
			// Use styles.css (compiled with DaisyUI) from static server
			// Check if staticBaseUrl already includes /static, adjust path accordingly
			var cssBasePath = '/assets/styles.css';
			var cssHashedPath = '/assets/styles.82a6336e.css';

			// If staticBaseUrl already ends with /static, don't add it again
			if (staticBaseUrl.endsWith('/static')) {
				cssBasePath = '/assets/styles.css';
				cssHashedPath = '/assets/styles.82a6336e.css';
			} else if (!staticBaseUrl.includes('/static')) {
				// If staticBaseUrl doesn't include /static, add it
				cssBasePath = '/static/assets/styles.css';
				cssHashedPath = '/static/assets/styles.82a6336e.css';
			}

			var cssUrl = staticBaseUrl + cssBasePath;
			var cssHashedUrl = staticBaseUrl + cssHashedPath;

			// Try to load the hashed version first, fallback to non-hashed
			loadPromises.push(
				loadStylesheet(cssHashedUrl, 'Main CSS').catch(function(error) {
					// Try fallback to non-hashed version
					console.log('MeetNearMe Embed: Hashed CSS not found, trying fallback: ' + cssUrl);
					return loadStylesheet(cssUrl, 'Main CSS (fallback)').catch(function(fallbackError) {
						failedDependencies.push('Main CSS');
						console.warn('MeetNearMe Embed: Main CSS failed to load, widget may have limited styling');
						// Don't throw - continue without main CSS
					});
				})
			);
		}

		// Load Tailwind if needed
		if (!dependencies.tailwind) {
			loadPromises.push(
				loadScript('https://cdn.tailwindcss.com', 'Tailwind CSS').then(function() {
					return new Promise(function(resolve) {
						console.log('MeetNearMe Embed: Waiting for Tailwind CSS to be ready...');
						waitForTailwind(function() {
							console.log('MeetNearMe Embed: Tailwind CSS is ready');
							resolve();
						});
					});
				}).catch(function(error) {
					failedDependencies.push('Tailwind CSS');
					throw error;
				})
			);
		}

		// Load Alpine.js Focus plugin BEFORE Alpine.js (if both need to be loaded)
		// The Focus plugin must be loaded before Alpine.js initializes to work properly
		// If Alpine is already loaded, the plugin will auto-register when the script loads
		if (!dependencies.focusPlugin && !dependencies.alpine) {
			// Both need to be loaded: load Focus plugin first, then Alpine.js
			loadPromises.push(
				loadScript('https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js', 'Alpine.js Focus Plugin').then(function() {
					// Focus plugin loaded, now load Alpine.js
					return loadScript('https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js', 'Alpine.js').catch(function(error) {
						failedDependencies.push('Alpine.js');
						throw error;
					});
				}).catch(function(error) {
					failedDependencies.push('Alpine.js Focus Plugin');
					// Don't throw - Focus plugin is important but we can continue without it (with warnings)
					console.warn('MeetNearMe Embed: Alpine.js Focus plugin failed to load, x-trap directives will not work');
					// Still try to load Alpine.js even if Focus plugin failed
					if (!dependencies.alpine) {
						return loadScript('https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js', 'Alpine.js').catch(function(error) {
							failedDependencies.push('Alpine.js');
							throw error;
						});
					}
				})
			);
		} else if (!dependencies.focusPlugin) {
			// Only Focus plugin needs to be loaded (Alpine is already loaded)
			loadPromises.push(
				loadScript('https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js', 'Alpine.js Focus Plugin').catch(function(error) {
					failedDependencies.push('Alpine.js Focus Plugin');
					console.warn('MeetNearMe Embed: Alpine.js Focus plugin failed to load, x-trap directives will not work');
				})
			);
		} else if (!dependencies.alpine) {
			// Only Alpine.js needs to be loaded (Focus plugin is already loaded)
			loadPromises.push(
				loadScript('https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js', 'Alpine.js').catch(function(error) {
					failedDependencies.push('Alpine.js');
					throw error;
				})
			);
		}

		// Load HTMX if needed
		if (!dependencies.htmx) {
			loadPromises.push(
				loadScript('https://unpkg.com/htmx.org@1.9.10', 'HTMX').catch(function(error) {
					failedDependencies.push('HTMX');
					throw error;
				})
			);
		}

		Promise.all(loadPromises).then(function() {
			console.log('MeetNearMe Embed: All dependencies loaded successfully');
			if (failedDependencies.length > 0) {
				console.warn('MeetNearMe Embed: Some non-critical dependencies failed to load: ' + failedDependencies.join(', '));
			}
			var embedUrl = baseUrl + '/api/html/embed?userId=' + encodeURIComponent(userId);
			console.log('MeetNearMe Embed: Fetching widget HTML from: ' + embedUrl);

			fetch(embedUrl, {
				method: 'GET',
				headers: {
					'Accept': 'text/html'
				},
				credentials: 'omit'
			})
			.then(function(response) {
				if (!response.ok) {
					throw new Error('Failed to load widget: HTTP ' + response.status + ' ' + response.statusText);
				}
				return response.text();
			})
			.then(function(html) {
				console.log('MeetNearMe Embed: Received HTML, length: ' + html.length);

				// Step 6: HTML Injection & Script Execution
				// CRITICAL: innerHTML does NOT execute <script> tags, so we must handle them manually
				// Also, we need to inject HTML FIRST so #alpine-state exists when scripts execute

				// Parse HTML to extract scripts before injection
				var parser = new DOMParser();
				var doc = parser.parseFromString(html, 'text/html');
				var scripts = doc.querySelectorAll('script');
				console.log('MeetNearMe Embed: Found ' + scripts.length + ' script tag(s) in HTML');

				// CRITICAL: #alpine-state is a <script> tag, so we need to keep it in the DOM
				// Extract #alpine-state script separately
				var alpineStateScript = null;
				var scriptsToExecute = [];
				scripts.forEach(function(script) {
					if (script.id === 'alpine-state') {
						alpineStateScript = script;
						console.log('MeetNearMe Embed: Found #alpine-state script tag');
					} else {
						scriptsToExecute.push(script);
					}
				});

				// Remove ALL scripts from HTML before injecting (we'll add #alpine-state back and execute others separately)
				var htmlWithoutScripts = html.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '');

				// Inject HTML into container (without scripts)
				container.innerHTML = htmlWithoutScripts;
				console.log('MeetNearMe Embed: HTML injected into container');

				// CRITICAL: Add #alpine-state script back to the container so it exists in DOM
				// This script tag needs to exist for functions to access its data attributes
				if (alpineStateScript) {
					var alpineStateElement = document.createElement('script');
					alpineStateElement.id = 'alpine-state';
					// Copy all data attributes
					Array.from(alpineStateScript.attributes).forEach(function(attr) {
						if (attr.name !== 'id') { // id is already set
							alpineStateElement.setAttribute(attr.name, attr.value);
						}
					});
					// Add empty text content (the actual script content will be executed separately)
					alpineStateElement.textContent = '';
					container.appendChild(alpineStateElement);
					console.log('MeetNearMe Embed: #alpine-state script tag added back to container');
				}

				// Verify #alpine-state exists (needed for store registration)
				var alpineState = document.querySelector('#alpine-state');
				if (!alpineState) {
					console.error('MeetNearMe Embed: #alpine-state not found after adding it back!');
				} else {
					console.log('MeetNearMe Embed: #alpine-state found');
				}

				// DIAGNOSTIC: Check for elements with x-data="getLocationSearchState()"
				var locationSearchElements = container.querySelectorAll('[x-data*="getLocationSearchState"]');
				console.log('MeetNearMe Embed: Found ' + locationSearchElements.length + ' element(s) with x-data containing getLocationSearchState');
				if (locationSearchElements.length > 0) {
					locationSearchElements.forEach(function(el, idx) {
						console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' x-data attribute:', el.getAttribute('x-data'));
						console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' in container:', container.contains(el));
					});
				} else {
					console.warn('MeetNearMe Embed: No elements found with x-data="getLocationSearchState()" in container!');
					// Check if any x-data exists at all
					var allXDataElements = container.querySelectorAll('[x-data]');
					console.log('MeetNearMe Embed: Total elements with x-data in container:', allXDataElements.length);
					if (allXDataElements.length > 0) {
						allXDataElements.forEach(function(el, idx) {
							console.log('MeetNearMe Embed: x-data element ' + (idx + 1) + ':', el.getAttribute('x-data'));
						});
					}
				}

				// Execute scripts manually (innerHTML doesn't execute them)
				// Execute them sequentially and verify functions are available
				// Note: We execute alpineStateScript first (if it exists), then other scripts
				var executeScripts = function(scriptIndex) {
					// First execute #alpine-state script if it exists
					if (scriptIndex === 0 && alpineStateScript) {
						console.log('MeetNearMe Embed: Executing #alpine-state script first');
						var alpineStateScriptElement = document.createElement('script');
						alpineStateScriptElement.id = 'alpine-state-exec';
						Array.from(alpineStateScript.attributes).forEach(function(attr) {
							alpineStateScriptElement.setAttribute(attr.name, attr.value);
						});
						alpineStateScriptElement.textContent = alpineStateScript.textContent;
						document.head.appendChild(alpineStateScriptElement);
						// Wait a bit for it to execute, then continue
						setTimeout(function() {
							executeScripts(1);
						}, 10);
						return;
					}

					// Adjust index for other scripts (skip alpineStateScript in scriptsToExecute)
					var actualIndex = alpineStateScript ? scriptIndex - 1 : scriptIndex;
					if (actualIndex >= scriptsToExecute.length) {
						console.log('MeetNearMe Embed: All scripts executed');
						// Wait a bit more to ensure scripts are fully processed
						setTimeout(function() {
							// Verify all required functions are available in global scope
							var requiredFunctions = ['getHomeState', 'getFilterFormState', 'getLocationSearchState'];
							var allAvailable = requiredFunctions.every(function(funcName) {
								return typeof window[funcName] === 'function';
							});

							if (allAvailable) {
								console.log('MeetNearMe Embed: All required functions verified in global scope after script execution');
								// Note: Functions may access Alpine stores, but stores aren't registered yet
								// The functions have defensive checks to handle missing stores during initialization
								// Proceed to store registration
								setTimeout(registerStores, 50);
							} else {
								console.warn('MeetNearMe Embed: Some functions not yet available in global scope, waiting...');
								requiredFunctions.forEach(function(funcName) {
									console.log('MeetNearMe Embed: ' + funcName + ':', typeof window[funcName]);
								});
								setTimeout(function() {
									allAvailable = requiredFunctions.every(function(funcName) {
										return typeof window[funcName] === 'function';
									});
									if (allAvailable) {
										console.log('MeetNearMe Embed: All functions now available in global scope');
										setTimeout(registerStores, 50);
									} else {
										console.error('MeetNearMe Embed: Some functions still not available after wait');
										requiredFunctions.forEach(function(funcName) {
											console.log('MeetNearMe Embed: ' + funcName + ':', typeof window[funcName]);
										});
										// Try to execute scripts again using eval to ensure they're in global scope
										console.log('MeetNearMe Embed: Attempting to re-execute scripts using eval for global scope...');
										scripts.forEach(function(script) {
											if (script.textContent && !script.hasAttribute('src')) {
												try {
													eval(script.textContent);
												} catch (e) {
													console.error('MeetNearMe Embed: Error re-executing script:', e);
												}
											}
										});
										// Check again after re-execution
										setTimeout(function() {
											allAvailable = requiredFunctions.every(function(funcName) {
												return typeof window[funcName] === 'function';
											});
											if (allAvailable) {
												console.log('MeetNearMe Embed: All functions available after re-execution');
												setTimeout(registerStores, 50);
											} else {
												console.error('MeetNearMe Embed: Some functions still not available, proceeding anyway');
												requiredFunctions.forEach(function(funcName) {
													console.log('MeetNearMe Embed: ' + funcName + ':', typeof window[funcName]);
												});
												setTimeout(registerStores, 50);
											}
										}, 50);
									}
								}, 100);
							}
						}, 50);
						return;
					}

					var script = scriptsToExecute[actualIndex];
					console.log('MeetNearMe Embed: Executing script ' + (actualIndex + 1) + ' of ' + scriptsToExecute.length);
					var newScript = document.createElement('script');

					// Copy all attributes from original script
					Array.from(script.attributes).forEach(function(attr) {
						newScript.setAttribute(attr.name, attr.value);
					});

					// Copy script content
					if (script.textContent) {
						newScript.textContent = script.textContent;
					}

					// For inline scripts (no src), execute immediately
					if (!script.hasAttribute('src')) {
						// Append to document.head to execute in global scope
						document.head.appendChild(newScript);
						// Inline scripts execute synchronously when appended
						// Use setTimeout to ensure execution completes before next script
						setTimeout(function() {
							executeScripts(scriptIndex + 1);
						}, 0);
					} else {
						// For external scripts, wait for onload
						newScript.onload = function() {
							executeScripts(scriptIndex + 1);
						};
						document.head.appendChild(newScript);
					}
				};

				// CRITICAL: Prevent Alpine from auto-processing the container until we're ready
				// Temporarily remove x-data attributes to prevent Alpine from processing
				var xDataElements = container.querySelectorAll('[x-data]');
				var xDataBackup = [];
				xDataElements.forEach(function(el) {
					var xDataValue = el.getAttribute('x-data');
					if (xDataValue) {
						xDataBackup.push({ element: el, value: xDataValue });
						el.removeAttribute('x-data');
						el.setAttribute('data-x-data-backup', xDataValue);
					}
				});
				console.log('MeetNearMe Embed: Temporarily removed x-data from ' + xDataBackup.length + ' element(s) to prevent auto-processing');

				// Start executing scripts
				executeScripts(0);

				// Step 7: Alpine Store Registration
				// CRITICAL: Stores must be registered BEFORE Alpine processes HTML
				// Also need to ensure functions like getHomeState() are available
				// Wait for scripts to finish executing, then trigger alpine:init
				function registerStores() {
					if (!window.Alpine) {
						console.error('MeetNearMe Embed: Alpine.js not available for store registration');
						return;
					}

					// Verify that all required functions are available in global scope
					var requiredFunctions = ['getHomeState', 'getFilterFormState', 'getLocationSearchState'];
					var allAvailable = requiredFunctions.every(function(funcName) {
						return typeof window[funcName] === 'function';
					});

					if (!allAvailable) {
						var missing = requiredFunctions.filter(function(funcName) {
							return typeof window[funcName] !== 'function';
						});
						var missingStr = missing.length > 0 ? missing.join(', ') : 'unknown';
						console.warn('MeetNearMe Embed: Some functions not yet available in global scope:', missingStr);
						// Wait a bit more for scripts to fully execute
						setTimeout(registerStores, 50);
						return;
					}
					console.log('MeetNearMe Embed: All required functions (getHomeState, getFilterFormState, getLocationSearchState) are available in global scope');

					console.log('MeetNearMe Embed: Triggering alpine:init event to register stores');
					document.dispatchEvent(new CustomEvent('alpine:init', { bubbles: true }));

					// Step 8: Alpine Initialization function (define before use)
					function initializeAlpine() {
						if (!window.Alpine) {
							console.error('MeetNearMe Embed: Alpine.js not available for initialization');
							return;
						}

						// Verify stores are accessible before initializing
						var urlState = window.Alpine.store('urlState');
						if (!urlState) {
							console.error('MeetNearMe Embed: urlState store not found before Alpine initialization!');
							return;
						}

						// CRITICAL: Restore x-data attributes before Alpine processes
						var xDataElements = container.querySelectorAll('[data-x-data-backup]');
						xDataElements.forEach(function(el) {
							var xDataValue = el.getAttribute('data-x-data-backup');
							if (xDataValue) {
								el.setAttribute('x-data', xDataValue);
								el.removeAttribute('data-x-data-backup');
							}
						});
						console.log('MeetNearMe Embed: Restored x-data attributes to ' + xDataElements.length + ' element(s)');

						// DIAGNOSTIC: Before Alpine initialization, check function availability
						console.log('MeetNearMe Embed: === PRE-ALPINE INITIALIZATION DIAGNOSTICS ===');
						console.log('MeetNearMe Embed: window.getLocationSearchState type:', typeof window.getLocationSearchState);
						console.log('MeetNearMe Embed: window.getLocationSearchState === getLocationSearchState:', window.getLocationSearchState === getLocationSearchState);
						console.log('MeetNearMe Embed: getLocationSearchState in global scope:', typeof getLocationSearchState);

						// Check if function can be called
						try {
							var testCall = window.getLocationSearchState();
							console.log('MeetNearMe Embed: Successfully called window.getLocationSearchState(), returned:', typeof testCall, testCall);
						} catch (e) {
							console.error('MeetNearMe Embed: Error calling window.getLocationSearchState():', e);
						}

						// Check for elements with x-data before Alpine processes
						var locationSearchElements = container.querySelectorAll('[x-data*="getLocationSearchState"]');
						console.log('MeetNearMe Embed: Elements with x-data="getLocationSearchState()" before Alpine init:', locationSearchElements.length);
						if (locationSearchElements.length > 0) {
							locationSearchElements.forEach(function(el, idx) {
								console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' exists in DOM:', el.isConnected);
								console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' parent:', el.parentElement ? el.parentElement.tagName : 'none');
							});
						}

						// Check if Alpine is already initialized on host page
						var isAlreadyInit = window.Alpine._initialized ||
							(window.Alpine.version && document.body.querySelector('[x-data]'));

						console.log('MeetNearMe Embed: Alpine already initialized on host page:', isAlreadyInit);

						if (isAlreadyInit) {
							// Alpine already running - use initTree to process only new HTML
							console.log('MeetNearMe Embed: Alpine already initialized, using Alpine.initTree() to process widget');
							if (typeof window.Alpine.initTree === 'function') {
								console.log('MeetNearMe Embed: Calling Alpine.initTree(container)...');

								// Before calling initTree, verify the function will be accessible
								var testElement = container.querySelector('[x-data*="getLocationSearchState"]');
								if (testElement) {
									console.log('MeetNearMe Embed: Found element with x-data="getLocationSearchState()"');
									console.log('MeetNearMe Embed: Testing if function is accessible from element context...');
									try {
										// Try to evaluate the expression in the element's context
										var xDataValue = testElement.getAttribute('x-data');
										console.log('MeetNearMe Embed: x-data value:', xDataValue);
										// Check if we can access the function
										var func = new Function('return ' + xDataValue);
										var result = func.call(window);
										console.log('MeetNearMe Embed: Function evaluation result:', result);
										console.log('MeetNearMe Embed: Result has isLoading:', 'isLoading' in result);
									} catch (e) {
										console.error('MeetNearMe Embed: Error evaluating function:', e);
									}
								}

								// Before calling initTree, verify the element that will be processed
								var locationElement = container.querySelector('[x-data*="getLocationSearchState"]');
								if (locationElement) {
									console.log('MeetNearMe Embed: About to process element with x-data="getLocationSearchState()"');
									console.log('MeetNearMe Embed: Element before initTree:', locationElement);

									// Check if Alpine has already processed this element
									if (locationElement._x_dataStack) {
										console.log('MeetNearMe Embed: Element already has Alpine data stack!', locationElement._x_dataStack);
									}
								}

								// CRITICAL: Alpine.initTree() might not be processing child elements correctly
								// The issue is that child elements need to inherit the parent's scope
								// In Alpine.js, child elements access parent scope through Alpine's internal scope resolution
								// The problem: When initTree() is called, child elements might be evaluated before
								// the parent's component data is fully set up, or the scope chain isn't established

								// Solution: The issue is that Alpine.initTree() might not be setting up scope inheritance
								// correctly. In Alpine.js, when a child element evaluates an expression, it should
								// look up the parent chain to find the nearest element with x-data and use its scope.
								//
								// The problem: initTree() might be processing elements in a way that doesn't
								// establish the scope chain correctly, or child elements are being evaluated
								// before the parent's component data is available.
								//
								// Let's try processing the root element with x-data first, then the container
								var rootElementWithXData = container.querySelector('[x-data*="getLocationSearchState"]');
								if (rootElementWithXData) {
									// Process the root element first to ensure its component data is set up
									// This should establish the scope that child elements can inherit from
									window.Alpine.initTree(rootElementWithXData);
									console.log('MeetNearMe Embed: Processed root element with x-data="getLocationSearchState()"');
								}

								// Also process the container to catch any other elements
								window.Alpine.initTree(container);

								console.log('MeetNearMe Embed: Alpine tree initialized successfully');
								console.log('MeetNearMe Embed: [NEW CODE v2] About to start time filter button diagnostics in 150ms...');
								console.log('MeetNearMe Embed: [DEBUG] Setting up setTimeout for button diagnostics...');

								// DIAGNOSTIC: Check if time filter buttons are processed by Alpine
								setTimeout(function() {
									console.log('MeetNearMe Embed: [DEBUG] setTimeout callback executing now...');
									console.log('MeetNearMe Embed: ============================================');
									console.log('MeetNearMe Embed: === TIME FILTER BUTTONS DIAGNOSTICS v2 ===');
									console.log('MeetNearMe Embed: ============================================');
									console.log('MeetNearMe Embed: [NEW CODE] This is the updated diagnostic code');

									// Find buttons with @click handlers for time filters
									var timeFilterButtons = container.querySelectorAll('button[type="button"]');
									console.log('MeetNearMe Embed: Found ' + timeFilterButtons.length + ' button(s) in container');

									var buttonsWithClick = [];
									timeFilterButtons.forEach(function(btn, idx) {
										var clickAttr = btn.getAttribute('@click') || btn.getAttribute('x-on:click');
										var text = btn.textContent.trim();

										// Check if this is a time filter button (TODAY, TOMORROW, etc.)
										if (text === 'TODAY' || text === 'TOMORROW' || text === 'THIS WEEK' || text === 'THIS MONTH') {
											try {
												console.log('MeetNearMe Embed: Found time filter button: "' + text + '"');
												console.log('MeetNearMe Embed:   - Button element:', btn);
												console.log('MeetNearMe Embed:   - Button textContent:', btn.textContent);

												// Check for click attribute (Alpine uses x-on:click, not @click in the DOM)
												var clickAttr = btn.getAttribute('x-on:click') || btn.getAttribute('@click');
												console.log('MeetNearMe Embed:   - Has x-on:click attribute:', !!btn.getAttribute('x-on:click'));
												console.log('MeetNearMe Embed:   - Has @click attribute:', !!btn.getAttribute('@click'));
												if (clickAttr) {
													console.log('MeetNearMe Embed:   - Click attribute value:', clickAttr);
												} else {
													console.warn('MeetNearMe Embed:   - WARNING: No click attribute found on button!');
												}

												// Check if Alpine has processed this button
												var hasAlpineData = !!(btn._x_dataStack || btn.__x);
												console.log('MeetNearMe Embed:   - Has Alpine data stack (_x_dataStack):', !!btn._x_dataStack);
												console.log('MeetNearMe Embed:   - Has Alpine data (__x):', !!btn.__x);
												console.log('MeetNearMe Embed:   - Has Alpine data (either):', hasAlpineData);

												// Check if button has Alpine event listeners
												var hasAlpineListeners = !!(btn._x_attributeCleanups || btn.__x_attributeCleanups);
												console.log('MeetNearMe Embed:   - Has Alpine listeners (_x_attributeCleanups):', !!btn._x_attributeCleanups);
												console.log('MeetNearMe Embed:   - Has Alpine listeners (__x_attributeCleanups):', !!btn.__x_attributeCleanups);
												console.log('MeetNearMe Embed:   - Has Alpine listeners (either):', hasAlpineListeners);

												// Check if button is in an Alpine context (has parent with x-data)
												var parentWithXData = btn.closest('[x-data]');
												console.log('MeetNearMe Embed:   - Has parent with x-data:', !!parentWithXData);
												if (parentWithXData) {
													console.log('MeetNearMe Embed:   - Parent x-data:', parentWithXData.getAttribute('x-data'));
												} else {
													console.warn('MeetNearMe Embed:   - WARNING: Button is not inside an element with x-data!');
												}

												// Check if $store is accessible (test by trying to read it)
												try {
													// We can't directly test $store without Alpine context, but we can check if Alpine.store exists
													var storeExists = window.Alpine && window.Alpine.store && typeof window.Alpine.store('urlState') !== 'undefined';
													console.log('MeetNearMe Embed:   - urlState store exists:', storeExists);
													if (storeExists) {
														var urlState = window.Alpine.store('urlState');
														console.log('MeetNearMe Embed:   - urlState store object:', urlState);
													}
												} catch (e) {
													console.error('MeetNearMe Embed:   - Error checking store:', e);
												}

												buttonsWithClick.push({
													button: btn,
													text: text,
													hasClickAttr: !!clickAttr,
													hasAlpineData: hasAlpineData,
													hasAlpineListeners: hasAlpineListeners
												});
											} catch (e) {
												console.error('MeetNearMe Embed:   - ERROR processing button "' + text + '":', e);
												console.error('MeetNearMe Embed:   - Error stack:', e.stack);
											}
										}
									});


									console.log('MeetNearMe Embed: [CRITICAL] After forEach loop, buttonsWithClick.length:', buttonsWithClick.length);
									console.log('MeetNearMe Embed: Summary - Found ' + buttonsWithClick.length + ' time filter button(s) **');
									console.log('MeetNearMe Embed: [CRITICAL] About to check buttonsWithClick.length === 0');

									if (buttonsWithClick.length === 0) {
										console.warn('MeetNearMe Embed: WARNING - No time filter buttons found!');
									} else {
										console.log('MeetNearMe Embed: [DEBUG] Entering else block, buttonsWithClick.length:', buttonsWithClick.length);
										try {
											var processedCount = buttonsWithClick.filter(function(b) { return b.hasAlpineData || b.hasAlpineListeners; }).length;
											console.log('MeetNearMe Embed: [DEBUG] processedCount calculated:', processedCount);
											console.log('MeetNearMe Embed: ' + processedCount + ' of ' + buttonsWithClick.length + ' button(s) appear to be processed by Alpine');
											console.log('MeetNearMe Embed: [DEBUG] About to check if buttonsWithClick.length > 0, length is:', buttonsWithClick.length);
										} catch (e) {
											console.error('MeetNearMe Embed: [ERROR] Error calculating processedCount:', e);
											console.error('MeetNearMe Embed: [ERROR] Error stack:', e.stack);
										}

										// DIAGNOSTIC: Add manual click listener to ALL buttons to test if clicks are received
										try {
											if (buttonsWithClick.length > 0) {
												console.log('MeetNearMe Embed: [DEBUG] Entering buttonsWithClick.length > 0 block');
												console.log('MeetNearMe Embed: === CLICK HANDLER ANALYSIS ===');
												console.log('MeetNearMe Embed: Adding manual click listeners to all ' + buttonsWithClick.length + ' time filter button(s)');

												// Add manual click listener to each button
												buttonsWithClick.forEach(function(buttonInfo, idx) {
													var btn = buttonInfo.button;
													var btnText = buttonInfo.text;
													var clickAttr = btn.getAttribute('@click') || btn.getAttribute('x-on:click');

													console.log('MeetNearMe Embed: Adding listener to button ' + (idx + 1) + ': "' + btnText + '"');

													// Add a manual click listener to see if clicks are being received
													btn.addEventListener('click', function(e) {
														console.log('MeetNearMe Embed: [MANUAL LISTENER] Button "' + btnText + '" clicked!');
														console.log('MeetNearMe Embed: [MANUAL LISTENER] Event:', e);
														console.log('MeetNearMe Embed: [MANUAL LISTENER] Alpine listeners exist:', !!btn._x_attributeCleanups);

														// Check if Alpine's click handler would have access to $store
														if (window.Alpine && window.Alpine.store) {
															var urlState = window.Alpine.store('urlState');
															console.log('MeetNearMe Embed: [MANUAL LISTENER] urlState accessible:', !!urlState);

															// Try to manually evaluate the expression
															try {
																// Get parent scope
																var parentWithXData = btn.closest('[x-data]');
																if (parentWithXData && parentWithXData._x_dataStack && parentWithXData._x_dataStack.length > 0) {
																	var parentData = parentWithXData._x_dataStack[0];
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] Parent data available:', !!parentData);

																	// Try to manually execute the expression
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] Expression would be:', clickAttr);
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] Attempting to manually call setParam...');

																	// Check what value Alpine would pass to setParam
																	// The expression is: $store.urlState.setParam('start_time', 'this_month')
																	// But Alpine might not be evaluating it correctly
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] Checking current store state...');
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] urlState.start_time:', urlState.start_time);
																	console.log('MeetNearMe Embed: [MANUAL LISTENER] urlState object keys:', Object.keys(urlState));

																	// Check the hidden form inputs to see what values they have
																	var form = document.getElementById('event-search-form');
																	if (form) {
																		var startTimeInput = form.querySelector('#start_time');
																		var endTimeInput = form.querySelector('#end_time');
																		var qInput = form.querySelector('#q');
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] Form found:', !!form);
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] start_time input value:', startTimeInput ? startTimeInput.value : 'not found');
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] end_time input value:', endTimeInput ? endTimeInput.value : 'not found');
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] q input value:', qInput ? qInput.value : 'not found');
																	} else {
																		console.warn('MeetNearMe Embed: [MANUAL LISTENER] Form not found!');
																	}

																	// Extract the value from the click attribute
																	// The expression is: $store.urlState.setParam('start_time', 'this_month')
																	// We need to extract 'this_month' from the string
																	var match = clickAttr.match(/setParam\(['"]start_time['"],\s*['"]([^'"]+)['"]\)/);
																	if (match && match[1]) {
																		var extractedValue = match[1];
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] Extracted value from expression:', extractedValue);

																		// Check store state BEFORE calling setParam
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] Store start_time BEFORE setParam:', urlState.start_time);

																		// Call setParam
																		console.log('MeetNearMe Embed: [MANUAL LISTENER] Calling setParam with extracted value...');
																		try {
																			urlState.setParam('start_time', extractedValue);
																			console.log('MeetNearMe Embed: [MANUAL LISTENER] setParam call succeeded with value:', extractedValue);

																			// Check store state AFTER calling setParam
																			console.log('MeetNearMe Embed: [MANUAL LISTENER] Store start_time AFTER setParam:', urlState.start_time);

																			// Check hidden input value AFTER setParam
																			var form = document.getElementById('event-search-form');
																			if (form) {
																				var startTimeInput = form.querySelector('#start_time');
																				console.log('MeetNearMe Embed: [MANUAL LISTENER] Hidden input value AFTER setParam:', startTimeInput ? startTimeInput.value : 'not found');

																				// If the input is still empty, manually set it
																				if (startTimeInput && !startTimeInput.value && urlState.start_time) {
																					console.log('MeetNearMe Embed: [MANUAL LISTENER] Input is empty but store has value - manually setting input value');
																					startTimeInput.value = urlState.start_time;
																					console.log('MeetNearMe Embed: [MANUAL LISTENER] Manually set input value to:', startTimeInput.value);
																				}
																			}
																		} catch (setParamError) {
																			console.error('MeetNearMe Embed: [MANUAL LISTENER] setParam call failed:', setParamError);
																		}
																	} else {
																		console.warn('MeetNearMe Embed: [MANUAL LISTENER] Could not extract value from expression:', clickAttr);
																	}
																}
															} catch (err) {
																console.error('MeetNearMe Embed: [MANUAL LISTENER] Error:', err);
															}
														}
													}, true); // Use capture phase to see if it fires before Alpine's handler

													console.log('MeetNearMe Embed: Added manual click listener to button "' + btnText + '"');

													// DIAGNOSTIC: Check store state and form inputs BEFORE any clicks (only once, not per button)
													if (idx === 0) {
														console.log('MeetNearMe Embed: === PRE-CLICK STATE CHECK ===');
														if (window.Alpine && window.Alpine.store) {
															var urlState = window.Alpine.store('urlState');
															console.log('MeetNearMe Embed: [PRE-CLICK] urlState.start_time:', urlState.start_time);
															console.log('MeetNearMe Embed: [PRE-CLICK] urlState object:', urlState);

															// Check hidden form inputs
															var form = document.getElementById('event-search-form');
															if (form) {
																var startTimeInput = form.querySelector('#start_time');
																var endTimeInput = form.querySelector('#end_time');
																var qInput = form.querySelector('#q');
																console.log('MeetNearMe Embed: [PRE-CLICK] Form found');
																console.log('MeetNearMe Embed: [PRE-CLICK] start_time input value:', startTimeInput ? startTimeInput.value : 'not found');
																console.log('MeetNearMe Embed: [PRE-CLICK] start_time input :value binding:', startTimeInput ? startTimeInput.getAttribute(':value') : 'not found');
																console.log('MeetNearMe Embed: [PRE-CLICK] end_time input value:', endTimeInput ? endTimeInput.value : 'not found');
																console.log('MeetNearMe Embed: [PRE-CLICK] q input value:', qInput ? qInput.value : 'not found');
															} else {
																console.warn('MeetNearMe Embed: [PRE-CLICK] Form not found!');
															}
														}
														console.log('MeetNearMe Embed: === END PRE-CLICK CHECK ===');
														console.log('MeetNearMe Embed: Now you can click any time filter button - logs will appear above');
													}
												});
										} else {
											console.log('MeetNearMe Embed: [DEBUG] buttonsWithClick.length is 0, skipping click listener setup');
										}
										} catch (e) {
											console.error('MeetNearMe Embed: [ERROR] Error in click handler analysis:', e);
											console.error('MeetNearMe Embed: [ERROR] Error stack:', e.stack);
										}

										// MANUAL EVENT LISTENER: Search form submit handler
										// This is a workaround for Alpine scope inheritance issues
										// The form's @submit.prevent handler uses $store which isn't accessible in embed context
										try {
											var searchInput = container.querySelector('#search-input');
											var searchForm = searchInput ? searchInput.closest('form') : null;

											if (searchForm && searchInput) {
												console.log('MeetNearMe Embed: Found search form, adding manual submit listener');
												console.log('MeetNearMe Embed: [SEARCH FORM] Form element:', searchForm);
												console.log('MeetNearMe Embed: [SEARCH FORM] Form has @submit.prevent:', searchForm.hasAttribute('@submit') || searchForm.getAttribute('x-on:submit'));

												// Add listener in capture phase (before Alpine's handler)
												searchForm.addEventListener('submit', function(e) {
													console.log('MeetNearMe Embed: [SEARCH FORM] ===== FORM SUBMIT EVENT FIRED =====');
													console.log('MeetNearMe Embed: [SEARCH FORM] Event type:', e.type);
													console.log('MeetNearMe Embed: [SEARCH FORM] Event defaultPrevented:', e.defaultPrevented);
													console.log('MeetNearMe Embed: [SEARCH FORM] Event target:', e.target);
													console.log('MeetNearMe Embed: [SEARCH FORM] Event currentTarget:', e.currentTarget);

													// Always prevent default to stop normal form submission
													e.preventDefault();
													e.stopPropagation();
													e.stopImmediatePropagation();

													console.log('MeetNearMe Embed: [SEARCH FORM] Prevented default form submission');

													var searchValue = searchInput.value;
													console.log('MeetNearMe Embed: [SEARCH FORM] Search input value:', searchValue);

													if (window.Alpine && window.Alpine.store) {
														var urlState = window.Alpine.store('urlState');
														if (urlState && urlState.setParam) {
															console.log('MeetNearMe Embed: [SEARCH FORM] Calling setParam with q:', searchValue);
															console.log('MeetNearMe Embed: [SEARCH FORM] Store q BEFORE setParam:', urlState.q);
															try {
																urlState.setParam('q', searchValue);
																console.log('MeetNearMe Embed: [SEARCH FORM] setParam call succeeded');
																console.log('MeetNearMe Embed: [SEARCH FORM] Store q AFTER setParam:', urlState.q);
																console.log('MeetNearMe Embed: [SEARCH FORM] Current URL:', window.location.href);
															} catch (setParamError) {
																console.error('MeetNearMe Embed: [SEARCH FORM] setParam call failed:', setParamError);
															}
														} else {
															console.error('MeetNearMe Embed: [SEARCH FORM] urlState or setParam not available');
														}
													} else {
														console.error('MeetNearMe Embed: [SEARCH FORM] Alpine or store not available');
													}

													return false;
												}, true); // Use capture phase to intercept before Alpine's handler

												// Also check if form has action attribute (which would cause navigation)
												if (searchForm.hasAttribute('action')) {
													console.warn('MeetNearMe Embed: [SEARCH FORM] WARNING: Form has action attribute:', searchForm.getAttribute('action'));
												} else {
													console.log('MeetNearMe Embed: [SEARCH FORM] Form has no action attribute (good)');
												}

												console.log('MeetNearMe Embed: Added manual submit listener to search form');
											} else {
												console.warn('MeetNearMe Embed: Search form or input not found');
											}
										} catch (searchFormError) {
											console.error('MeetNearMe Embed: [ERROR] Error setting up search form listener:', searchFormError);
										}
									}
								}, 150);

								// After initTree, check if component data was set correctly
								if (locationElement) {
									setTimeout(function() {
											console.log('MeetNearMe Embed: === POST-INITTREE ELEMENT CHECK ===');
											console.log('MeetNearMe Embed: Element after initTree:', locationElement);
											if (locationElement._x_dataStack && locationElement._x_dataStack.length > 0) {
												var componentData = locationElement._x_dataStack[0];
												console.log('MeetNearMe Embed: Component data object:', componentData);
												console.log('MeetNearMe Embed: Component has isLoading:', 'isLoading' in componentData);
												console.log('MeetNearMe Embed: Component has isOpen:', 'isOpen' in componentData);
												console.log('MeetNearMe Embed: Component has options:', 'options' in componentData);
												console.log('MeetNearMe Embed: isLoading value:', componentData.isLoading);
												console.log('MeetNearMe Embed: isOpen value:', componentData.isOpen);

												// Check child elements
												var childWithIsLoading = locationElement.querySelector('[x-bind\\:disabled*="isLoading"]');
												if (childWithIsLoading) {
													console.log('MeetNearMe Embed: Found child element trying to use isLoading:', childWithIsLoading);
													console.log('MeetNearMe Embed: Child element parent:', childWithIsLoading.parentElement);
													console.log('MeetNearMe Embed: Child element has Alpine data:', childWithIsLoading._x_dataStack ? 'yes' : 'no');

													// Check if parent has data stack
													var parent = childWithIsLoading.parentElement;
													while (parent && parent !== locationElement) {
														if (parent._x_dataStack) {
															console.log('MeetNearMe Embed: Found parent with data stack:', parent);
															break;
														}
														parent = parent.parentElement;
													}
													if (parent === locationElement && locationElement._x_dataStack) {
														console.log('MeetNearMe Embed: Child should inherit from locationElement which has data stack');
													}
												}
								} else {
									console.error('MeetNearMe Embed: Element does not have Alpine data stack after initTree!');
								}
							}, 100);
						}

						// DIAGNOSTIC: After initTree, check if elements were processed
								setTimeout(function() {
									console.log('MeetNearMe Embed: === POST-ALPINE INITIALIZATION DIAGNOSTICS ===');
									var processedElements = container.querySelectorAll('[x-data*="getLocationSearchState"]');
									console.log('MeetNearMe Embed: Elements with x-data after Alpine.initTree():', processedElements.length);
									// Check if Alpine has processed these elements
									if (processedElements.length > 0) {
										processedElements.forEach(function(el, idx) {
											var hasAlpineData = el._x_dataStack || el.__x;
											console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' has Alpine data:', !!hasAlpineData);
										});
									}
								}, 100);
							} else {
								console.warn('MeetNearMe Embed: Alpine.initTree() not available, Alpine should auto-process new HTML');
							}
						} else {
							// Alpine not initialized - start it
							console.log('MeetNearMe Embed: Starting Alpine.js');
							if (typeof window.Alpine.start === 'function') {
								console.log('MeetNearMe Embed: Calling Alpine.start()...');
								window.Alpine.start();
								console.log('MeetNearMe Embed: Alpine.js started successfully');

								// DIAGNOSTIC: After start, check if elements were processed
								setTimeout(function() {
									console.log('MeetNearMe Embed: === POST-ALPINE INITIALIZATION DIAGNOSTICS ===');
									var processedElements = container.querySelectorAll('[x-data*="getLocationSearchState"]');
									console.log('MeetNearMe Embed: Elements with x-data after Alpine.start():', processedElements.length);
									if (processedElements.length > 0) {
										processedElements.forEach(function(el, idx) {
											var hasAlpineData = el._x_dataStack || el.__x;
											console.log('MeetNearMe Embed: Element ' + (idx + 1) + ' has Alpine data:', !!hasAlpineData);
										});
									}
								}, 100);
							} else {
								console.warn('MeetNearMe Embed: Alpine.start() not available');
							}
						}

						// TODO: Proceed to Step 9 (Form Submission Handling) and Step 10 (HTMX Initialization) in next implementation
					}

					// Wait for stores to register, then verify
					setTimeout(function() {
						var urlState = window.Alpine && window.Alpine.store ? window.Alpine.store('urlState') : null;
						var filters = window.Alpine && window.Alpine.store ? window.Alpine.store('filters') : null;
						var location = window.Alpine && window.Alpine.store ? window.Alpine.store('location') : null;

						console.log('MeetNearMe Embed: Store check - urlState:', !!urlState, 'filters:', !!filters, 'location:', !!location);

						if (!urlState || !filters || !location) {
							console.warn('MeetNearMe Embed: Some stores not registered, retrying alpine:init...');
							// Retry alpine:init event
							document.dispatchEvent(new CustomEvent('alpine:init', { bubbles: true }));

							// Wait again and verify
							setTimeout(function() {
								urlState = window.Alpine && window.Alpine.store ? window.Alpine.store('urlState') : null;
								filters = window.Alpine && window.Alpine.store ? window.Alpine.store('filters') : null;
								location = window.Alpine && window.Alpine.store ? window.Alpine.store('location') : null;

								console.log('MeetNearMe Embed: After retry - urlState:', !!urlState, 'filters:', !!filters, 'location:', !!location);

								if (!urlState || !filters || !location) {
									console.error('MeetNearMe Embed: Stores still not registered after retry');
									// Proceed to Step 8 anyway - Alpine might still work
									initializeAlpine();
								} else {
									console.log('MeetNearMe Embed: All stores registered successfully');
									initializeAlpine();
								}
							}, 100);
						} else {
							console.log('MeetNearMe Embed: All stores registered successfully');
							initializeAlpine();
						}
					}, 100);
				}
			})
			.catch(function(error) {
				console.error('MeetNearMe Embed: Error fetching widget HTML', error);
				var errorMsg = '<div style="padding: 1rem; background-color: #fee; border: 1px solid #fcc; border-radius: 0.5rem; color: #c33;">MeetNearMe Embed Error: Failed to load widget. ' + error.message + ' Please check the console for details.</div>';
				container.innerHTML = errorMsg;
			});
		}).catch(function(error) {
			console.error('MeetNearMe Embed: Error loading dependencies', error);
			if (failedDependencies.length > 0) {
				console.error('MeetNearMe Embed: Failed to load the following dependencies: ' + failedDependencies.join(', '));
			}
			// Show error in container
			container.innerHTML = '<div style="padding: 1rem; background-color: #fee; border: 1px solid #fcc; border-radius: 0.5rem; color: #c33;">MeetNearMe Embed Error: Failed to load required dependencies. Please check the console for details.</div>';
		});
	})();`, staticBaseUrl)

	// Return handler function that sets headers and writes the script
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for cross-origin script loading
		transport.SetCORSHeaders(w, r)

		// Set content type for JavaScript
		w.Header().Set("Content-Type", "application/javascript")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(script))
	}
}
