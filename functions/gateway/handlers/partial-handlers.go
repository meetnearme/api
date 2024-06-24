package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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

func GeoLookup(ctx context.Context, req transport.Request, db *dynamodb.Client) (transport.Response, error) {

	var inputPayload GeoLookupInputPayload
	err := json.Unmarshal([]byte(req.Body), &inputPayload)
	if err != nil {
			log.Printf("Invalid JSON payload: %v", err)
			return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
			log.Printf("Invalid body: %v", err)
			return transport.SendClientError(http.StatusBadRequest, "Invalid Body")
	}

	if err != nil {
			return transport.SendServerError(err)
	}

	lat, lon, address, err := services.GetGeo(inputPayload.Location)

	if err != nil {
		return transport.SendServerError(err)
	}

	geoLookupPartial := partials.GeoLookup(lat, lon, address, false)

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}

	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func GeoThenPatchSeshuSession(ctx context.Context, req transport.Request, db *dynamodb.Client) (transport.Response, error) {

	var inputPayload GeoThenSeshuPatchInputPayload
	err := json.Unmarshal([]byte(req.Body), &inputPayload)

	if err != nil {
			log.Printf("Invalid JSON payload: %v", err)
			return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
			log.Printf("Invalid body: %v", err)
			return transport.SendClientError(http.StatusBadRequest, "Invalid Body")
	}

	if err != nil {
			return transport.SendServerError(err)
	}

	lat, lon, address, err := services.GetGeo(inputPayload.Location)

	if err != nil {
		return transport.SendServerError(err)
	}

	var updateSehsuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(req.Body), &updateSehsuSession)

	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		log.Printf("Invalid latitude value: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid latitude value")
	}
	updateSehsuSession.LocationLatitude = latFloat
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		log.Printf("Invalid latitude value: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid longitude value")
	}
	updateSehsuSession.LocationLongitude = lonFloat
	updateSehsuSession.LocationAddress = address

	if err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return transport.SendClientError(http.StatusUnprocessableEntity, "Invalid JSON payload")
	}

	if (updateSehsuSession.Url == "") {
		var msg = "ERR: Invalid body: url is required"
		log.Println(msg)
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	geoLookupPartial := partials.GeoLookup(lat, lon, address, true)

	_, err = services.UpdateSeshuSession(ctx, db, updateSehsuSession)

	if err != nil {
		log.Printf("Error updating target URL session: %v", err)
		return transport.SendClientError(http.StatusNotFound, "Error updating target URL session")
	}

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}

	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func SubmitSeshuEvents(ctx context.Context, req transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var inputPayload SeshuSessionEventsPayload

	err := json.Unmarshal([]byte(req.Body), &inputPayload)
	if err != nil {
		msg := "Invalid JSON payload"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		msg := "Invalid request body"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	var updateSehsuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(req.Body), &updateSehsuSession)

	if err != nil {
		msg := "Invalid JSON payload"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	updateSehsuSession.Url = inputPayload.Url
	// Note that only OpenAI can push events as candidates, `eventValidations` is an array of
	// arrays that confirms the subfields, but avoids a scenario where users can push string data
	// that is prone to manipulation
	updateSehsuSession.EventValidations = inputPayload.EventValidations

	_, err = services.UpdateSeshuSession(ctx, db, updateSehsuSession)

	if err != nil {
		msg := "Failed to update Event Target URL session"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	successPartial := partials.SuccessBannerHTML(`We've noted the events you've confirmed as accurate`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}

	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}

func SubmitSeshuSession(ctx context.Context, req transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var inputPayload SeshuSessionSubmitPayload
	err := json.Unmarshal([]byte(req.Body), &inputPayload)
	if err != nil {
		msg := "Invalid JSON payload"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		msg := "Invalid request body"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	var updateSehsuSession services.SeshuSessionUpdate
	err = json.Unmarshal([]byte(req.Body), &updateSehsuSession)

	if err != nil {
		msg := "Invalid JSON payload"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	updateSehsuSession.Url = inputPayload.Url
	updateSehsuSession.Status = "submitted"

	_, err = services.UpdateSeshuSession(ctx, db, updateSehsuSession)

	if err != nil {
		msg := "Failed to update Event Target URL session"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}

	// TODO: this sets the session to `submitted`, in a follow-up PR this will call a function
	// that manages the handoff to the event scraping queue to do the real ingestion work

	successPartial := partials.SuccessBannerHTML(`Your Event Source his been added. We will put it in the queue and let you know when it's imported.`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerError(err)
	}

	return transport.Response{
		Headers:         map[string]string{"Content-Type": "text/html"},
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}, nil
}
