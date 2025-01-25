package helpers

import (
	"fmt"
	"reflect"

	"github.com/meetnearme/api/functions/gateway/types"
)

type AWSReqKey string

const ApiGwV2ReqKey AWSReqKey = "ApiGwV2Req"

const RsvpsTablePrefix = "EventRsvps"
const PurchasesTablePrefix = "PurchasesV2"
const PurchasablesTablePrefix = "Purchasables"
const SeshuSessionTablePrefix = "SeshuSessions"
const RegistrationsTablePrefix = "Registrations"
const RegistrationFieldsTablePrefix = "RegistrationFields"
const CompetitionResultsTablePrefix = "CompetitionResults"
const CompetitionConfigTablePrefix = "CompetitionConfig"
const CompetitionRoundsTablePrefix = "CompetitionRounds"
const VotesTablePrefix = "Votes"

const EVENT_ID_KEY string = "eventId"
const PRIMARY_OWNER_KEY string = "primaryOwner"
const COMPETITIONS_ID_KEY string = "competitionId"
const ROUND_NUMBER_KEY string = "roundNumber"
const USER_ID_KEY string = "userId"
const SUBDOMAIN_KEY = "subdomain"
const INTERESTS_KEY = "interests"
const META_ABOUT_KEY = "about"
const ERR_KV_KEY_EXISTS = "key already exists in KV store"
const GO_TEST_ENV = "test"

const MOCK_CLOUDFLARE_URL = "http://localhost:8999"
const MOCK_ZITADEL_HOST = "localhost:8998"
const MOCK_MARQO_URL = "http://localhost:8997"

const JWT_ASSERTION_TYPE = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
const AUTH_ROLE_CLAIMS_KEY = "urn:zitadel:iam:org:project:<project-id>:roles"
const AUTH_METADATA_KEY = "urn:zitadel:iam:user:metadata"

const DEFAULT_PAGINATION_LIMIT = 50
const DEFAULT_MAX_RADIUS = 999999

const EventOwnerNameDelimiter = " _|_ "

const EV_MODE_CAROUSEL = "CAROUSEL"
const EV_MODE_UPCOMING = "DETAILED"
const EV_MODE_LIST = "LIST"

const (
	ES_SINGLE_EVENT  = "SLF"
	ES_EVENT_SERIES  = "EVS"
	ES_SERIES_PARENT = "SLF_EVS"
)

// NOTE: these are the default searchable event source types that show up in the home event list view
var DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES = []string{ES_SINGLE_EVENT, ES_EVENT_SERIES}

var DEFAULT_NON_SEARCHABLE_EVENT_SOURCE_TYPES = []string{ES_SERIES_PARENT}

var ALL_EVENT_SOURCE_TYPES []string

func init() {
	ALL_EVENT_SOURCE_TYPES = append(DEFAULT_SEARCHABLE_EVENT_SOURCE_TYPES, DEFAULT_NON_SEARCHABLE_EVENT_SOURCE_TYPES...)

	seen := make(map[string]bool)
	uniqueTypes := []string{}
	for _, sourceType := range ALL_EVENT_SOURCE_TYPES {
		if !seen[sourceType] {
			seen[sourceType] = true
			uniqueTypes = append(uniqueTypes, sourceType)
		}
	}

	ALL_EVENT_SOURCE_TYPES = uniqueTypes

	// Validate SitePages keys
	for key, page := range SitePages {
		if key != page.Key {
			panic(fmt.Sprintf("SitePage key mismatch: map key %q != struct key %q", key, page.Key))
		}
	}

	// Initialize the EventFields struct with field names
	eventType := reflect.TypeOf(types.Event{})
	eventFieldsValue := reflect.ValueOf(&EventFields).Elem()

	fieldDisplayNames = make(map[string]string)

	for i := 0; i < eventType.NumField(); i++ {
		field := eventType.Field(i)

		// Set the field name in EventFields
		if f := eventFieldsValue.FieldByName(field.Name); f.IsValid() {
			f.SetString(field.Name)
		}

		// Initialize the display names map
		fieldDisplayNames[field.Name] = humanizeFieldName(field.Name)
	}
}

type UserInfo struct {
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	FamilyName        string `json:"family_name"`
	GivenName         string `json:"given_name"`
	Locale            string `json:"locale"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Sub               string `json:"sub"` // This is the userID
	UpdatedAt         int    `json:"updated_at"`
	Metadata          string `json:"metadata"`
}

type PurchaseStatuses struct {
	Settled    string
	Pending    string
	Canceled   string
	Registered string
	Interested string
}

var PurchaseStatus = PurchaseStatuses{
	Settled:    "SETTLED",
	Pending:    "PENDING",
	Canceled:   "CANCELED",
	Registered: "REGISTERED",
	Interested: "INTERESTED",
}

// RoleClaim represents a formatted role claim.
type RoleClaim struct {
	Role        string `json:"role"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

type Role string

const (
	SuperAdmin Role = "superAdmin"
	OrgAdmin   Role = "orgAdmin"
)

var Roles = map[Role]string{
	SuperAdmin: string(SuperAdmin),
	OrgAdmin:   string(OrgAdmin),
}

type Category struct {
	Name, Desc, Slug string
	Items            []Subcategory
}

type SubnavOption string

const (
	NvMain    SubnavOption = "main"
	NvFilters SubnavOption = "filters"
	NvCart    SubnavOption = "cart"
)

var SubnavItems = map[SubnavOption]string{
	NvMain:    string(NvMain),
	NvFilters: string(NvFilters),
	NvCart:    string(NvCart),
}

type SitePage struct {
	Key         string
	Slug        string
	Name        string
	SubnavItems []string
}

var SitePages = map[string]SitePage{
	// NOTE: the {trailingslash:\\/?} is required for a route to match with or without a trailing slash, the
	// solution is from this github comment (see discussion as well) https://github.com/gorilla/mux/issues/30#issuecomment-1666428538
	"home":             {Key: "home", Slug: "/", Name: "Home", SubnavItems: []string{SubnavItems[NvMain], SubnavItems[NvFilters]}},
	"about":            {Key: "about", Slug: "/about{trailingslash:\\/?}", Name: "About", SubnavItems: []string{SubnavItems[NvMain]}},
	"profile":          {Key: "profile", Slug: "/admin/profile{trailingslash:\\/?}", Name: "Profile", SubnavItems: []string{SubnavItems[NvMain]}},
	"add-event-source": {Key: "add-event-source", Slug: "/admin/add-event-source{trailingslash:\\/?}", Name: "Add Event Source", SubnavItems: []string{SubnavItems[NvMain]}},
	"settings":         {Key: "settings", Slug: "/admin/profile/settings{trailingslash:\\/?}", Name: "Settings", SubnavItems: []string{SubnavItems[NvMain]}},
	"map-embed":        {Key: "map-embed", Slug: "/map-embed{trailingslash:\\/?}", Name: "MapEmbed", SubnavItems: []string{SubnavItems[NvMain]}},
	"user":             {Key: "user", Slug: "/user/{" + USER_ID_KEY + "}{trailingslash:\\/?}", Name: "User", SubnavItems: []string{SubnavItems[NvMain]}},
	"event-detail":     {Key: "event-detail", Slug: "/event/{" + EVENT_ID_KEY + "}{trailingslash:\\/?}", Name: "Event Detail", SubnavItems: []string{SubnavItems[NvMain], SubnavItems[NvCart]}},
	"add-event":        {Key: "add-event", Slug: "/admin/event/new{trailingslash:\\/?}", Name: "Add Event", SubnavItems: []string{SubnavItems[NvMain]}},
	"edit-event":       {Key: "edit-event", Slug: "/admin/event/{" + EVENT_ID_KEY + "}/edit{trailingslash:\\/?}", Name: "Edit Event", SubnavItems: []string{SubnavItems[NvMain]}},
	"attendees-event":  {Key: "attendees-event", Slug: "/admin/event/{" + EVENT_ID_KEY + "}/attendees{trailingslash:\\/?}", Name: "Event Attendees", SubnavItems: []string{SubnavItems[NvMain]}},
	"competition-new":  {Key: "competition-new", Slug: "/admin/competition/new", Name: "Add Competition", SubnavItems: []string{SubnavItems[NvMain]}},
	"competition-edit": {Key: "competition-edit", Slug: "/admin/competition/{" + COMPETITIONS_ID_KEY + "}/edit", Name: "Edit Competition", SubnavItems: []string{SubnavItems[NvMain]}},
}

// EventFields holds references to all fields in the Event struct
var EventFields struct {
	Id                    string
	EventOwners           string
	EventOwnerName        string
	EventSourceType       string
	Name                  string
	Description           string
	StartTime             string
	EndTime               string
	Address               string
	Lat                   string
	Long                  string
	EventSourceId         string
	StartingPrice         string
	Currency              string
	PayeeId               string
	HasRegistrationFields string
	HasPurchasable        string
	ImageUrl              string
	Timezone              string
	Categories            string
	Tags                  string
	CreatedAt             string
	UpdatedAt             string
	UpdatedBy             string
	RefUrl                string
	HideCrossPromo        string
	LocalizedStartDate    string
	LocalizedStartTime    string
}

var fieldDisplayNames map[string]string

// GetFieldDisplayName returns the human-readable name for a field
func GetFieldDisplayName(field string) string {
	if displayName, exists := fieldDisplayNames[field]; exists {
		return displayName
	}
	panic(fmt.Sprintf("No display name mapping for field: %s", field))
}

func humanizeFieldName(field string) string {
	switch field {
	case "Id":
		return "ID"
	case "EventOwners":
		return "Event Owners"
	case "EventOwnerName":
		return "Event Owner Name"
	case "EventSourceType":
		return "Event Source Type"
	case "Name":
		return "Name"
	case "Description":
		return "Description"
	case "StartTime":
		return "Start Date & Time"
	case "EndTime":
		return "End Date & Time"
	case "Address":
		return "Address"
	case "Lat":
		return "Latitude"
	case "Long":
		return "Longitude"
	case "EventSourceId":
		return "Event Source ID"
	case "StartingPrice":
		return "Starting Price"
	case "Currency":
		return "Currency"
	case "PayeeId":
		return "Payee ID"
	case "HasRegistrationFields":
		return "Has Registration Fields"
	case "HasPurchasable":
		return "Has Purchasable Items"
	case "ImageUrl":
		return "Image URL"
	case "Timezone":
		return "Timezone"
	case "Categories":
		return "Categories"
	case "Tags":
		return "Tags"
	case "CreatedAt":
		return "Created At"
	case "UpdatedAt":
		return "Updated At"
	case "UpdatedBy":
		return "Updated By"
	case "RefUrl":
		return "Reference URL"
	case "HideCrossPromo":
		return "Hide Cross Promotion"
	case "LocalizedStartDate":
		return "Localized Start Date"
	case "LocalizedStartTime":
		return "Localized Start Time"
	default:
		panic(fmt.Sprintf("No display name mapping for field: %s", field))
	}
}

type Subcategory struct {
	Name, Desc, Slug string
}

type CdnLocation struct {
	IATA   string  `json:"iata"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	CCA2   string  `json:"cca2"`
	Region string  `json:"region"`
	City   string  `json:"city"`
}

var Categories = []Category{
	{
		Name: "Academic & Career Development",
		Desc: "Add description later",
		Slug: "academic-career-development",
		Items: []Subcategory{
			{
				Name: "Public conferences",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Seminars",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Symposiums",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Workshops",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Training sessions",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Business",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Technology",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Health",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Academia",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Trade shows",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Expos",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Product demonstrations",
				Desc: "",
				Slug: "/",
			},
		},
	},
	{
		Name: "Arts & Community",
		Desc: "Add description later",
		Slug: "arts-community",
		Items: []Subcategory{
			{
				Name: "Art exhibitions",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Music concerts",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Dance performances",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Theater shows",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Cultural festivals",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Literary festivals",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Community gatherings",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Charity events",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Public fashion shows",
				Desc: "",
				Slug: "/",
			},
		},
	},
	{
		Name: "Civic & Advocacy",
		Desc: "Add description later",
		Slug: "civic-advocacy",
		Items: []Subcategory{
			{
				Name: "Political rallies",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Advocacy workshops",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Town hall meetings",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Civic engagement events",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Public information sessions",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Community planning meetings",
				Desc: "",
				Slug: "/",
			},
			{
				Name: "Public service",
				Desc: "",
				Slug: "/",
			},
		},
	},
	{
		Name: "Health & Wellness",
		Desc: "Add description later",
		Slug: "health-wellness",
		Items: []Subcategory{
			{
				Name: "Health fairs",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Wellness workshops",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Fitness classes",
				Desc: "",
				Slug: "",
			},
		},
	},
	{
		Name: "Kids & Families",
		Desc: "Add description later",
		Slug: "kids-families",
		Items: []Subcategory{
			{
				Name: "Age 0 - 5",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Age 5 - 8",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Age 8 - 12",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Age 12 - 15",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Age 15 - 18",
				Desc: "",
				Slug: "",
			},
		},
	},
	{
		Name: "Religious & Spiritual Gatherings",
		Desc: "Add description later",
		Slug: "religious-spiritual",
		Items: []Subcategory{
			{
				Name: "Worship services",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Public spiritual gatherings",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Religious festivals",
				Desc: "",
				Slug: "",
			},
		},
	},
	{
		Name: "Special Interests & Hobbies",
		Desc: "Add description later",
		Slug: "special-interests-hobbies",
		Items: []Subcategory{
			{
				Name: "Book clubs",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Photography walks",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Craft workshops",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Collectors' meetups",
				Desc: "",
				Slug: "",
			},
		},
	},
	{
		Name: "Sports & Outdoor Activities",
		Desc: "Add description later",
		Slug: "sports-outdoor-activities",
		Items: []Subcategory{
			{
				Name: "Sporting events",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Races, marathons, cycling",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Outdoor adventures",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Hiking",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Camping",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Skiing & Snowboarding",
				Desc: "",
				Slug: "",
			},
		},
	},
	{
		Name: "Technology & Innovation",
		Desc: "Add description later",
		Slug: "technology-innovation",
		Items: []Subcategory{
			{
				Name: "Tech meetups",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Hackathons",
				Desc: "",
				Slug: "",
			},
			{
				Name: "Innovation summits",
				Desc: "",
				Slug: "",
			},
		},
	},
}
