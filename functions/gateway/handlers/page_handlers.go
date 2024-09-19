package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
)

func ParseStartEndTime(startTimeStr, endTimeStr string) (_startTimeUnix, _endTimeUnix int64) {
	var startTime time.Time
	var endTime time.Time

	var startTimeUnix int64
	var endTimeUnix int64

	// NOTE: This assumes the UI home page default is "THIS MONTH" and the absence
	// of an explicit start_time query param ...
	if (startTimeStr == "" && endTimeStr == "" )|| strings.ToLower(startTimeStr) == "this_month" {
		startTime = time.Now()
		endTime = startTime.AddDate(0,1,0)
	} else if strings.ToLower(startTimeStr) == "today" {
		startTime = time.Now()
		endTime = startTime.AddDate(0,0,1)
	// NOTE: "tomorrow" is a time-bound concept that should eventually be timezone relative
	// to the user, this is currently simplistic and is just 24 - 48hrs from the current time
	} else if strings.ToLower(startTimeStr) == "tomorrow" {
		startTime = time.Now().AddDate(0,0,1)
		endTime = startTime.AddDate(0,0,1)
	} else if strings.ToLower(startTimeStr) == "this_week" {
		startTime = time.Now()
		endTime = startTime.AddDate(0,0,7)
	}

	// return early if one of the above are found
	if !startTime.IsZero() && !endTime.IsZero() {
		return startTime.Unix(), endTime.Unix()
	}

	// convert startTime either UTC / time.RFC3339 or integer
	// string (presumed unix) to int64
	if	_, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	} else if startTimeUnix, err = strconv.ParseInt(startTimeStr, 10, 64); err == nil {
		startTime = time.Unix(startTimeUnix, 0)
	// default wrong query string usage to NOW for startTime
	} else {
		startTime = time.Now()
	}

	// convert endTime either UTC / time.RFC3339 or integer
	// string (presumed unix) to int64
	if	_, err = time.Parse(time.RFC3339, endTimeStr); err == nil {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	} else if endTimeUnix, err = strconv.ParseInt(endTimeStr, 10, 64); err == nil {
		endTime = time.Unix(endTimeUnix, 0)
	// Set end time to 24 hours after start time
	// default wrong query string usage to PLUS ONE MONTH for endTime
	} else {
		endTime = startTime.AddDate(0,1,0)
	}

	startTimeUnix = startTime.Unix()
	endTimeUnix = endTime.Unix()

	return startTimeUnix, endTimeUnix
}

func GetSearchParamsFromReq(r *http.Request) (query string, userLocation []float64, maxDistance float64, startTime int64, endTime int64, cfLocation helpers.CdnLocation)  {
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	latStr := r.URL.Query().Get("lat")
	longStr := r.URL.Query().Get("lon")
	radiusStr := r.URL.Query().Get("radius")
	q := r.URL.Query().Get("q")
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
	lat := float64(39.8283)
	long := float64(-98.5795)

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
	if radius == 0.0 && cfLocationLat != services.InitialEmptyLatLong && cfLocationLon != services.InitialEmptyLatLong {
		radius = float64(150.0)
	// we still don't have lat/lon, which means we'll be using "geographic center of US"
	// which is in the middle of nowhere. Expand the radius to show all of the country
	// showing events from anywhere
	} else if radius == 0.0 {
		radius = float64(2500.0)
	}


	startTimeUnix, endTimeUnix := ParseStartEndTime(startTimeStr, endTimeStr)

	return q, []float64{lat, long}, radius, startTimeUnix, endTimeUnix, cfLocation
}

func GetHomePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()
	q, userLocation, radius, startTimeUnix, endTimeUnix, cfLocation := GetSearchParamsFromReq(r)

	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
	}

	subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")

	ownerIds := []string{}
	if subdomainValue != "" {
		ownerIds = append(ownerIds, subdomainValue)
	}

	res, err := services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
	}

	events := res.Events

	var userInfo helpers.UserInfo
	if ctx.Value("userInfo") != nil {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	homePage := pages.HomePage(events, cfLocation, fmt.Sprint(userLocation[0]), fmt.Sprint(userLocation[1]))
	layoutTemplate := pages.Layout("Home", userInfo, homePage)

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetProfilePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	roleClaims := ctx.Value("roleClaims").([]helpers.RoleClaim)

	adminPage := pages.ProfilePage(userInfo, roleClaims)
	layoutTemplate := pages.Layout("Admin", userInfo, adminPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetMapEmbedPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)

	queryParameters := apiGwV2Req.QueryStringParameters

	mapEmbedPage := pages.MapEmbedPage(queryParameters["address"])
	layoutTemplate := pages.Layout("Embed", helpers.UserInfo{}, mapEmbedPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetCfRay(r *http.Request) string {
	log.Printf(`r.Header.Get("Cf-Ray"): %+v`,r.Header.Get("Cf-Ray"))
	if cfRay := r.Header.Get("Cf-Ray"); cfRay != "" {
		return cfRay
	}
	return ""
}

func GetEventDetailsPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// TODO: Extract reading param values into a helper method.
	ctx := r.Context()
	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]
	if eventId == "" {
		// TODO: If no eventID is passed, return a 404 page or redirect to events list.
		fmt.Println("No event ID provided. Redirecting to home page.")
		http.Redirect(w, r, "/", http.StatusFound)
	}
	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
	}
	event, err := services.GetMarqoEventByID(marqoClient, eventId)
	if err != nil || event.Id == "" {
		event = &services.Event{}
	}
	eventDetailsPage := pages.EventDetailsPage(*event)
	var userInfo helpers.UserInfo
	if ctx.Value("userInfo") != nil {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}

	layoutTemplate := pages.Layout("Event Details", userInfo, eventDetailsPage)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetAddEventSourcePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var userInfo helpers.UserInfo
	if ctx.Value("userInfo") != nil {
		userInfo = ctx.Value("userInfo").(helpers.UserInfo)
	}
	adminPage := pages.AddEventSource()
	layoutTemplate := pages.Layout("Admin", userInfo, adminPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}
