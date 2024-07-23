package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	"net/http"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"
)

type GeoLookupInputPayload struct {
	Location string `json:"location" validate:"required"`
}

type GeoThenSeshuPatchInputPayload struct {
	Location string `json:"location" validate:"required"`
	Url string `json:"url" validate:"required"` // URL is the DB key in SeshuSession
}

type SeshuSessionSubmitPayload struct {
	Url string `json:"url" validate:"required"` // URL is the DB key in SeshuSession
}

type SeshuSessionEventsPayload struct {
	Url string `json:"url" validate:"required"` // URL is the DB key in SeshuSession
	EventValidations [][]bool `json:"eventValidations" validate:"required"`
}

func GeoLookup(w http.ResponseWriter, r *http.Request) http.HandlerFunc {

	ctx := r.Context()
	var inputPayload GeoLookupInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
			return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)

	if err != nil {
			return transport.SendServerRes(w, []byte(string("Invalid Body: ") + err.Error()), http.StatusBadRequest, err)
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		return transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, err)
	}

	lat, lon, address, err := services.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error getting geocoordinates: ") + err.Error()), http.StatusInternalServerError, err)
	}

	geoLookupPartial := partials.GeoLookup(lat, lon, address)

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GeoThenPatchSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var inputPayload GeoThenSeshuPatchInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid Body: "+err.Error()), http.StatusBadRequest, err)
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		return transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, err)
	}
	lat, lon, address, err := services.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to get geocoordinates: "+err.Error()), http.StatusInternalServerError, err)
	}

	var updateSeshuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, err)
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusUnprocessableEntity, err)
	}

	updateSeshuSession.LocationLatitude = latFloat
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusUnprocessableEntity, err)
	}
	updateSeshuSession.LocationLongitude = lonFloat
	updateSeshuSession.LocationAddress = address

	if (updateSeshuSession.Url == "") {
		return transport.SendHtmlRes(w, []byte("ERR: Invalid body: url is required"), http.StatusBadRequest, nil)
	}
	geoLookupPartial := partials.GeoLookup(lat, lon, address)

	_, err = services.UpdateSeshuSession(ctx, Db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update target URL session"), http.StatusNotFound, err)
	}

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func SubmitSeshuEvents(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var inputPayload SeshuSessionEventsPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, err)
	}

	var updateSeshuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, err)
	}

	updateSeshuSession.Url = inputPayload.Url
	// Note that only OpenAI can push events as candidates, `eventValidations` is an array of
	// arrays that confirms the subfields, but avoids a scenario where users can push string data
	// that is prone to manipulation
	updateSeshuSession.EventValidations = inputPayload.EventValidations

	_, err = services.UpdateSeshuSession(ctx, Db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, err)
	}

	successPartial := partials.SuccessBannerHTML(`We've noted the events you've confirmed as accurate`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func SubmitSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {

	ctx := r.Context()
	var inputPayload SeshuSessionSubmitPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, err)
	}

	var updateSeshuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, err)
	}

	updateSeshuSession.Url = inputPayload.Url
	updateSeshuSession.Status = "submitted"

	_, err = services.UpdateSeshuSession(ctx, Db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, err)
	}

	// TODO: this sets the session to `submitted`, in a follow-up PR this will call a function
	// that manages the handoff to the event scraping queue to do the real ingestion work

	successPartial := partials.SuccessBannerHTML(`Your Event Source his been added. We will put it in the queue and let you know when it's imported.`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)

	// TODO: delete me, this is just a test
}
