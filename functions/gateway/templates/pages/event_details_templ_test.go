package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/types"
)

func TestEventDetailsPage(t *testing.T) {
	dstTm := "2099-05-02T00:00:00Z"
	loc, _ := time.LoadLocation("America/Los_Angeles")
	validEventDSTStartTime, err := helpers.UtcToUnix64(dstTm, loc)
	if err != nil || validEventDSTStartTime == 0 {
		t.Fatalf("Failed to convert UTC to unix: %v", err)
	}
	nonDstTm := "2099-01-31T00:00:00Z"
	validEventNonDSTStartTime, err := helpers.UtcToUnix64(nonDstTm, loc)
	if err != nil || validEventNonDSTStartTime == 0 {
		t.Fatalf("Failed to convert UTC to unix: %v", err)
	}

	tests := []struct {
		name             string
		event            types.Event
		checkoutParamVal string
		expected         []string
		canEdit          bool
	}{
		{
			name: "Valid DST event",
			event: types.Event{
				Id:              "123",
				Name:            "Test Event",
				Description:     "This is a test event",
				Address:         "123 Test St",
				StartTime:       validEventDSTStartTime,
				EventOwners:     []string{"abc-uuid"},
				EventOwnerName:  "Brians Pub",
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Lat:             38.896305,
				Long:            -77.023289,
				Timezone:        *loc,
			},
			checkoutParamVal: "",
			expected: []string{
				"Test Event",
				"This is a test event",
				"123 Test St",
				"May 2, 2099",
				"12:00am",
				"abc-uuid",
				"Brians Pub",
			},
			canEdit: false,
		},
		{
			name: "Valid Non-DST event",
			event: types.Event{
				Id:              "123",
				Name:            "Test Event",
				Description:     "This is a test event",
				Address:         "123 Test St",
				StartTime:       validEventNonDSTStartTime,
				EventOwners:     []string{"abc-uuid"},
				EventOwnerName:  "Brians Pub",
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Lat:             38.896305,
				Long:            -77.023289,
				Timezone:        *loc,
			},
			checkoutParamVal: "",
			expected: []string{
				"Test Event",
				"This is a test event",
				"123 Test St",
				"Jan 31, 2099",
				"12:00am",
				"abc-uuid",
				"Brians Pub",
			},
			canEdit: false,
		},
		{
			name: "Valid single event",
			event: types.Event{
				Id:              "123",
				Name:            "Karaoke Nationals",
				Description:     "This is a test event",
				Address:         "123 Test St",
				StartTime:       validEventNonDSTStartTime,
				EventOwners:     []string{"abc-uuid"},
				EventOwnerName:  "Brians Pub",
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Lat:             38.896305,
				Long:            -77.023289,
				Timezone:        *loc,
			},
			checkoutParamVal: "",
			expected: []string{
				"Karaoke Nationals",
				"This is a test event",
				"123 Test St",
				"Jan 31, 2099",
				"12:00am",
				"abc-uuid",
				"Brians Pub",
				"<title>Meet Near Me - Karaoke Nationals</title>",
			},
			canEdit: false,
		},
		{
			name: "Valid series event",
			event: types.Event{
				Id:              "123",
				Name:            "Weekly Karaoke at Buddys",
				Description:     "This is a test event",
				Address:         "123 Test St",
				StartTime:       validEventNonDSTStartTime,
				EventOwners:     []string{"abc-uuid"},
				EventOwnerName:  "Brians Pub",
				EventSourceType: helpers.ES_EVENT_SERIES,
				Lat:             38.896305,
				Long:            -77.023289,
				Timezone:        *loc,
			},
			checkoutParamVal: "",
			expected: []string{
				"Weekly Karaoke at Buddys",
				"This is a test event",
				"123 Test St",
				"Jan 31, 2099",
				"12:00am",
				"abc-uuid",
				"Brians Pub",
				"<title>Meet Near Me - Weekly Karaoke at Buddys</title>",
			},
			canEdit: false,
		},
		{
			name: "Valid event, editor role sees edit button",
			event: types.Event{
				Id:              "123",
				Name:            "Test Event",
				Description:     "This is a test event",
				Address:         "123 Test St",
				StartTime:       validEventNonDSTStartTime,
				EventOwners:     []string{"abc-uuid"},
				EventOwnerName:  "Brians Pub",
				EventSourceType: helpers.ES_SINGLE_EVENT,
				Lat:             38.896305,
				Long:            -77.023289,
				Timezone:        *loc,
			},
			checkoutParamVal: "",
			expected: []string{
				"Test Event",
				"This is a test event",
				"123 Test St",
				"Jan 31, 2099",
				"12:00am",
				"abc-uuid",
				"Brians Pub",
				"editor for this event",
			},
			canEdit: true,
		},
		{
			name:  "Empty event",
			event: types.Event{},
			expected: []string{
				"404 - Can't Find That Event",
			},
			canEdit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := EventDetailsPage(tt.event, helpers.UserInfo{}, tt.checkoutParamVal, tt.canEdit)

			// Wrap the component with Layout
			layoutTemplate := Layout(helpers.SitePages["event-detail"], helpers.UserInfo{}, component, tt.event)

			// Render the component to a string
			var buf bytes.Buffer
			err := layoutTemplate.Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Error rendering component: %v", err)
			}
			result := buf.String()
			// Check if all expected strings are in the result
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("Expected string not found: %s", exp)
					t.Logf("Result: %s", result)
				}
			}
		})
	}
}
