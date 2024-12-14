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
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
)

const US_GEO_CENTER_LAT = float64(39.8283)
const US_GEO_CENTER_LONG = float64(-98.5795)

func ParseStartEndTime(startTimeStr, endTimeStr string) (_startTimeUnix, _endTimeUnix int64) {
	var startTime time.Time
	var endTime time.Time

	var startTimeUnix int64
	var endTimeUnix int64

	// NOTE: This assumes the UI home page default is "THIS MONTH" and the absence
	// of an explicit start_time query param ...
	if (startTimeStr == "" && endTimeStr == "") || strings.ToLower(startTimeStr) == "this_year" {
		startTime = time.Now()
		endTime = startTime.AddDate(1, 0, 0)
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
	if _, err = time.Parse(time.RFC3339, endTimeStr); err == nil {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	} else if endTimeUnix, err = strconv.ParseInt(endTimeStr, 10, 64); err == nil {
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
	lat := US_GEO_CENTER_LAT
	long := US_GEO_CENTER_LONG

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
		lat != US_GEO_CENTER_LAT && long != US_GEO_CENTER_LONG) {
		radius = float64(150.0)
		// we still don't have lat/lon, which means we'll be using "geographic center of US"
		// which is in the middle of nowhere. Expand the radius to show all of the country
		// showing events from anywhere
	} else if radius == 0.0 {
		radius = float64(2500.0)
	}

	startTimeUnix, endTimeUnix := ParseStartEndTime(startTimeStr, endTimeStr)

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

func GetHomePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()
	q, userLocation, radius, startTimeUnix, endTimeUnix, cfLocation, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	originalQueryLat := r.URL.Query().Get("lat")
	originalQueryLong := r.URL.Query().Get("lon")

	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
	}

	subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")

	// we override the `owners` query param here, because subdomains should always show only
	// the owner as declared authoritatively by the subdomain ID lookup in Cloudflare KV
	if subdomainValue != "" {
		ownerIds = []string{subdomainValue}
	}

	res, err := services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
	}

	events := res.Events

	userInfo := helpers.UserInfo{}
	if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	homePage := pages.HomePage(
		events,
		cfLocation,
		fmt.Sprint(userLocation[0]),
		fmt.Sprint(userLocation[1]),
		fmt.Sprint(originalQueryLat),
		fmt.Sprint(originalQueryLong),
	)
	layoutTemplate := pages.Layout(helpers.SitePages["home"], userInfo, homePage, types.Event{})

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

	layoutTemplate := pages.Layout(helpers.SitePages["about"], helpers.UserInfo{}, aboutPage, types.Event{})
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
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
	adminPage := pages.ProfilePage(userInfo, roleClaims, userInterests, userSubdomain)
	layoutTemplate := pages.Layout(helpers.SitePages["profile"], userInfo, adminPage, types.Event{})
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
	layoutTemplate := pages.Layout(helpers.SitePages["settings"], userInfo, settingsPage, types.Event{})

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
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

	validRoles := []string{"superAdmin", "eventEditor"}
	if !helpers.HasRequiredRole(roleClaims, validRoles) {
		err := errors.New("Only event editors can add or edit events")
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusForbidden, "page", err)
	}

	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	var pageObj helpers.SitePage
	var event types.Event
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
	}
	addOrEditEventPage := pages.AddOrEditEventPage(pageObj, event, isEditor)

	layoutTemplate := pages.Layout(pageObj, userInfo, addOrEditEventPage, event)

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

	validRoles := []string{"superAdmin", "eventEditor"}
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
	}
	addOrEditEventPage := pages.EventAttendeesPage(pageObj, event, isEditor)

	layoutTemplate := pages.Layout(pageObj, userInfo, addOrEditEventPage, event)

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

	queryParameters := apiGwV2Req.QueryStringParameters

	mapEmbedPage := pages.MapEmbedPage(queryParameters["address"])
	layoutTemplate := pages.Layout(helpers.SitePages["embed"], helpers.UserInfo{}, mapEmbedPage, types.Event{})
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
		event = &types.Event{}
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
	checkoutParamVal := r.URL.Query().Get("checkout")

	eventDetailsPage := pages.EventDetailsPage(*event, checkoutParamVal, canEdit)
	layoutTemplate := pages.Layout(helpers.SitePages["event-detail"], userInfo, eventDetailsPage, *event)
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
	layoutTemplate := pages.Layout(helpers.SitePages["add-event-source"], userInfo, adminPage, types.Event{})
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "page", nil)
}
