package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/a-h/templ"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"

	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type GeoLookupInputPayload struct {
	Location string `json:"location" validate:"required"`
}

type GeoThenSeshuPatchInputPayload struct {
	Location string `json:"location" validate:"required"`
	Url      string `json:"url" validate:"required"` // URL is the DB key in SeshuSession
}

type SeshuSessionSubmitPayload struct {
	Url string `json:"url" validate:"required"` // URL is the DB key in SeshuSession
}

type SeshuSessionEventsPayload struct {
	Url                     string                          `json:"url" validate:"required"` // URL is the DB key in SeshuSession
	EventBoolValid          []internal_types.EventBoolValid `json:"eventValidations" validate:"required"`
	EventRecursiveBoolValid []internal_types.EventBoolValid `json:"eventValidationRecursive" validate:"omitempty"`
}

type EventDomPaths struct {
	EventTitle       string `json:"event_title_dom"`
	EventLocation    string `json:"event_location_dom"`
	EventStartTime   string `json:"event_start_time_dom"`
	EventEndTime     string `json:"event_end_time_dom"`
	EventURL         string `json:"event_url"`
	EventDescription string `json:"event_description_dom"`
}

type SetMnmOptionsRequestPayload struct {
	Subdomain    string `json:"subdomain" validate:"required"`
	PrimaryColor string `json:"primaryColor,omitempty"`
	ThemeMode    string `json:"themeMode,omitempty"`
}

type UpdateUserAboutRequestPayload struct {
	About string `json:"about" validate:"required"`
}

type eventSearchResult struct {
	events []internal_types.Event
	err    error
}

type eventParentResult struct {
	event *internal_types.Event
	err   error
}

func SetMnmOptions(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload SetMnmOptionsRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError)
	}
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userID := userInfo.Sub

	// Call Cloudflare KV store to save the subdomain
	cfMetadataValue := fmt.Sprintf(`userId=%s;--p=%s;themeMode=%s`, userID, inputPayload.PrimaryColor, inputPayload.ThemeMode)
	metadata := map[string]string{"": ""}
	err = helpers.SetCloudflareMnmOptions(inputPayload.Subdomain, userID, metadata, cfMetadataValue)
	if err != nil {
		if err.Error() == helpers.ERR_KV_KEY_EXISTS {
			return transport.SendHtmlErrorPartial([]byte("Subdomain already taken"), http.StatusInternalServerError)
		} else {
			return transport.SendHtmlErrorPartial([]byte("Failed to set subdomain: "+err.Error()), http.StatusInternalServerError)
		}
	}

	var buf bytes.Buffer
	var successPartial templ.Component
	if r.URL.Query().Has("theme") {
		successPartial = partials.SuccessBannerHTML(`Theme updated successfully`)
	} else {
		successPartial = partials.SuccessBannerHTML(`Subdomain set successfully`)
	}

	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetEventsPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()

	q, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
	}

	mnmOptions := helpers.GetMnmOptionsFromContext(ctx)
	mnmUserId := mnmOptions["userId"]

	// we override the `owners` query param here, because subdomains should always show only
	// the owner as declared authoritatively by the subdomain ID lookup in Cloudflare KV
	if mnmUserId != "" {
		ownerIds = []string{mnmUserId}
	}

	res, err := services.SearchWeaviateEvents(ctx, weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
	}

	events := res.Events
	listMode := r.URL.Query().Get("list_mode")
	// Sort events by StartTime
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime < events[j].StartTime
	})

	eventListPartial := pages.EventsInner(events, listMode)

	var buf bytes.Buffer
	err = eventListPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GetEventAdminChildrenPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()

	q, userLocation, radius, startTimeUnix, endTimeUnix, _, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds := GetSearchParamsFromReq(r)

	radius = helpers.DEFAULT_MAX_RADIUS
	farFutureTime, _ := time.Parse(time.RFC3339, "2099-01-01T00:00:00Z")
	endTimeUnix = farFutureTime.Unix()

	weaviateClient, err := services.GetWeaviateClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get weaviate client: "+err.Error()), http.StatusInternalServerError, err)
	}

	eventId := mux.Vars(r)[helpers.EVENT_ID_KEY]

	// NOTE: we want the children AND the parent event, empty string gets the parent
	eventSourceIds = []string{eventId}
	eventSourceTypes = []string{helpers.ES_EVENT_SERIES, helpers.ES_EVENT_SERIES_UNPUB}

	// Separate parent and children events
	var eventParent *internal_types.Event
	var eventChildren []internal_types.Event

	parentChan := make(chan eventParentResult)
	searchChan := make(chan eventSearchResult)

	// Launch parent event fetch in goroutine
	go func() {
		parent, err := services.GetWeaviateEventByID(ctx, weaviateClient, eventId, "")
		parentChan <- eventParentResult{event: parent, err: err}
	}()

	// Launch search in parallel
	go func() {
		res, err := services.SearchWeaviateEvents(ctx, weaviateClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds, categories, address, parseDates, eventSourceTypes, eventSourceIds)
		if err != nil {
			searchChan <- eventSearchResult{err: err}
			return
		}
		searchChan <- eventSearchResult{events: res.Events}
	}()

	// Wait for both results
	parentResult := <-parentChan
	if parentResult.err != nil {
		return transport.SendServerRes(w, []byte("Failed to get event: "+parentResult.err.Error()), http.StatusInternalServerError, parentResult.err)
	}
	eventParent = parentResult.event

	searchResult := <-searchChan
	if searchResult.err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+searchResult.err.Error()), http.StatusInternalServerError, searchResult.err)
	}
	eventChildren = searchResult.events

	// Sort eventChildren by StartTime
	sort.Slice(eventChildren, func(i, j int) bool {
		return eventChildren[i].StartTime < eventChildren[j].StartTime
	})

	eventListPartial := partials.EventAdminChildren(*eventParent, eventChildren)

	var buf bytes.Buffer
	err = eventListPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GeoLookup(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var inputPayload GeoLookupInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, "partial", err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Invalid Body: ")+err.Error()), http.StatusBadRequest, "partial", err)
	}

	var baseUrl string
	deploymentTarget := os.Getenv("DEPLOYMENT_TARGET")
	if deploymentTarget == helpers.ACT {
		baseUrl = "http://localhost:8000"
	} else {
		baseUrl = helpers.GetBaseUrlFromReq(r)
	}

	if baseUrl == "" {
		return transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, "partial", err)
	}

	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error getting geocoordinates: ")+err.Error()), http.StatusInternalServerError, "partial", err)
	}
	// Convert lat and lon to float64
	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusInternalServerError, "partial", err)
	}
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusInternalServerError, "partial", err)
	}
	geoLookupPartial := partials.GeoLookup(latFloat, lonFloat, address, "form-hidden")

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
	}
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GeoThenPatchSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := transport.GetDB()
		GeoThenPatchSeshuSessionHandler(w, r, db)
	}
}

func GeoThenPatchSeshuSessionHandler(w http.ResponseWriter, r *http.Request, db internal_types.DynamoDBAPI) {
	ctx := r.Context()
	var inputPayload GeoThenSeshuPatchInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, "partial", err)
		return
	}
	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, "partial", err)
		return
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid Body: "+err.Error()), http.StatusBadRequest, "partial", err)
		return
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, "partial", err)
		return
	}

	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to get geocoordinates: "+err.Error()), http.StatusInternalServerError, "partial", err)
		return
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, "partial", err)
		return
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusUnprocessableEntity, "partial", err)
		return
	}

	updateSeshuSession.LocationLatitude = latFloat
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusUnprocessableEntity, "partial", err)
		return
	}
	updateSeshuSession.LocationLongitude = lonFloat
	updateSeshuSession.LocationAddress = address

	if updateSeshuSession.Url == "" {
		transport.SendHtmlRes(w, []byte("ERR: Invalid body: url is required"), http.StatusBadRequest, "partial", nil)
		return
	}
	geoLookupPartial := partials.GeoLookup(latFloat, lonFloat, address, "badge")

	_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to update target URL session"), http.StatusNotFound, "partial", err)
		return
	}

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, "partial", err)
		return
	}
	transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func SubmitSeshuEvents(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	db := transport.GetDB()

	ctx := r.Context()
	var inputPayload SeshuSessionEventsPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, "partial", err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, "partial", err)
	}

	// Note that only OpenAI can push events as candidates, `eventValidations` is an array of
	// arrays that confirms the subfields, but avoids a scenario where users can push string data
	// that is prone to manipulation
	updateSeshuSession = internal_types.SeshuSessionUpdate{
		Url:              inputPayload.Url,
		EventValidations: inputPayload.EventBoolValid,
	}

	seshuService := services.GetSeshuService()
	_, err = seshuService.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
	}

	// Updating the parent and child
	if len(inputPayload.EventRecursiveBoolValid) > 0 {

		parentSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
			Url: inputPayload.Url,
		})
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get parent session"), http.StatusInternalServerError, "partial", err)
		}

		childSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
			Url: parentSession.ChildId,
		})
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to get child session"), http.StatusInternalServerError, "partial", err)
		}

		updatedCandidates := parentSession.EventCandidates
		updatedValidations := parentSession.EventValidations

		childCandidate := childSession.EventCandidates[0]
		childValidation := inputPayload.EventRecursiveBoolValid[0]

		updated := false
		for i, parentCandidate := range parentSession.EventCandidates {
			if parentCandidate.EventURL == childCandidate.EventURL {
				updatedCandidates[i] = childCandidate
				updatedValidations[i] = childValidation
				updated = true
				break
			}
		}

		if !updated {
			return transport.SendHtmlRes(w, []byte("Unable to find child session"), http.StatusInternalServerError, "partial", err)
		}

		parentUpdate := internal_types.SeshuSessionUpdate{
			Url:              parentSession.Url,
			EventCandidates:  updatedCandidates,
			EventValidations: updatedValidations,
		}

		_, err = seshuService.UpdateSeshuSession(ctx, db, parentUpdate)

		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update parent session with child data"), http.StatusBadRequest, "partial", err)
		}

		childUpdate := internal_types.SeshuSessionUpdate{
			Url:              parentSession.ChildId,
			EventValidations: []internal_types.EventBoolValid{inputPayload.EventRecursiveBoolValid[0]},
		}

		_, err = seshuService.UpdateSeshuSession(ctx, db, childUpdate)

		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
		}
	}

	successPartial := partials.SuccessBannerHTML(`We've noted the events you've confirmed as accurate`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func isFakeData(val string) bool {
	switch val {
	case services.FakeCity:
		return true
	case services.FakeUrl1:
		return true
	case services.FakeUrl2:
		return true
	case services.FakeEventTitle1:
		return true
	case services.FakeEventTitle2:
		return true
	case services.FakeStartTime1:
		return true
	case services.FakeStartTime2:
		return true
	case services.FakeEndTime1:
		return true
	case services.FakeEndTime2:
		return true
	}
	return false
}

func getValidatedEvents(candidates []internal_types.EventInfo, validations []internal_types.EventBoolValid, hasDefaultLocation bool) []internal_types.EventInfo {
	var validatedEvents []internal_types.EventInfo

	for i := range candidates {
		isValid := true

		// Validate Event Title
		if candidates[i].EventTitle == "" || isFakeData(candidates[i].EventTitle) || !validations[i].EventValidateTitle {
			isValid = false
		}

		// Validate Event Location (Only if no default location is provided)
		if !hasDefaultLocation {
			if candidates[i].EventLocation == "" || isFakeData(candidates[i].EventLocation) || !validations[i].EventValidateLocation {
				isValid = false
			}
		}

		// Validate Event Start Time
		if candidates[i].EventStartTime == "" || isFakeData(candidates[i].EventStartTime) || !validations[i].EventValidateStartTime {
			isValid = false
		}

		// If valid, add event to list
		if isValid {
			validatedEvents = append(validatedEvents, candidates[i])
		}

	}
	return validatedEvents
}

// TODO: I have no idea if this actually works or not, this is provided by ChatGPT 4o
// I'm leaving this unifinished to go work on other more urgent items, please fix
// change, or remove this as needed

// func findTextSubstring(doc *goquery.Document, substring string) (string, bool) {
// 	var path string
// 	found := false

// 	// Recursive function to traverse nodes and build the path
// 	var traverse func(*goquery.Selection, string)
// 	traverse = func(s *goquery.Selection, currentPath string) {
// 		if found {
// 			return
// 		}

// 		s.Contents().Each(func(i int, node *goquery.Selection) {
// 			if found {
// 				return
// 			}

// 			nodeText := node.Text()
// 			if strings.Contains(nodeText, substring) {
// 				path = currentPath
// 				found = true
// 				return
// 			}

// 			// Build path for the current node
// 			nodeTag := goquery.NodeName(node)
// 			// TODO: this definitely looks like incorrect LLM output
// 			nodePath := fmt.Sprintf("%s > %s:nth-child(%d)", currentPath, nodeTag, i+1)

// 			traverse(node, nodePath)
// 		})
// 	}

// 	// Start traversing from the document root
// 	traverse(doc.Selection, "html")

// 	return path, found
// }

func SubmitSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db := transport.GetDB()

	var inputPayload SeshuSessionEventsPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	var payload SeshuSessionEventsPayload
	err = json.Unmarshal([]byte(body), &payload)
	if err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	var seshuSessionGet internal_types.SeshuSessionGet
	// TODO: this needs to use Auth
	seshuSessionGet.OwnerId = "123"
	seshuSessionGet.Url = payload.Url
	seshuService := services.GetSeshuService()

	session, err := seshuService.GetSeshuSession(ctx, db, seshuSessionGet)
	if err != nil {
		log.Println("Failed to get SeshuSession. ID: ", session, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	inputPayload.EventBoolValid = session.EventValidations

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, "partial", err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, "partial", err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, "partial", err)
	}

	defer func() {
		// check for valid latitude / longitude that is NOT equal to `services.InitialEmptyLatLong`
		// which is an intentionally invalid placeholder

		hasDefaultLat := false
		latMatch, err := regexp.MatchString(services.LatitudeRegex, fmt.Sprint(session.LocationLatitude))
		if session.LocationLatitude == services.InitialEmptyLatLong {
			hasDefaultLat = false
		} else if err != nil || !latMatch {
			hasDefaultLat = true
		}

		hasDefaultLon := false
		lonMatch, err := regexp.MatchString(services.LongitudeRegex, fmt.Sprint(session.LocationLongitude))
		if session.LocationLongitude == services.InitialEmptyLatLong {
			hasDefaultLon = false
		} else if err != nil || !lonMatch || session.LocationLongitude == services.InitialEmptyLatLong {
			hasDefaultLon = true
		}

		validatedEvents := getValidatedEvents(session.EventCandidates, session.EventValidations, hasDefaultLat && hasDefaultLon)

		log.Println("Candidate Events, length: ", len(session.EventCandidates), " | events: ", session.EventCandidates)
		log.Println("Validated Events, length: ", len(validatedEvents), " | events: ", validatedEvents)

		// TODO: search `session.Html` for the items in the `validatedEvents` array
		//Loop through eventTitle

		// doc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
		// if err != nil {
		// 	print("Error for doc conversion")
		// }
		parentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
		if err != nil {
			log.Println("Error parsing parent HTML:", err)
		}

		// Optionally parse child HTML if child session exists
		var childDoc *goquery.Document
		if session.ChildId != "" {
			childSession, err := seshuService.GetSeshuSession(ctx, db, internal_types.SeshuSessionGet{
				Url: session.ChildId,
			})
			if err != nil {
				log.Println("Could not retrieve child session:", err)
			} else {
				childDoc, err = goquery.NewDocumentFromReader(strings.NewReader(childSession.Html))
				if err != nil {
					log.Println("Failed to parse child HTML:", err)
				}
			}
		}

		//URL as a key --

		// TODO: [0] is just a placeholder, should be a loop over `validatedEvents` array and search for each
		// or maybe once it finds the first one that's good enough? Walking a long array might be wasted compute
		// if the first one is good enough

		// Find the path to the text substring
		// TODO: this is commented out because it's not verified and I don't want to introduce regression,
		// uncomment this and figure out if it this approach works for traversing the DOM

		// NOTE: I think searching the DOM string for `>{ validatedEvents[0].EventTitle }<` and then backtracing
		// to get a full DOM Querystring path is a better approach because `validatedEvents[0].EventTitle` can appear
		// in HTML attributes in my testing, we want to find in where it's the opening of an HTML tag

		// doc, err := goquery.NewDocumentFromReader(strings.NewReader(session.Html))
		// if err != nil {
		// 	log.Println("Failed to parse HTML document: ", err)
		// }

		// substring := validatedEvents[0].EventTitle
		// path, found := findTextSubstring(doc, substring)
		// if found {
		// 	fmt.Printf("Text '%s' found at path: %s\n", substring, path)
		// } else {
		// 	fmt.Printf("Text '%s' not found\n", substring)
		// }

		// TODO: delete this `SeshuSession` once the handoff to the `SeshuJobs` table is complete

		// Map to hold DOM paths for each event by its title
		validationMap := make(map[string]EventDomPaths)

		for _, event := range validatedEvents {
			var docToUse *goquery.Document
			var url string
			var endTag string
			var descriptionTag string

			// Use childDoc if this event matches the last child candidate (child is always only one)
			if childDoc != nil && event.EventSource == "rs" {
				docToUse = childDoc
				url = session.ChildId
			} else {
				docToUse = parentDoc
				url = inputPayload.Url
			}

			startTag := findTagByPartialText(docToUse, event.EventStartTime)

			if event.EventEndTime == "" {
				endTag = ""
			} else {
				endTag = findTagByPartialText(docToUse, event.EventEndTime)
			}

			if event.EventDescription == "" {
				descriptionTag = ""
			} else {
				descriptionTag = findTagByPartialText(docToUse, event.EventDescription[:utf8.RuneCountInString(event.EventDescription)/2])
			}

			paths := EventDomPaths{
				EventTitle:       findTagByExactText(docToUse, event.EventTitle),
				EventLocation:    findTagByExactText(docToUse, event.EventLocation),
				EventStartTime:   startTag,
				EventEndTime:     endTag,
				EventURL:         url,
				EventDescription: descriptionTag,
			}

			validationMap[event.EventTitle] = paths

			seshuJob := internal_types.SeshuJob{
				NormalizedURLKey:         url,
				LocationLatitude:         0.0,
				LocationLongitude:        0.0,
				LocationAddress:          "",
				ScheduledScrapeTime:      0,
				TargetNameCSSPath:        findTagByExactText(docToUse, event.EventTitle),
				TargetLocationCSSPath:    findTagByExactText(docToUse, event.EventLocation),
				TargetStartTimeCSSPath:   startTag,
				TargetDescriptionCSSPath: descriptionTag, // optional
				TargetHrefCSSPath:        url,            // optional
				Status:                   "HEALTHY",      // assume healthy if parse succeeded
				LastScrapeSuccess:        time.Now().Unix(),
				LastScrapeFailure:        0,
				LastScrapeFailureCount:   0,
				OwnerID:                  "123",    // ideally from auth context
				KnownScrapeSource:        "MEETUP", // or infer from URL pattern/domain
			}

			payloadBytes, err := json.Marshal(seshuJob)
			if err != nil {
				log.Println("Error marshaling SeshuJob:", err)
				return
			}

			// POST to internal handler
			req, err := http.NewRequest(http.MethodPost, os.Getenv("SESHUJOBS_URL")+"/api/seshujob", bytes.NewReader(payloadBytes))
			if err != nil {
				log.Println("Error creating HTTP request for seshuJob:", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			res, err := client.Do(req)
			if err != nil {
				log.Println("Failed to send seshuJob to handler:", err)
				continue
			}
			defer res.Body.Close()

			if res.StatusCode >= 400 {
				body, _ := io.ReadAll(res.Body)
				log.Printf("Handler responded with status %d: %s", res.StatusCode, string(body))
			}
		}
		b, _ := json.MarshalIndent(validationMap, "", "  ")
		fmt.Println(string(b))
	}()

	updateSeshuSession.Url = inputPayload.Url
	updateSeshuSession.Status = "submitted"

	_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
	}

	if session.ChildId != "" {
		updateSeshuSession.Url = session.ChildId
		updateSeshuSession.Status = "submitted"

		_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)
		if err != nil {
			return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, "partial", err)
		}
	}

	successPartial := partials.SuccessBannerHTML(`Your Event Source has been added. We will put it in the queue and let you know when it's imported.`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func UpdateUserInterests(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	r.ParseForm()
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userID := userInfo.Sub

	// TODO: pretty sure the Form approach here makes it so that you can't submit this multiple
	// times in succession when using the profile settings page
	categories := r.Form

	// Use a map to track unique elements
	flattenedCategories := []string{}

	// Flatten and split by "|", then add to the map to remove duplicates
	for key, values := range categories {
		if strings.HasSuffix(key, "category") || strings.HasSuffix(key, "subCategory") {
			for _, value := range values {
				// Split by comma and trim spaces, in case there are multiple values
				for _, item := range strings.Split(value, ",") {
					trimmedItem := strings.TrimSpace(item)
					if trimmedItem != "" {
						flattenedCategories = append(flattenedCategories, trimmedItem)
					}
				}
			}
		}
	}

	flattenedCategoriesString := strings.Join(flattenedCategories, "|")

	err := helpers.UpdateUserMetadataKey(userID, helpers.INTERESTS_KEY, flattenedCategoriesString)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to save interests: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML(`Your interests have been updated successfully.`)
	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func UpdateUserAbout(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload UpdateUserAboutRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError)
	}
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userID := userInfo.Sub
	err = helpers.UpdateUserMetadataKey(userID, helpers.META_ABOUT_KEY, inputPayload.About)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to update 'about' field: "+err.Error()), http.StatusInternalServerError)
	}

	var buf bytes.Buffer
	successPartial := partials.SuccessBannerHTML(`About section successfully saved`)

	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func getFullDomPath(element *goquery.Selection) string {
	var path []string

	// Traverse up the parent elements
	for node := element; node.Length() > 0; node = node.Parent() {
		tag := goquery.NodeName(node)

		// Get unique identifiers (ID or first class)
		id, existsID := node.Attr("id")
		if existsID {
			path = append([]string{fmt.Sprintf("%s#%s", tag, id)}, path...)
			break // IDs are unique, stop traversal
		}

		class, existsClass := node.Attr("class")
		if existsClass {
			classes := strings.Fields(class)
			if len(classes) > 0 {
				path = append([]string{fmt.Sprintf("%s.%s", tag, classes[0])}, path...)
				continue
			}
		}

		// If no ID or class, just append the tag
		path = append([]string{tag}, path...)
	}

	return strings.Join(path, " > ") // Return full selector path
}

func findTagByExactText(doc *goquery.Document, targetText string) string {
	var exactMatch *goquery.Selection

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Get only the element's own text (exclude children)
		nodeText := strings.TrimSpace(s.Clone().Children().Remove().End().Text())

		if nodeText == targetText {
			exactMatch = s
		}
	})

	if exactMatch != nil {
		return getFullDomPath(exactMatch)
	}
	return ""
}

func findTagByPartialText(doc *goquery.Document, targetSubstring string) string {
	var bestMatch *goquery.Selection

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Normalize and preserve space between children
		text := strings.Join(s.Contents().Map(func(i int, c *goquery.Selection) string {
			return strings.TrimSpace(c.Text()) // → "Wednesday7pm"
		}), " ") // → "Wednesday 7pm"

		if strings.Contains(text, targetSubstring) {
			hasChildMatch := false

			s.Children().Each(func(i int, child *goquery.Selection) {
				childText := strings.Join(child.Contents().Map(func(i int, c *goquery.Selection) string {
					return strings.TrimSpace(c.Text())
				}), " ")
				if strings.Contains(childText, targetSubstring) {
					hasChildMatch = true
				}
			})

			if !hasChildMatch {
				bestMatch = s
			}
		}
	})

	if bestMatch != nil {
		return getFullDomPath(bestMatch)
	}
	return ""
}
