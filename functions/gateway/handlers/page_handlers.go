package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/services/dynamodb_service"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var US_GEO_DEFAULT_LAT = float64(helpers.Cities[0].Latitude)
var US_GEO_DEFAULT_LONG = float64(helpers.Cities[0].Longitude)

func ParseStartEndTime(startTimeStr, endTimeStr string) (_startTimeUnix, _endTimeUnix int64) {
	var startTime time.Time
	var endTime time.Time

	var startTimeUnix int64
	var endTimeUnix int64

	// NOTE: This assumes the UI home page default is "THIS MONTH" and the absence
	// of an explicit start_time query param ...
	if startTimeStr == "" && endTimeStr == "" {
		startTime = time.Now()
		// NOTE: default to 3 months
		endTime = startTime.AddDate(0, 3, 0)
	} else if strings.ToLower(startTimeStr) == "this_month" {
		startTime = time.Now()
		endTime = startTime.AddDate(0, 1, 0)
	} else if strings.ToLower(startTimeStr) == "today" {
		startTime = time.Now()
		endTime = startTime.AddDate(0, 0, 1)
		// NOTE: "tomorrow" is a time-bound concept that should eventually be timezone relative
		// to the user, this is currently simplistic and is just 24 - 48hrs from the current time
	} else if strings.ToLower(startTimeStr) == "tomorrow" {
		startTime = time.Now().AddDate(0, 0, 1)
		endTime = startTime.AddDate(0, 0, 1)
	} else if strings.ToLower(startTimeStr) == "this_week" {
		startTime = time.Now()
		endTime = startTime.AddDate(0, 0, 7)
	} else if strings.ToLower(startTimeStr) == "this_year" {
		startTime = time.Now()
		endTime = startTime.AddDate(1, 0, 0)
	}

	// return early if one of the above are found
	if !startTime.IsZero() && !endTime.IsZero() {
		return startTime.Unix(), endTime.Unix()
	}

	// convert startTime either UTC / time.RFC3339 or integer
	// string (presumed unix) to int64
	if _, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	} else if startTimeUnix, err = strconv.ParseInt(startTimeStr, 10, 64); err == nil {
		startTime = time.Unix(startTimeUnix, 0)
		// default wrong query string usage to NOW for startTime
	} else {
		startTime = time.Now()
	}

	// convert endTime either UTC / time.RFC3339 or integer
	// string (presumed unix) to int64
	if _, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	} else if endTimeUnix, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
		endTime = time.Unix(endTimeUnix, 0)
		// Set end time to 24 hours after start time
		// default wrong query string usage to PLUS ONE MONTH for endTime
	} else {
		endTime = startTime.AddDate(0, 1, 0)
	}

	startTimeUnix = startTime.Unix()
	endTimeUnix = endTime.Unix()

	return startTimeUnix, endTimeUnix
}

func GetSearchParamsFromReq(r *http.Request) (query string, userLocation []float64, maxDistance float64, startTime int64, endTime int64, cfLocation helpers.CdnLocation, ownerIds []string, categories string, address string, parseDatesBool string, eventSourceTypes []string, eventSourceIds []string) {
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	latStr := r.URL.Query().Get("lat")
	longStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")
	q := r.URL.Query().Get("q")
	owners := r.URL.Query().Get("owners")
	categoriesStr := r.URL.Query().Get("categories")
	address = r.URL.Query().Get("address")
	parseDates := r.URL.Query().Get("parse_dates")
	eventSourceTypesStr := r.URL.Query().Get("event_source_types")
	eventSourceIdsStr := r.URL.Query().Get("event_source_ids")
	cfRay := GetCfRay(r)
	rayCode := ""

	cfLocationLat := services.InitialEmptyLatLong
	cfLocationLon := services.InitialEmptyLatLong

	if len(cfRay) > 2 {
		rayCode = cfRay[len(cfRay)-3:]
		cfLocation = helpers.CfLocationMap[rayCode]
		cfLocationLat = cfLocation.Lat
		cfLocationLon = cfLocation.Lon
	}

	// default lat / lon to geographic center of US
	lat := US_GEO_DEFAULT_LAT
	long := US_GEO_DEFAULT_LONG

	// Parse parameter values if provided
	if latStr != "" {
		lat64, _ := strconv.ParseFloat(latStr, 32)
		lat = float64(lat64)
	} else if cfLocationLat != services.InitialEmptyLatLong {
		lat = float64(cfLocationLat)
	}
	if longStr != "" {
		long64, _ := strconv.ParseFloat(longStr, 32)
		long = float64(long64)
	} else if cfLocationLon != services.InitialEmptyLatLong {
		long = float64(cfLocationLon)
	}

	var radius float64

	if radiusStr != "" {
		radius64, err := strconv.ParseFloat(radiusStr, 32)
		// only set the radius if string successfully converts to a float64
		if err == nil {
			radius = float64(radius64)
		}
	}

	// we failed to get a radius string, set an implicit default, if cfLocationLat/Lon
	// is not the initial empty value (can't use 0.0, a valid lat/lon) we assume
	// cfLocation has given us a reasonable local guess

	if radius < 0.0001 && (cfLocationLat != services.InitialEmptyLatLong && cfLocationLon != services.InitialEmptyLatLong ||
		lat != US_GEO_DEFAULT_LAT && long != US_GEO_DEFAULT_LONG) {
		radius = helpers.DEFAULT_SEARCH_RADIUS
		// we still don't have lat/lon, which means we'll be using "geographic center of US"
		// which is in the middle of nowhere. Expand the radius to show all of the country
		// showing events from anywhere
	} else if radius == 0.0 {
		radius = helpers.DEFAULT_EXPANDED_SEARCH_RADIUS
	}

	startTimeUnix, endTimeUnix := ParseStartEndTime(startTimeStr, endTimeStr)

	if startTimeUnix < time.Now().Unix() && endTimeStr == "" {
		endTimeUnix = time.Now().Unix()
	}

	// Handle owners query parameter
	ownerIds = []string{}
	if owners != "" {
		ownerIds = strings.Split(owners, ",")
	}

	// Decode the URL-encoded categories string
	decodedCategories, err := url.QueryUnescape(categoriesStr)
	if err != nil {
		log.Printf("Error decoding categories: %v", err)
		decodedCategories = categories // Use the original string if decoding fails
	}

	// Split comma-separated values into slices
	if eventSourceTypesStr != "" {
		eventSourceTypes = strings.Split(eventSourceTypesStr, ",")
	}
	if eventSourceIdsStr != "" {
		eventSourceIds = strings.Split(eventSourceIdsStr, ",")
	}

	return q, []float64{lat, long}, radius, startTimeUnix, endTimeUnix, cfLocation, ownerIds, decodedCategories, address, parseDates, eventSourceTypes, eventSourceIds
}

func DeriveEventsFromRequest(r *http.Request) ([]types.Event, helpers.CdnLocation, []float64, *types.UserSearchResult, int, error) {
	// Extract parameter values from the request query parameters
	q, userLocation, radius, startTimeUnix, endTimeUnix, cfLocation, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)
	userId := mux.Vars(r)[helpers.USER_ID_KEY]

	// Setup channels for concurrent operations
	type userResult struct {
		user types.UserSearchResult
		err  error
	}
	type searchResult struct {
		res types.EventSearchResponse
		err error
	}
	type aboutResult struct {
		data string
		err  error
	}

	userChan := make(chan userResult, 1)
	searchChan := make(chan searchResult, 1)
	aboutChan := make(chan aboutResult, 1)

	var pageUser *types.UserSearchResult
	subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")
	if subdomainValue != "" {
		userId = subdomainValue
	}
	// Start concurrent operations if userId exists
	if userId != "" {
		// Single goroutine for all three requests when userId exists
		go func() {
			// Get user data - hard fail if this errors
			user, err := helpers.GetOtherUserByID(userId)
			if err != nil {
				userChan <- userResult{user, err}
				// Early return since user data is required
				searchChan <- searchResult{types.EventSearchResponse{}, err}
				aboutChan <- aboutResult{"", err} // Close about channel
				return
			}
			// user resolved successfully, push to channel
			userChan <- userResult{user, nil}

			// Get about data - soft fail if this errors
			aboutData, err := helpers.GetOtherUserMetaByID(userId, helpers.META_ABOUT_KEY)
			if err != nil {
				// Check if it's a 4xx error (we can soft fail)
				if strings.HasPrefix(err.Error(), "4") {
					aboutChan <- aboutResult{"", nil} // Ignore 4xx errors
				} else {
					aboutChan <- aboutResult{"", err} // Propagate other errors
				}
			} else {
				aboutChan <- aboutResult{aboutData, nil}
			}

			// Get search results
			marqoClient, err := services.GetMarqoClient()
			if err != nil {
				searchChan <- searchResult{types.EventSearchResponse{}, errors.New("failed to get marqo client: " + err.Error())}
				return
			}

			if subdomainValue != "" {
				ownerIds = []string{subdomainValue}
			} else {
				ownerIds = []string{userId}
			}

			res, err := services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
			searchChan <- searchResult{res, err}
		}()
	} else {
		// Just do the search request directly when userId doesn't exist
		close(userChan)
		close(aboutChan)

		marqoClient, err := services.GetMarqoClient()
		if err != nil {
			searchChan <- searchResult{types.EventSearchResponse{}, errors.New("failed to get marqo client: " + err.Error())}
			return []types.Event{}, cfLocation, []float64{}, nil, http.StatusInternalServerError, err
		}

		subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")
		if subdomainValue != "" {
			ownerIds = []string{subdomainValue}
		}

		res, err := services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
		searchChan <- searchResult{res, err}
	}

	// fetch the `about` metadata for the user
	var aboutData string
	if userId != "" {
		// NOTE: here we ignore the error because we allow the page/user to not have an about section
		aboutData, _ = helpers.GetOtherUserMetaByID(userId, helpers.META_ABOUT_KEY)
		// Get user result from channel
		userResult := <-userChan
		if userResult.err != nil {
			return []types.Event{}, cfLocation, []float64{}, nil, http.StatusInternalServerError, userResult.err
		}
		pageUser = &userResult.user
		pageUser.UserID = userId
	}

	// Get search results from channel
	result := <-searchChan
	if result.err != nil {
		return []types.Event{}, cfLocation, []float64{}, nil, http.StatusInternalServerError, result.err
	}

	events := result.res.Events
	if pageUser != nil {
		// Initialize the metadata map if it's nil
		if aboutData != "" && pageUser.Metadata == nil {
			pageUser.Metadata = make(map[string]string)
			pageUser.Metadata[helpers.META_ABOUT_KEY] = aboutData
		}
	}

	return events, cfLocation, userLocation, pageUser, http.StatusOK, nil
}

func GetHomeOrUserPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	originalQueryLat := r.URL.Query().Get("lat")
	originalQueryLong := r.URL.Query().Get("lon")
	originalQueryLocation := r.URL.Query().Get("location")
	events, cfLocation, userLocation, pageUser, status, err := DeriveEventsFromRequest(r)
	if err != nil {
		subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")
		if subdomainValue != "" || strings.Contains(r.URL.Path, "/user") {
			return transport.SendHtmlErrorPage([]byte("User Not Found"), 200, true)
		}
		return transport.SendHtmlRes(w, []byte(err.Error()), status, "page", err)
	}

	homePage := pages.HomePage(
		events,
		pageUser,
		cfLocation,
		fmt.Sprint(userLocation[0]),
		fmt.Sprint(userLocation[1]),
		fmt.Sprint(originalQueryLat),
		fmt.Sprint(originalQueryLong),
		originalQueryLocation,
	)

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	layoutTemplate := pages.Layout(helpers.SitePages["home"], userInfo, homePage, types.Event{}, []string{"https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js"})

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetAboutPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	aboutPage := pages.AboutPage()
	ctx := r.Context()

	layoutTemplate := pages.Layout(helpers.SitePages["about"], helpers.UserInfo{}, aboutPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetProfilePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	// TODO: add a unit test that verifies each page handler works both WITH and also
	// WITHOUT a user present in context
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	roleClaims := []helpers.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = claims
	}

	userMetaClaims := map[string]interface{}{}
	if _, ok := ctx.Value("userMetaClaims").(map[string]interface{}); ok {
		userMetaClaims = ctx.Value("userMetaClaims").(map[string]interface{})
	}
	userInterests := helpers.GetUserInterestFromMap(userMetaClaims, helpers.INTERESTS_KEY)
	userSubdomain := helpers.GetBase64ValueFromMap(userMetaClaims, helpers.SUBDOMAIN_KEY)
	userAboutData, err := helpers.GetOtherUserMetaByID(userInfo.Sub, helpers.META_ABOUT_KEY)
	adminPage := pages.ProfilePage(userInfo, roleClaims, userInterests, userSubdomain, userAboutData)
	layoutTemplate := pages.Layout(helpers.SitePages["profile"], userInfo, adminPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetProfileSettingsPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	userMetaClaims := map[string]interface{}{}
	if _, ok := ctx.Value("userMetaClaims").(map[string]interface{}); ok {
		userMetaClaims = ctx.Value("userMetaClaims").(map[string]interface{})
	}
	parsedInterests := helpers.GetUserInterestFromMap(userMetaClaims, helpers.INTERESTS_KEY)
	settingsPage := pages.ProfileSettingsPage(parsedInterests)
	layoutTemplate := pages.Layout(helpers.SitePages["settings"], userInfo, settingsPage, types.Event{}, []string{})

	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetAddOrEditEventPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	roleClaims := []helpers.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = claims
	}

	validRoles := []string{"superAdmin", "eventAdmin"}
	if !helpers.HasRequiredRole(roleClaims, validRoles) {
		err := errors.New("Only event editors can add or edit events")
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusForbidden, "page", err)
	}

	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	var pageObj helpers.SitePage
	var event internal_types.Event
	var isEditor bool = false
	if eventId == "" {
		pageObj = helpers.SitePages["add-event"]
	} else {
		pageObj = helpers.SitePages["edit-event"]
		marqoClient, err := services.GetMarqoClient()
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, "page", err)
		}
		eventPtr, err := services.GetMarqoEventByID(marqoClient, eventId, "")
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get event: "+err.Error()), http.StatusInternalServerError, "page", err)
		}
		if eventPtr != nil {
			event = *eventPtr
		}
		if event.EventOwners == nil {
			event.EventOwners = []string{}
		}
		canEdit := helpers.CanEditEvent(&event, &userInfo, roleClaims)
		if !canEdit {
			err := errors.New("You are not authorized to edit this event")
			return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusNotFound, "page", err)
		}
		isEditor = canEdit
	}

	cfRay := GetCfRay(r)
	rayCode := ""
	cfLocation := helpers.CdnLocation{}
	cfLocationLat := services.InitialEmptyLatLong
	cfLocationLon := services.InitialEmptyLatLong

	if len(cfRay) > 2 {
		rayCode = cfRay[len(cfRay)-3:]
		cfLocation = helpers.CfLocationMap[rayCode]
		cfLocationLat = cfLocation.Lat
		cfLocationLon = cfLocation.Lon
	}

	isCompetitionAdmin := helpers.HasRequiredRole(roleClaims, []string{"superAdmin", "competitionAdmin"})

	addOrEditEventPage := pages.AddOrEditEventPage(pageObj, userInfo, event, isEditor, cfLocationLat, cfLocationLon, isCompetitionAdmin)

	layoutTemplate := pages.Layout(pageObj, userInfo, addOrEditEventPage, event, []string{"https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js", "https://cdn.jsdelivr.net/npm/@alpinejs/sort@3.x.x/dist/cdn.min.js", "https://cdn.jsdelivr.net/npm/@alpinejs/mask@3.x.x/dist/cdn.min.js"})

	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusNotFound, "page", err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetEventAttendeesPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	roleClaims := []helpers.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = claims
	}

	validRoles := []string{"superAdmin", "eventAdmin"}
	if !helpers.HasRequiredRole(roleClaims, validRoles) {
		err := errors.New("Only event editors can add or edit events")
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusForbidden, "page", err)
	}

	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	var pageObj helpers.SitePage
	pageObj = helpers.SitePages["attendees-event"]
	var event types.Event
	var isEditor bool = false
	if eventId != "" {
		marqoClient, err := services.GetMarqoClient()
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, "page", err)
		}
		eventPtr, err := services.GetMarqoEventByID(marqoClient, eventId, "")
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get event: "+err.Error()), http.StatusInternalServerError, "page", err)
		}
		if eventPtr != nil {
			event = *eventPtr
		}
		if event.EventOwners == nil {
			event.EventOwners = []string{}
		}
		canEdit := helpers.CanEditEvent(&event, &userInfo, roleClaims)
		if !canEdit {
			err := errors.New("You are not authorized to edit this event")
			return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusNotFound, "page", err)
		}
		isEditor = canEdit
	}

	addOrEditEventPage := pages.EventAttendeesPage(pageObj, event, isEditor)

	layoutTemplate := pages.Layout(pageObj, userInfo, addOrEditEventPage, event, []string{})

	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusNotFound, "page", err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetMapEmbedPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	queryParameters := apiGwV2Req.QueryStringParameters

	mapEmbedPage := pages.MapEmbedPage(queryParameters["address"])
	layoutTemplate := pages.Layout(helpers.SitePages["embed"], userInfo, mapEmbedPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetPrivacyPolicyPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	privacyPolicyPage := pages.PrivacyPolicyPage(helpers.SitePages["privacy-policy"])
	layoutTemplate := pages.Layout(helpers.SitePages["privacy-policy"], userInfo, privacyPolicyPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetDataRequestPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	dataRequestPage := pages.DataRequestPage(helpers.SitePages["data-request"])
	layoutTemplate := pages.Layout(helpers.SitePages["data-request"], userInfo, dataRequestPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetTermsOfServicePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	termsOfServicePage := pages.TermsOfServicePage(helpers.SitePages["terms-of-service"], userInfo)
	layoutTemplate := pages.Layout(helpers.SitePages["terms-of-service"], userInfo, termsOfServicePage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetCfRay(r *http.Request) string {
	if cfRay := r.Header.Get("Cf-Ray"); cfRay != "" {
		return cfRay
	}
	return ""
}

func GetEventDetailsPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// TODO: Extract reading param values into a helper method.
	ctx := r.Context()
	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	parseDates := r.URL.Query().Get("parse_dates")
	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
	}
	event, err := services.GetMarqoEventByID(marqoClient, eventId, parseDates)
	if err != nil || event.Id == "" {
		event = &internal_types.Event{}
	}
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	roleClaims := []helpers.RoleClaim{}
	if _, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = ctx.Value("roleClaims").([]helpers.RoleClaim)
	}
	canEdit := helpers.CanEditEvent(event, &userInfo, roleClaims)

	eventDetailsPage := pages.EventDetailsPage(*event, userInfo, canEdit)
	layoutTemplate := pages.Layout(helpers.SitePages["event-detail"], userInfo, eventDetailsPage, *event, []string{})
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetAddEventSourcePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	adminPage := pages.AddEventSource()
	layoutTemplate := pages.Layout(helpers.SitePages["add-event-source"], userInfo, adminPage, types.Event{}, []string{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}

func GetAddOrEditCompetitionPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db := transport.GetDB()
	// Get user info from context
	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}

	// Get role claims from context
	roleClaims := []helpers.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]helpers.RoleClaim); ok {
		roleClaims = claims
	}

	// Check if user has required roles
	validRoles := []string{"superAdmin", "competitionAdmin"}
	if !helpers.HasRequiredRole(roleClaims, validRoles) {
		err := errors.New("You are not authorized to edit competitions.")
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusForbidden, "page", err)
	}

	// Get competition ID and event ID from URL
	vars := mux.Vars(r)
	competitionId := vars[helpers.COMPETITIONS_ID_KEY]

	var pageObj helpers.SitePage
	var competitionConfig internal_types.CompetitionConfig
	var users []types.UserSearchResultDangerous
	pageObj = helpers.SitePages["add-competition"]
	// Check if we are editing or adding
	if competitionId == "" {
		pageObj = helpers.SitePages["competition-new"]
		// Set default values for new competition
		competitionConfig = internal_types.CompetitionConfig{
			EventIds:     []string{},
			Status:       "DRAFT",
			PrimaryOwner: userInfo.Sub,
		}

	} else {
		eventCompetitionRoundService := dynamodb_service.NewCompetitionConfigService()
		pageObj = helpers.SitePages["competition-edit"]
		competitionConfigResponse, err := eventCompetitionRoundService.GetCompetitionConfigById(ctx, db, competitionId)
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get competition: "+err.Error()),
				http.StatusInternalServerError, "page", err)
		}

		if competitionConfigResponse.CompetitionConfig.Id == "" {
			return transport.SendHtmlRes(w, []byte("Competition not found"),
				http.StatusNotFound, "page", errors.New("empty competition ID"))
		}
		competitionConfig = competitionConfigResponse.CompetitionConfig
		users = competitionConfigResponse.Owners
	}

	competitionPage := pages.AddOrEditCompetitionPage(pageObj, competitionConfig, users)
	layoutTemplate := pages.Layout(pageObj, userInfo, competitionPage, internal_types.Event{}, []string{"https://cdn.jsdelivr.net/npm/@alpinejs/focus@3.x.x/dist/cdn.min.js"})

	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "page", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}
