package types

import (
	"net/url"

	"github.com/meetnearme/api/functions/gateway/constants"
	"gorm.io/gorm"
)

type EventInfo struct {
	EventTitle        string  `json:"event_title"`
	EventLocation     string  `json:"event_location"`
	EventStartTime    string  `json:"event_start_datetime"`
	EventEndTime      string  `json:"event_end_datetime"`
	EventTimezone     string  `json:"event_timezone,omitempty"`
	EventURL          string  `json:"event_url"`
	EventDescription  string  `json:"event_description"`
	EventHostName     string  `json:"event_host_name"`
	EventLatitude     float64 `json:"event_latitude,omitempty"`
	EventLongitude    float64 `json:"event_longitude,omitempty"`
	KnownScrapeSource string  `json:"known_scrape_source"`
	ScrapeMode        string  `json:"scrape_mode"`
	SourceUrl         string  `json:"source_url,omitempty"`
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
	LocationLatitude  *float64         `json:"locationLatitude" dynamodbav:"locationLatitude"`
	LocationLongitude *float64         `json:"locationLongitude" dynamodbav:"locationLongitude"`
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
	NormalizedUrlKey              string  `json:"normalized_url_key" validate:"required" gorm:"column:normalized_url_key;primaryKey"`
	LocationLatitude              float64 `json:"location_latitude,omitempty" gorm:"column:location_latitude"`
	LocationLongitude             float64 `json:"location_longitude,omitempty" gorm:"column:location_longitude"`
	LocationAddress               string  `json:"location_address,omitempty" gorm:"column:location_address"`
	ScheduledHour                 int     `json:"scheduled_hour,omitempty" validate:"min=0,max=23" gorm:"column:scheduled_hour"` // Hour of the day (0-23)
	TargetNameCSSPath             string  `json:"target_name_css_path" validate:"required" gorm:"column:target_name_css_path"`
	TargetLocationCSSPath         string  `json:"target_location_css_path" validate:"required" gorm:"column:target_location_css_path"`
	TargetStartTimeCSSPath        string  `json:"target_start_time_css_path" validate:"required" gorm:"column:target_start_time_css_path"`
	TargetEndTimeCSSPath          string  `json:"target_end_time_css_path,omitempty" gorm:"column:target_end_time_css_path"`
	TargetDescriptionCSSPath      string  `json:"target_description_css_path,omitempty" validate:"required" gorm:"column:target_description_css_path"`
	TargetHrefCSSPath             string  `json:"target_href_css_path,omitempty" validate:"required" gorm:"column:target_href_css_path"`
	TargetChildNameCSSPath        string  `json:"target_child_name_css_path,omitempty" gorm:"column:target_child_name_css_path"`
	TargetChildLocationCSSPath    string  `json:"target_child_location_css_path,omitempty" gorm:"column:target_child_location_css_path"`
	TargetChildStartTimeCSSPath   string  `json:"target_child_start_time_css_path,omitempty" gorm:"column:target_child_start_time_css_path"`
	TargetChildEndTimeCSSPath     string  `json:"target_child_end_time_css_path,omitempty" gorm:"column:target_child_end_time_css_path"`
	TargetChildDescriptionCSSPath string  `json:"target_child_description_css_path,omitempty" gorm:"column:target_child_description_css_path"`
	IsRecursive                   bool    `json:"is_recursive,omitempty" gorm:"column:is_recursive"`
	Status                        string  `json:"status" validate:"required" gorm:"column:status"` // e.g. "HEALTHY", "WARNING", "FAILING"
	LastScrapeSuccess             int64   `json:"last_scrape_success,omitempty" validate:"required" gorm:"column:last_scrape_success"`
	LastScrapeFailure             int64   `json:"last_scrape_failure,omitempty" validate:"gte=0" gorm:"column:last_scrape_failure"`
	LastScrapeFailureCount        int     `json:"last_scrape_failure_count" validate:"gte=0" gorm:"column:last_scrape_failure_count"`
	OwnerID                       string  `json:"owner_id" validate:"required" gorm:"column:owner_id"`
	KnownScrapeSource             string  `json:"known_scrape_source" gorm:"column:known_scrape_source"` // e.g. "MEETUP", "EVENTBRITE", etc.
	LocationTimezone              string  `json:"location_timezone,omitempty" gorm:"column:location_timezone"`
}

// TableName tells GORM the exact table name to use for SeshuJob.
// The existing DB uses the singular/concatenated name `seshujobs` (see seshujobs_init.sql),
// while GORM's default pluralization would look for `seshu_jobs` which causes "relation does not exist" errors.
func (SeshuJob) TableName() string {
	return "seshujobs"
}

type Locatable interface {
	GetLocationLatitude() float64
	GetLocationLongitude() float64
	SetLocationLatitude(lat float64)
	SetLocationLongitude(lng float64)
}

func (s *SeshuSession) GetLocationLatitude() float64     { return s.LocationLatitude }
func (s *SeshuSession) GetLocationLongitude() float64    { return s.LocationLongitude }
func (s *SeshuSession) SetLocationLatitude(lat float64)  { s.LocationLatitude = lat }
func (s *SeshuSession) SetLocationLongitude(lng float64) { s.LocationLongitude = lng }

func (s *SeshuSessionInsert) GetLocationLatitude() float64     { return s.LocationLatitude }
func (s *SeshuSessionInsert) GetLocationLongitude() float64    { return s.LocationLongitude }
func (s *SeshuSessionInsert) SetLocationLatitude(lat float64)  { s.LocationLatitude = lat }
func (s *SeshuSessionInsert) SetLocationLongitude(lng float64) { s.LocationLongitude = lng }

func (s *SeshuSessionUpdate) GetLocationLatitude() float64 {
	if s.LocationLatitude == nil {
		return 0
	}
	return *s.LocationLatitude
}

func (s *SeshuSessionUpdate) GetLocationLongitude() float64 {
	if s.LocationLongitude == nil {
		return 0
	}
	return *s.LocationLongitude
}

func (s *SeshuSessionUpdate) SetLocationLatitude(lat float64) {
	s.LocationLatitude = &lat
}

func (s *SeshuSessionUpdate) SetLocationLongitude(lng float64) {
	s.LocationLongitude = &lng
}

func (j *SeshuJob) GetLocationLatitude() float64     { return j.LocationLatitude }
func (j *SeshuJob) GetLocationLongitude() float64    { return j.LocationLongitude }
func (j *SeshuJob) SetLocationLatitude(lat float64)  { j.LocationLatitude = lat }
func (j *SeshuJob) SetLocationLongitude(lng float64) { j.LocationLongitude = lng }

// Ensures defaults are set before creating a new record
// GORM automatically invokes these hooks
func (j *SeshuJob) BeforeCreate(tx *gorm.DB) (err error) {
	if j.LocationLatitude == 0 && j.LocationLongitude == 0 {
		j.LocationLatitude = constants.INITIAL_EMPTY_LAT_LONG
		j.LocationLongitude = constants.INITIAL_EMPTY_LAT_LONG
	}
	return nil
}

// BeforeSave is called both for create and update, ensuring consistency
func (j *SeshuJob) BeforeSave(tx *gorm.DB) (err error) {
	if j.LocationLatitude == 0 && j.LocationLongitude == 0 {
		j.LocationLatitude = constants.INITIAL_EMPTY_LAT_LONG
		j.LocationLongitude = constants.INITIAL_EMPTY_LAT_LONG
	}
	return nil
}
