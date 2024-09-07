package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/zitadel-go/v3/pkg/authentication"
	openid "github.com/zitadel/zitadel-go/v3/pkg/authentication/oidc"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/transport"
)

var mw *authentication.Interceptor[*openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]]

func setUserInfo(authCtx *openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo], userInfo helpers.UserInfo) (helpers.UserInfo, error) {
	if authCtx != nil && authCtx.UserInfo != nil {
		data, err := json.MarshalIndent(authCtx.UserInfo, "", "	")
		if err != nil {
			return userInfo, err
		}
		err = json.Unmarshal(data, &userInfo)
		if err != nil {
			return userInfo, err
		}
	}
	return userInfo, nil
}

func GetHomePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()

	db := transport.GetDB()
	apiGwV2Req, ok := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)
	if !ok {
		log.Println("APIGatewayV2HTTPRequest not found in context, creating default")
		// For testing or non-API gateway envs
		apiGwV2Req = events.APIGatewayV2HTTPRequest{
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					Method: r.Method,
					Path:   r.URL.Path,
				},
			},
		}
	}

	// cfRay := GetCfRay(ctx)
	// rayCode := ""
	var cfLocation helpers.CdnLocation
	cfLocationLat := services.InitialEmptyLatLon
	cfLocationLon := services.InitialEmptyLatLon
	//   if len(cfRay) > 2 {
	//     rayCode = cfRay[len(cfRay)-3:]
	// 		cfLocation = helpers.CfLocationMap[rayCode]
	// 	  cfLocationLat = cfLocation.Lat
	//     cfLocationLon = cfLocation.Lon
	//   }

	queryParameters := apiGwV2Req.QueryStringParameters
	startTimeStr := queryParameters["start_time"]
	endTimeStr := queryParameters["end_time"]
	latStr := queryParameters["lat"]
	lonStr := queryParameters["lon"]
	radiusStr := queryParameters["radius"]

	// Set default values if query parameters are not provided
	startTime := time.Now()
	endTime := startTime.AddDate(100, 0, 0)
	lat := float32(39.8283)
	lon := float32(-98.5795)
	radius := float32(2500.0)

	// Parse parameter values if provided
	if startTimeStr != "" {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	}
	if endTimeStr != "" {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	}
	if latStr != "" {
		lat64, _ := strconv.ParseFloat(latStr, 32)
		lat = float32(lat64)
	} else if cfLocationLat != services.InitialEmptyLatLon {
		lat = float32(cfLocationLat)
	}
	if lonStr != "" {
		lon64, _ := strconv.ParseFloat(lonStr, 32)
		lon = float32(lon64)
	} else if cfLocationLon != services.InitialEmptyLatLon {
		lon = float32(cfLocationLon)
	}
	if radiusStr != "" {
		radius64, _ := strconv.ParseFloat(radiusStr, 32)
		radius = float32(radius64)
	}

	// Call the GetEventsZOrder service to retrieve events
	events, err := services.GetEventsZOrder(ctx, db, startTime, endTime, lat, lon, radius)

	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events by ZOrder: "+err.Error()), http.StatusInternalServerError, err)
	}

	var authCtx *openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]
	if mw != nil {
		authCtx = mw.Context(r.Context())
	} else {
		authCtx = &openid.UserInfoContext[*oidc.IDTokenClaims, *oidc.UserInfo]{}
	}

	userInfo := helpers.UserInfo{}
	userInfo, err = setUserInfo(authCtx, userInfo)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}
	homePage := pages.HomePage(events, cfLocation, latStr, lonStr)
	layoutTemplate := pages.Layout("Home", userInfo, homePage)

	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetLoginPage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	loginPage := pages.LoginPage()
	layoutTemplate := pages.Layout("Login", helpers.UserInfo{}, loginPage)
	var buf bytes.Buffer
	err := layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetProfilePage(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	// mw, _ := services.GetAuthMw()
	userInfoCtx := mw.Context(ctx)

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userInfo, err := setUserInfo(userInfoCtx, userInfo)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}
	adminPage := pages.ProfilePage(userInfo)
	layoutTemplate := pages.Layout("Admin", userInfo, adminPage)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
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

func GetCfRay(c context.Context) string {
	apiGwV2Req, ok := c.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest)
	if !ok {
		log.Println(("APIGatewayV2HTTPRequest not found in context"))
		return ""
	}
	if apiGwV2Req.Headers == nil {
		log.Println(("Headers not found in APIGatewayV2HTTPRequest"))
		return ""
	}
	if cfRay := apiGwV2Req.Headers["cf-ray"]; cfRay != "" {
		log.Println(("cf-ray found in APIGatewayV2HTTPRequest: " + fmt.Sprint(cfRay)))
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
	db := transport.GetDB()
	authCtx := mw.Context(ctx)

	event, err := services.GetEventById(ctx, db, eventId)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to get event: "+err.Error()), http.StatusInternalServerError, err)
	}

	eventDetailsPage := pages.EventDetailsPage(*event)
	userInfo := helpers.UserInfo{}
	userInfo, err = setUserInfo(authCtx, userInfo)
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
	authCtx := mw.Context(ctx)
	userInfo := helpers.UserInfo{}
	userInfo, err := setUserInfo(authCtx, userInfo)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}
	adminPage := pages.AddEventSource()
	layoutTemplate := pages.Layout("Admin", userInfo, adminPage)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte(err.Error()), http.StatusNotFound, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}
