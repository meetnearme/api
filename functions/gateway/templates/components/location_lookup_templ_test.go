package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meetnearme/api/functions/gateway/services"
)

func TestLocationLookupPartial(t *testing.T) {

	// Define test cases
	tests := []struct {
		name              string
		hxMethod          string
		hxApiPath         string
		hxAfterReqStr     string
		inputModelStr     string
		buttonPreText     string
		buttonPostText    string
		titleStr          string
		lat               float64
		lon               float64
		address           string
		cfLocationLat     float64
		cfLocationLon     float64
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name:           "Location lookup, post ",
			hxMethod:       "post",
			hxApiPath:      "/api/location/geo",
			hxAfterReqStr:  "",
			inputModelStr:  "formData.url",
			buttonPreText:  "Confirm Address",
			buttonPostText: "Address Confirmed",
			titleStr:       "Address",
			lat:            0,
			lon:            0,
			address:        "",
			cfLocationLat:  services.InitialEmptyLatLong,
			cfLocationLon:  services.InitialEmptyLatLong,
			expectedContent: []string{
				"hx-post=\"/api/location/geo\"",
				"hx-target=\"#location-confirmation\"",
				"@htmx:after-request=\"handleHtmxAfterReq(event)\"",
				"data-lat-lon-unknown=\"90000000000.000000\"",
				"data-cf-lat=\"90000000000.000000\"",
				"data-cf-lon=\"90000000000.000000\"",
			},
			unexpectedContent: []string{},
		},
		{
			name:           "Location lookup, patch, with hxApiPath and hxAfterReqStr",
			hxMethod:       "patch",
			hxApiPath:      "/fake/api/path",
			hxAfterReqStr:  "testAfterReq(event)",
			inputModelStr:  "formData.url",
			buttonPreText:  "Confirm Address",
			buttonPostText: "Address Confirmed",
			titleStr:       "Address",
			lat:            0,
			lon:            0,
			address:        "",
			cfLocationLat:  38.893726,
			cfLocationLon:  -77.096976,
			expectedContent: []string{
				"hx-patch=\"/fake/api/path\"",
				"hx-target=\"#location-confirmation\"",
				"@htmx:after-request=\"testAfterReq(event)\"",
				"data-lat-lon-unknown=\"90000000000.000000\"",
				"data-cf-lat=\"38.893726\"",
				"data-cf-lon=\"-77.096976\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// component := NestedCheckboxList(tt.isInDropdown, tt.interests)

			component := LocationLookupPartial(tt.hxMethod, tt.hxApiPath, tt.hxAfterReqStr, tt.inputModelStr, tt.buttonPreText, tt.buttonPostText, tt.lat, tt.lon, tt.address, tt.cfLocationLat, tt.cfLocationLon, tt.titleStr)
			// Render the template
			var buf bytes.Buffer
			err := component.Render(context.Background(), &buf)

			// Check for rendering errors
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Check if the rendered content contains expected information
			renderedContent := buf.String()

			for _, expected := range tt.expectedContent {
				if !strings.Contains(renderedContent, expected) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", expected)
				}
			}

			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(renderedContent, unexpected) {
					t.Errorf("Unexpected rendered content to contain '%s', but it did", unexpected)
				}
			}
		})
	}
}
