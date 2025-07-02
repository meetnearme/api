package types

import (
	"net/url"
)

type EventInfo struct {
	EventTitle       string `json:"event_title"`
	EventLocation    string `json:"event_location"`
	EventStartTime   string `json:"event_start_datetime"`
	EventEndTime     string `json:"event_end_datetime"`
	EventURL         string `json:"event_url"`
	EventDescription string `json:"event_description"`
	EventSource      string `json:"event_source"`
}

type EventBoolValid struct {
	EventValidateTitle       bool `json:"event_title" validate:"required"`
	EventValidateLocation    bool `json:"event_location" validate:"required"`
	EventValidateStartTime   bool `json:"event_start_datetime" validate:"required"`
	EventValidateEndTime     bool `json:"event_end_datetime" validate:"optional"`
	EventValidateURL         bool `json:"event_url" validate:"required"`
	EventValidateDescription bool `json:"event_description" validate:"optional"`
}

type SeshuSession struct {
	OwnerId           string           `json:"ownerId" validate:"required"`
	Url               string           `json:"url" validate:"required"`
	UrlDomain         string           `json:"urlDomain" validate:"required"`
	UrlPath           string           `json:"urlPath" validate:"optional"`
	UrlQueryParams    url.Values       `json:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64          `json:"locationLatitude" validate:"optional"`
	LocationLongitude float64          `json:"locationLongitude" validate:"optional"`
	LocationAddress   string           `json:"locationAddress" validate:"optional"`
	Html              string           `json:"html" validate:"required"`
	ChildId           string           `json:"childId" dynamodbav:"childId" validate:"optional"`
	EventCandidates   []EventInfo      `json:"eventCandidates" validate:"optional"`
	EventValidations  []EventBoolValid `json:"eventValidations" validate:"optional"`
	Status            string           `json:"status" validate:"optional"`
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
	UrlPath           string           `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams    url.Values       `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64          `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64          `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string           `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html              string           `json:"html" dynamodbav:"html" validate:"required"`
	ChildId           string           `json:"childId" dynamodbav:"childId" validate:"optional"`
	EventCandidates   []EventInfo      `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations  []EventBoolValid `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status            string           `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt         int64            `json:"createdAt" dynamodbav:"createdAt" validate:"required"`
	UpdatedAt         int64            `json:"updatedAt" dynamodbav:"updatedAt" validate:"required"`
	ExpireAt          int64            `json:"expireAt" dynamodbav:"expireAt" validate:"required"`
}
type SeshuSessionUpdate struct {
	OwnerId           string           `json:"ownerId" dynamodbav:"ownerId" validate:"optional"`
	Url               string           `json:"url" dynamodbav:"url" validate:"required"`
	UrlDomain         string           `json:"urlDomain" dynamodbav:"urlDomain" validate:"optional"`
	UrlPath           string           `json:"urlPath" dynamodbav:"urlPath" validate:"optional"`
	UrlQueryParams    url.Values       `json:"urlQueryParams" dynamodbav:"urlQueryParams" validate:"optional"`
	LocationLatitude  float64          `json:"locationLatitude" dynamodbav:"locationLatitude" validate:"optional"`
	LocationLongitude float64          `json:"locationLongitude" dynamodbav:"locationLongitude" validate:"optional"`
	LocationAddress   string           `json:"locationAddress" dynamodbav:"locationAddress" validate:"optional"`
	Html              string           `json:"html" dynamodbav:"html" validate:"optional"`
	ChildId           string           `json:"childId" dynamodbav:"childId" validate:"optional"`
	EventCandidates   []EventInfo      `json:"eventCandidates" dynamodbav:"eventCandidates" validate:"optional"`
	EventValidations  []EventBoolValid `json:"eventValidations" dynamodbav:"eventValidations" validate:"optional"`
	Status            string           `json:"status" dynamodbav:"status" validate:"optional"`
	CreatedAt         int64            `json:"createdAt" dynamodbav:"createdAt" validate:"optional"`
	UpdatedAt         int64            `json:"updatedAt" dynamodbav:"updatedAt" validate:"optional"`
	ExpireAt          int64            `json:"expireAt" dynamodbav:"expireAt" validate:"optional"`
}

type SeshuJob struct {
	NormalizedURLKey         string  `json:"normalized_url_key"`
	LocationLatitude         float64 `json:"location_latitude,omitempty"`
	LocationLongitude        float64 `json:"location_longitude,omitempty"`
	LocationAddress          string  `json:"location_address,omitempty"`
	ScheduledScrapeTime      int64   `json:"scheduled_scrape_time"`
	TargetNameCSSPath        string  `json:"target_name_css_path"`
	TargetLocationCSSPath    string  `json:"target_location_css_path"`
	TargetStartTimeCSSPath   string  `json:"target_start_time_css_path"`
	TargetDescriptionCSSPath string  `json:"target_description_css_path,omitempty"`
	TargetHrefCSSPath        string  `json:"target_href_css_path,omitempty"`
	Status                   string  `json:"status"` // e.g. "HEALTHY", "WARNING", "FAILING"
	LastScrapeSuccess        int64   `json:"last_scrape_success,omitempty"`
	LastScrapeFailure        int64   `json:"last_scrape_failure,omitempty"`
	LastScrapeFailureCount   int     `json:"last_scrape_failure_count"`
	OwnerID                  string  `json:"owner_id"`
	KnownScrapeSource        string  `json:"known_scrape_source"` // e.g. "MEETUP", "EVENTBRITE", etc.
}
