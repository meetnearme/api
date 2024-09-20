package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"net/http"

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
	Url              string   `json:"url" validate:"required"` // URL is the DB key in SeshuSession
	EventValidations [][]bool `json:"eventValidations" validate:"required"`
}

type SetSubdomainRequestPayload struct {
	Subdomain string `json:"subdomain" validate:"required"`
}

func SetUserSubdomain(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	var inputPayload SetSubdomainRequestPayload

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlError(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}
	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlError(w, []byte("Invalid JSON payload: "+err.Error()), http.StatusInternalServerError)
	}
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userID := userInfo.Sub

	// Call Cloudflare KV store to save the subdomain
	metadata := map[string]string{"": ""}
	err = helpers.SetCloudflareKV(inputPayload.Subdomain, userID, helpers.SUBDOMAIN_KEY, metadata)
	if err != nil {
		if err.Error() == helpers.ERR_KV_KEY_EXISTS {
			return transport.SendHtmlError(w, []byte("Subdomain already taken"), http.StatusInternalServerError)
		} else {
			return transport.SendHtmlError(w, []byte("Failed to set subdomain: "+err.Error()), http.StatusInternalServerError)
		}
	}

	var buf bytes.Buffer
	successPartial := partials.SuccessBannerHTML(`Subdomain set successfully`)

	err = successPartial.Render(r.Context(), &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GetEventsPartial(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Extract parameter values from the request query parameters
	ctx := r.Context()

	q, userLocation, radius, startTimeUnix, endTimeUnix, _ := GetSearchParamsFromReq(r)

	marqoClient, err := services.GetMarqoClient()
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get marqo client: "+err.Error()), http.StatusInternalServerError, err)
	}

	subdomainValue := r.Header.Get("X-Mnm-Subdomain-Value")

	ownerIds := []string{}
	if subdomainValue != "" {
		ownerIds = append(ownerIds, subdomainValue)
	}

	res, err := services.SearchMarqoEvents(marqoClient, q, userLocation, radius, startTimeUnix, endTimeUnix, ownerIds)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to get events via search: "+err.Error()), http.StatusInternalServerError, err)
	}

	events := res.Events

	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error getting geocoordinates: ")+err.Error()), http.StatusInternalServerError, err)
	}

	eventListPartial := pages.EventsInner(events)

	var buf bytes.Buffer
	err = eventListPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GeoLookup(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var inputPayload GeoLookupInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
	}

	err = json.Unmarshal([]byte(body), &inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendServerRes(w, []byte(string("Invalid Body: ")+err.Error()), http.StatusBadRequest, err)
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		return transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, err)
	}

	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		return transport.SendHtmlRes(w, []byte(string("Error getting geocoordinates: ")+err.Error()), http.StatusInternalServerError, err)
	}

	geoLookupPartial := partials.GeoLookup(lat, lon, address)

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func GeoThenPatchSeshuSession(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := transport.GetDB()
		geoThenPatchSeshuSessionHandler(w, r, db)
	}
}

func geoThenPatchSeshuSessionHandler(w http.ResponseWriter, r *http.Request, db internal_types.DynamoDBAPI) {
	ctx := r.Context()
	var inputPayload GeoThenSeshuPatchInputPayload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError, err)
		return
	}
	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, err)
		return
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid Body: "+err.Error()), http.StatusBadRequest, err)
		return
	}

	baseUrl := helpers.GetBaseUrlFromReq(r)

	if baseUrl == "" {
		transport.SendHtmlRes(w, []byte("Failed to get base URL from request"), http.StatusInternalServerError, err)
		return
	}

	geoService := services.GetGeoService()
	lat, lon, address, err := geoService.GetGeo(inputPayload.Location, baseUrl)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to get geocoordinates: "+err.Error()), http.StatusInternalServerError, err)
		return
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusUnprocessableEntity, err)
		return
	}

	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid latitude value"), http.StatusUnprocessableEntity, err)
		return
	}

	updateSeshuSession.LocationLatitude = latFloat
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		transport.SendHtmlRes(w, []byte("Invalid longitude value"), http.StatusUnprocessableEntity, err)
		return
	}
	updateSeshuSession.LocationLongitude = lonFloat
	updateSeshuSession.LocationAddress = address

	if updateSeshuSession.Url == "" {
		transport.SendHtmlRes(w, []byte("ERR: Invalid body: url is required"), http.StatusBadRequest, nil)
		return
	}
	geoLookupPartial := partials.GeoLookup(lat, lon, address)

	_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		transport.SendHtmlRes(w, []byte("Failed to update target URL session"), http.StatusNotFound, err)
		return
	}

	var buf bytes.Buffer
	err = geoLookupPartial.Render(ctx, &buf)
	if err != nil {
		transport.SendHtmlRes(w, []byte(err.Error()), http.StatusInternalServerError, err)
		return
	}
	transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
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
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, err)
	}

	updateSeshuSession.Url = inputPayload.Url
	// Note that only OpenAI can push events as candidates, `eventValidations` is an array of
	// arrays that confirms the subfields, but avoids a scenario where users can push string data
	// that is prone to manipulation
	updateSeshuSession.EventValidations = inputPayload.EventValidations

	seshuService := services.GetSeshuService()
	_, err = seshuService.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, err)
	}

	successPartial := partials.SuccessBannerHTML(`We've noted the events you've confirmed as accurate`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func getFieldIndices() map[string]int {
	indices := make(map[string]int)
	eventType := reflect.TypeOf(internal_types.EventInfo{})

	for i := 0; i < eventType.NumField(); i++ {
		fieldName := eventType.Field(i).Name
		switch fieldName {
		case "EventTitle", "EventLocation", "EventStartTime", "EventEndTime", "EventURL", "EventDescription":
			indices[fieldName] = i
		}
	}
	return indices
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

func getValidatedEvents(candidates []internal_types.EventInfo, validations [][]bool, hasDefaultLocation bool) []internal_types.EventInfo {
	var validatedEvents []internal_types.EventInfo
	indiceMap := getFieldIndices()

	log.Println("Indice Map: ", indiceMap)
	for i := range candidates {
		isValid := true

		if candidates[i].EventTitle == "" || !validations[i][indiceMap["EventTitle"]] || isFakeData(candidates[i].EventTitle) {
			isValid = false
		}
		if hasDefaultLocation {
			isValid = true
		} else if candidates[i].EventLocation == "" || !validations[i][indiceMap["EventLocation"]] || isFakeData(candidates[i].EventLocation) {
			isValid = false
		}
		if candidates[i].EventStartTime == "" || !validations[i][indiceMap["EventStartTime"]] || isFakeData(candidates[i].EventTitle) {
			isValid = false
		}

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

	err = json.Unmarshal([]byte(body), &inputPayload)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusInternalServerError, err)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid request body"), http.StatusBadRequest, err)
	}

	var updateSeshuSession internal_types.SeshuSessionUpdate
	err = json.Unmarshal([]byte(body), &updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Invalid JSON payload"), http.StatusBadRequest, err)
	}

	defer func() {
		var seshuSessionGet internal_types.SeshuSessionGet
		seshuSessionGet.Url = inputPayload.Url
		// TODO: this needs to use Auth
		seshuSessionGet.OwnerId = "123"
		session, err := services.GetSeshuSession(ctx, db, seshuSessionGet)

		if err != nil {
			log.Println("Failed to get SeshuSession. ID: ", session, err)
		}

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
	}()

	updateSeshuSession.Url = inputPayload.Url
	updateSeshuSession.Status = "submitted"

	_, err = services.UpdateSeshuSession(ctx, db, updateSeshuSession)

	if err != nil {
		return transport.SendHtmlRes(w, []byte("Failed to update Event Target URL session"), http.StatusBadRequest, err)
	}

	// TODO: this sets the session to `submitted`, in a follow-up PR this will call a function
	// that manages the handoff to the event scraping queue to do the real ingestion work

	successPartial := partials.SuccessBannerHTML(`Your Event Source has been added. We will put it in the queue and let you know when it's imported.`)

	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}

func UpdateUserInterests(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	r.ParseForm()
	ctx := r.Context()

	userInfo := ctx.Value("userInfo").(helpers.UserInfo)
	userID := userInfo.Sub

	categories := r.Form

	// Use a map to track unique elements
	uniqueItems := make(map[string]struct{})

	// Flatten and split by "|", then add to the map to remove duplicates
	for _, v := range categories["subCategory"] {
		parts := strings.Split(v, "|")
		for _, part := range parts {
			uniqueItems[part] = struct{}{}
		}
	}

	var flattenedCategories []string
	for item := range uniqueItems {
		flattenedCategories = append(flattenedCategories, item)
	}

	flattenedCategoriesString := strings.Join(flattenedCategories, ", ")

	err := helpers.UpdateUserMetadataKey(userID, helpers.INTERESTS_KEY, flattenedCategoriesString)
	if err != nil {
		return transport.SendHtmlError(w, []byte("Failed to save interests: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML(`Your interests have been updated successfully.`)
	var buf bytes.Buffer
	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendServerRes(w, []byte("Failed to render template: "+err.Error()), http.StatusInternalServerError, err)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, nil)
}
