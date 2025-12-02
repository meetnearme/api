package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestWidget(t *testing.T) {
	ctx := context.Background()

	loc, _ := time.LoadLocation("America/New_York")
	events := []types.Event{
		{
			Id:              "123",
			Name:            "Test Event 1",
			Description:     "Description for Test Event 1",
			Address:         "123 Test St",
			Lat:             40.7128,
			Long:            -74.0060,
			StartTime:       1704067200,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
		{
			Id:              "456",
			Name:            "Test Event 2",
			Description:     "Description for Test Event 2",
			Address:         "456 Test St",
			Lat:             40.7580,
			Long:            -73.9855,
			StartTime:       1704153600,
			Timezone:        *loc,
			EventSourceType: constants.ES_SINGLE_EVENT,
		},
	}

	origLatStr := ""
	origLonStr := ""
	tests := []struct {
		name              string
		pageUser          *types.UserSearchResult
		cfLocation        constants.CdnLocation
		cityStr           string
		latStr            string
		lonStr            string
		origQueryLocation string
		expectedItems     []string
		unexpectedItems   []string
	}{
		{
			name:              "No location set",
			pageUser:          nil,
			cityStr:           "",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"What type of local events are you looking to find?",
				"123 Test St",
			},
			unexpectedItems: []string{
				"Meet Near Me LLC. All rights reserved.",
			},
		},
		{
			name:              "In Los Angeles",
			pageUser:          nil,
			cityStr:           "Los Angeles, California",
			latStr:            "",
			lonStr:            "",
			origQueryLocation: "",
			expectedItems: []string{
				"Test Event 1",
				"Test Event 2",
				"New York, US",
				"Los Angeles",
				"What type of local events are you looking to find?",
				"123 Test St",
			},
			unexpectedItems: []string{
				"Meet Near Me LLC. All rights reserved.",
			},
		},
	}

	for _, tt := range tests {
		t.Run("widget loads and displays content", func(t *testing.T) {
			widget := Widget(ctx, events, tt.pageUser, tt.cfLocation, tt.cityStr, tt.latStr, tt.lonStr, origLatStr, origLonStr, tt.origQueryLocation, "test")

			var buf bytes.Buffer
			err := widget.Render(ctx, &buf)

			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			renderedContent := buf.String()

			for _, element := range tt.expectedItems {
				if !strings.Contains(renderedContent, element) {
					t.Errorf("Expected rendered content to contain '%s', but it didn't", element)
				}
			}

			for _, element := range tt.unexpectedItems {
				if strings.Contains(renderedContent, element) {
					t.Errorf("Expected rendered content to NOT contain '%s', but it did", element)
				}
			}
		})
	}
}
