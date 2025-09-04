package types

import (
	"net/url"
)

type EventInfo struct {
	EventTitle        string  `json:"event_title"`
	EventLocation     string  `json:"event_location"`
	EventStartTime    string  `json:"event_start_datetime"`
	EventEndTime      string  `json:"event_end_datetime"`
	EventTimezone     string  `json:"event_timezone"`
	EventURL          string  `json:"event_url"`
	EventDescription  string  `json:"event_description"`
	EventHostName     string  `json:"event_host_name"`
	EventLatitude     float64 `json:"event_latitude,omitempty"`
	EventLongitude    float64 `json:"event_longitude,omitempty"`
	KnownScrapeSource string  `json:"known_scrape_source"`
	ScrapeMode        string  `json:"scrape_mode"`
}

type EventBoolValid struct {
	EventValidateTitle       bool `json:"event_title" validate:"required"`
	EventValidateLocation    bool `json:"event_location" validate:"required"`
	EventValidateStartTime   bool `json:"event_start_datetime" validate:"required"`
	EventValidateEndTime     bool `json:"event_end_datetime"`
	EventValidateURL         bool `json:"event_url" validate:"required"`
	EventValidateDescription bool `json:"event_description"`
}

type SeshuSession struct {
	OwnerId           string           `json:"ownerId" validate:"required"`
	Url               string           `json:"url" validate:"required"`
	UrlDomain         string           `json:"urlDomain" validate:"required"`
	UrlPath           string           `json:"urlPath"`
	UrlQueryParams    url.Values       `json:"urlQueryParams"`
	LocationLatitude  float64          `json:"locationLatitude"`
	LocationLongitude float64          `json:"locationLongitude"`
	LocationAddress   string           `json:"locationAddress"`
	Html              string           `json:"html" validate:"required"`
	ChildId           string           `json:"childId" dynamodbav:"childId"`
	EventCandidates   []EventInfo      `json:"eventCandidates"`
	EventValidations  []EventBoolValid `json:"eventValidations"`
	Status            string           `json:"status"`
	CreatedAt         int64            `json:"createdAt" validate:"required"`
	UpdatedAt         int64            `json:"updatedAt" validate:"required"`
	ExpireAt          int64            `json:"expireAt" validate:"required"`
}

type SeshuSessionGet struct {
	OwnerId string `json:"ownerId" dynamodbav:"ownerId" validate:"required"`
	Url     string `json:"url" dynamodbav:"url" validate:"required"`
}

type SeshuSessionInput struct {
	SeshuSession
	CreatedAt struct{} `json:"createdAt,omitempty"`
	UpdatedAt struct{} `json:"updatedAt,omitempty"`
	ExpireAt  struct{} `json:"expireAt,omitempty"`
}

type SeshuSessionInsert struct {
	OwnerId           string           `json:"ownerId" dynamodbav:"ownerId" validate:"required"`
	Url               string           `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain         string           `json:"urlDomain" dynamodbav:"urlDomain" validate:"required"`
	UrlPath           string           `json:"urlPath" dynamodbav:"urlPath"`
	UrlQueryParams    url.Values       `json:"urlQueryParams" dynamodbav:"urlQueryParams"`
	LocationLatitude  float64          `json:"locationLatitude" dynamodbav:"locationLatitude"`
	LocationLongitude float64          `json:"locationLongitude" dynamodbav:"locationLongitude"`
	LocationAddress   string           `json:"locationAddress" dynamodbav:"locationAddress"`
	Html              string           `json:"html" dynamodbav:"html" validate:"required"`
	ChildId           string           `json:"childId" dynamodbav:"childId"`
	EventCandidates   []EventInfo      `json:"eventCandidates" dynamodbav:"eventCandidates"`
	EventValidations  []EventBoolValid `json:"eventValidations" dynamodbav:"eventValidations"`
	Status            string           `json:"status" dynamodbav:"status"`
	CreatedAt         int64            `json:"createdAt" dynamodbav:"createdAt" validate:"required"`
	UpdatedAt         int64            `json:"updatedAt" dynamodbav:"updatedAt" validate:"required"`
	ExpireAt          int64            `json:"expireAt" dynamodbav:"expireAt" validate:"required"`
}
type SeshuSessionUpdate struct {
	OwnerId           string           `json:"ownerId" dynamodbav:"ownerId"`
	Url               string           `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain         string           `json:"urlDomain" dynamodbav:"urlDomain"`
	UrlPath           string           `json:"urlPath" dynamodbav:"urlPath"`
	UrlQueryParams    url.Values       `json:"urlQueryParams" dynamodbav:"urlQueryParams"`
	LocationLatitude  float64          `json:"locationLatitude" dynamodbav:"locationLatitude"`
	LocationLongitude float64          `json:"locationLongitude" dynamodbav:"locationLongitude"`
	LocationAddress   string           `json:"locationAddress" dynamodbav:"locationAddress"`
	Html              string           `json:"html" dynamodbav:"html"`
	ChildId           string           `json:"childId" dynamodbav:"childId"`
	EventCandidates   []EventInfo      `json:"eventCandidates" dynamodbav:"eventCandidates"`
	EventValidations  []EventBoolValid `json:"eventValidations" dynamodbav:"eventValidations"`
	Status            string           `json:"status" dynamodbav:"status"`
	CreatedAt         int64            `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt         int64            `json:"updatedAt" dynamodbav:"updatedAt"`
	ExpireAt          int64            `json:"expireAt" dynamodbav:"expireAt"`
}

type SeshuJob struct {
	NormalizedUrlKey         string  `json:"normalized_url_key" validate:"required"`
	LocationLatitude         float64 `json:"location_latitude,omitempty"`
	LocationLongitude        float64 `json:"location_longitude,omitempty"`
	LocationAddress          string  `json:"location_address,omitempty"`
	ScheduledHour            int     `json:"scheduled_hour,omitempty" validate:"required"` // Hour of the day (0-23)
	TargetNameCSSPath        string  `json:"target_name_css_path" validate:"required"`
	TargetLocationCSSPath    string  `json:"target_location_css_path" validate:"required"`
	TargetStartTimeCSSPath   string  `json:"target_start_time_css_path" validate:"required"`
	TargetEndTimeCSSPath     string  `json:"target_end_time_css_path,omitempty"`
	TargetDescriptionCSSPath string  `json:"target_description_css_path,omitempty" validate:"required"`
	TargetHrefCSSPath        string  `json:"target_href_css_path,omitempty" validate:"required"`
	Status                   string  `json:"status" validate:"required"` // e.g. "HEALTHY", "WARNING", "FAILING"
	LastScrapeSuccess        int64   `json:"last_scrape_success,omitempty" validate:"required"`
	LastScrapeFailure        int64   `json:"last_scrape_failure,omitempty" validate:"gte=0"`
	LastScrapeFailureCount   int     `json:"last_scrape_failure_count" validate:"gte=0"`
	OwnerID                  string  `json:"owner_id" validate:"required"`
	KnownScrapeSource        string  `json:"known_scrape_source"` // e.g. "MEETUP", "EVENTBRITE", etc.
	LocationTimezone         *string `json:"location_timezone,omitempty"`
}
