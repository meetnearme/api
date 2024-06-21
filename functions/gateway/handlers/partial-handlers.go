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

func SubmitSeshuSession(ctx context.Context, req transport.Request, db *dynamodb.Client) (transport.Response, error) {
	var inputPayload services.SeshuSessionUpdate
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

	_, err = services.UpdateSeshuSession(ctx, db, inputPayload)

	if err != nil {
		msg := "Failed to update Event Target URL session"
		log.Println(msg + ": " + err.Error())
		return transport.SendClientError(http.StatusBadRequest, msg)
	}


	successPartial := partials.SuccessBannerHTML(`Your Event Source his beend added. We will put it in the queue and let you know when it's imported.`)

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
