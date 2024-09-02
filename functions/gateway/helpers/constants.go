package helpers

type AWSReqKey string
const ApiGwV2ReqKey AWSReqKey = "ApiGwV2Req"

const EventsTablePrefix = "Events"
const SeshuSessionTablePrefix = "SeshuSessions"
const EVENT_ID_KEY string = "eventId"
const SUBDOMAIN_KEY = "subdomain"
const ERR_KV_KEY_EXISTS = "key already exists in KV store"

type UserInfo struct {
	Email string `json:"email"`
	EmailVerified bool `json:"email_verified"`
	FamilyName string `json:"family_name"`
	GivenName string `json:"given_name"`
	Locale string `json:"locale"`
	Name string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Sub string `json:"sub"`
	UpdatedAt int `json:"updated_at"`
	Metadata string `json:"metadata"`
}

type Category struct {
	Name, Desc, Slug string
	Items []Subcategory
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

var Categories = []Category {
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
