package helpers

type AWSReqKey string

const ApiGwV2ReqKey AWSReqKey = "ApiGwV2Req"


const RsvpsTablePrefix = "EventRsvps"
const PurchasesTablePrefix = "Purchases"
const PurchasablesTablePrefix = "Purchasables"
const SeshuSessionTablePrefix = "SeshuSessions"
const RegistrationsTablePrefix = "Registrations"
const RegistrationFieldsTablePrefix = "RegistrationFields"
const EVENT_ID_KEY string = "eventId"
const SUBDOMAIN_KEY = "subdomain"
const INTERESTS_KEY = "interests"
const ERR_KV_KEY_EXISTS = "key already exists in KV store"
const GO_TEST_ENV = "test"

const MOCK_CLOUDFLARE_URL = "http://localhost:8999"
const MOCK_ZITADEL_HOST = "localhost:8998"
const MOCK_MARQO_URL = "http://localhost:8997"

const JWT_ASSERTION_TYPE = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

const PROJECT_ID_ROLE_CLAIMS_KEY = "urn:zitadel:iam:org:project:<project-id>:roles"

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

// RoleClaim represents a formatted role claim.
type RoleClaim struct {
	Role        string `json:"role"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
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
	Slug        string
	Name        string
	SubnavItems []string
}

var SitePages = map[string]SitePage{
	"home":             {Slug: "home", Name: "Home", SubnavItems: []string{SubnavItems[NvMain], SubnavItems[NvFilters]}},
	"about":            {Slug: "about", Name: "About", SubnavItems: []string{SubnavItems[NvMain]}},
	"profile":          {Slug: "admin/profile", Name: "Profile", SubnavItems: []string{SubnavItems[NvMain]}},
	"add-event-source": {Slug: "admin/add-event-source", Name: "Add Event Source", SubnavItems: []string{SubnavItems[NvMain]}},
	"settings":         {Slug: "settings", Name: "Settings", SubnavItems: []string{SubnavItems[NvMain]}},
	"embed":            {Slug: "embed", Name: "Embed", SubnavItems: []string{SubnavItems[NvMain]}},
	"events":           {Slug: "events", Name: "Event Details", SubnavItems: []string{SubnavItems[NvMain], SubnavItems[NvCart]}},
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
