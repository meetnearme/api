package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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

	// TODO: this needs to be parameterized!
	htmlString, err := services.GetHTMLFromURL("https://vxk2uxg8v4.execute-api.us-east-1.amazonaws.com/map?address=" + inputPayload.Location, 500, false)

	if err != nil {
		return transport.SendHTMLError(err, ctx, req)
	}

	// this regex specifically captures the pattern of a lat/lon pair e.g. [40.7128, -74.0060]
	re := regexp.MustCompile(`\[\-?\+?([1-8]?\d(\.\d+)?|90(\.0+)?),\s*\-?\+?(180(\.0+)?|((1[0-7]\d)|([1-9]?\d))(\.\d+)?)\]`)
	latLon := re.FindString(htmlString)

	if latLon == "" {
		return transport.SendHTMLError(fmt.Errorf("location is not valid"), ctx, req)
	}

	latLonArr := strings.Split(latLon, ",")
	lat := latLonArr[0]
	lon := latLonArr[1]
	re = regexp.MustCompile(`[^\d.]`)
	lat = re.ReplaceAllString(lat, "")
	lon = re.ReplaceAllString(lon, "")

	// Regular expression pattern
	pattern := `"([^"]*)"\s*,\s*\` + latLon
	re = regexp.MustCompile(pattern)

	// Find the matches
	address := re.FindStringSubmatch(htmlString)

	if len(address) < 1 {
		address = []string{"", "No address found"}
	}

	geoLookupPartial := partials.GeoLookup(lat, lon, address[1])

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
