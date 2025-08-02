package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/services"
	partials "github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"
	"github.com/meetnearme/api/functions/gateway/types"
)

// TODO: make sure all of these edge cases are covered for / explained / documented
// 1. OpenAI can respond with `[{ event_title: "My event", "event_location": "My locat... length}]` which is invalid JSON
// 2. Timeout for ZR can fail to fetch in time... retry? or just return an error?
// 3. Add special logic for meetup.com URLs to extract event info from <script type="application/ld+json">...</script>
// 4. Add special logic for eventbrite.com URLs to extract event info from <script type="application/ld+json">...</script>
// 5. Handle the "read more" / scroll down behavior for meetup.com / eventbrite via ZR remote JS
// 6. Handle / validate that problem with scraping API Key properly aborts
// 7. Check token consumption on OpenAI response (if present) and fork session if nearing limit
// 8. Validate that the invalid metadata strings fromt the example (e.g. `Nowhere City`) do not appear in the LLM event array output
// 9. Open AI can sometimes truncate the attempted JSON like this `<valid JSON start>...events/300401920/"}, {"event_tit...} stop}]`
// 10. Validate user input is really a URL prior to consuming API resources like ZR or OpenAI
// 11. Loop over OpenAI response and remove any that have no chance of validity (e.g. lack a time or event title) before sending to the client
// 12. Check for "Fake Event Title 1" (and the same for `location`, `time`, and `url`) and null those values before sending to client
// 13. Was discovered that google.com/events requires a "premium proxy" (20 scrape credits instead of 5)
//     so we want to create a "deny list" of sites we don't support and respond with an API error to let users know
// 14. For Meetup.com URLs, we want to ensure the query param `&eventType=inPerson` is present to avoid online events
// 15. Handle the scenario below where the scraped Markdown data is so large, that it exceeds the OpenAI API limit
//    and results in the error `Error: unexpected response format, `id` missing` because OpenAI literally returns an empty
//    Chat GPT response:  {  0  [] map[]}

// 		[markdown response from ZR]...nited KingdomUnited States\"]"}]}} 0x1288b40 50405 [] false api.openai.com map[] map[] <nil> map[]   <nil> <nil> <nil> {{}}}
// 		[sst] |  +10807ms Chat GPT response:  {  0  [] map[]}
// 		[sst] |  +10807ms Error creating chat session: unexpected response format, `id` missing
// 		[sst] |  +10808ms 2024/04/26 15:07:55 {"errorMessage":"unexpected response format, `id` missing","errorType":"errorString"}
// 		[sst] |  Error: unexpected response format, `id` missing

// 395KB is just a bit under the 400KB dynamoDB limit
// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/bp-use-s3-too.html
const maxHtmlDocSize = 395 * 1024

type InternalRequest struct {
	Method  string
	Action  string
	Body    string
	Headers map[string]string
}

type InternalResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
}

type SeshuInputPayload struct {
	Url string `json:"url" validate:"required"`
}

type SeshuRecursivePayload struct {
	ParentUrl string `json:"parent_url" validate:"required"`
	Url       string `json:"url" validate:"required"`
}

type SeshuResponseBody struct {
	SessionID   string            `json:"session_id"`
	EventsFound []types.EventInfo `json:"events_found"`
}

var db types.DynamoDBAPI
var scrapingService services.ScrapingService

func init() {
	db = transport.CreateDbClient()
	scrapingService = &services.RealScrapingService{}
}

func HandleSeshuJobSubmit(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	// Handle only POST
	if r.Method != http.MethodPost {
		return transport.SendHtmlErrorPartial([]byte("Method Not Allowed"), http.StatusMethodNotAllowed)
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read body: "+err.Error()), http.StatusBadRequest)
	}
	defer r.Body.Close()

	action := r.URL.Query().Get("action")

	urlToScrape, parentUrl, childID, err := parsePayload(action, string(body))
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte(err.Error()), http.StatusUnprocessableEntity)
	}

	var events []types.EventInfo
	events, htmlContent, err := services.ExtractEventsFromHTML(urlToScrape, action, scrapingService)
	if err != nil {
		log.Println("Event extraction error:", err)
		return transport.SendHtmlErrorPartial([]byte(err.Error()), http.StatusInternalServerError)
	}

	ctx := r.Context()

	defer saveSession(ctx, htmlContent, urlToScrape, childID, parentUrl, events, action, scrapingService)

	tmpl := partials.EventCandidatesPartial(events)
	var buf bytes.Buffer
	if err := tmpl.Render(ctx, &buf); err != nil {
		return transport.SendHtmlErrorPartial([]byte(err.Error()), http.StatusInternalServerError)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func parsePayload(action string, body string) (urlToScrape, parentUrl, childID string, err error) {
	switch action {
	case "init":
		var payload SeshuInputPayload
		err = parseAndValidatePayload(body, &payload)
		return payload.Url, "", "", err
	case "rs":
		var payload SeshuRecursivePayload
		err = parseAndValidatePayload(body, &payload)
		return payload.Url, payload.ParentUrl, "", err
	default:
		return "", "", "", errors.New("invalid action")
	}
}

func saveSession(ctx context.Context, htmlContent string, urlToScrape, childID, parentUrl string, events []types.EventInfo, action string, scraper services.ScrapingService) {
	if len(events) == 0 {
		return
	}

	url, err := url.Parse(urlToScrape)
	if err != nil {
		log.Println("ERR: Error parsing URL:", err)
	}

	truncatedHTMLStr, exceededLimit := helpers.TruncateStringByBytes(htmlContent, maxHtmlDocSize)
	if exceededLimit {
		log.Printf("WARN: HTML document exceeded %v byte limit, truncating", maxHtmlDocSize)
	}

	if action == "rs" {
		events[0].EventURL = urlToScrape
	}

	now := time.Now()
	payload := types.SeshuSessionInput{
		SeshuSession: types.SeshuSession{
			OwnerId:           "123",
			Url:               urlToScrape,
			UrlDomain:         url.Host,
			UrlPath:           url.Path,
			UrlQueryParams:    url.Query(),
			Html:              truncatedHTMLStr,
			ChildId:           childID,
			LocationLatitude:  services.InitialEmptyLatLong,
			LocationLongitude: services.InitialEmptyLatLong,
			EventCandidates:   events,
			CreatedAt:         now.Unix(),
			UpdatedAt:         now.Unix(),
			ExpireAt:          now.Add(24 * time.Hour).Unix(),
		},
	}

	if _, err := services.InsertSeshuSession(ctx, db, payload); err != nil {
		log.Println("Error inserting Seshu session:", err)
	}

	if action == "rs" {
		if parsedParentUrl, err := url.Parse(parentUrl); err == nil {
			_, err := services.UpdateSeshuSession(ctx, db, types.SeshuSessionUpdate{
				Url:     parsedParentUrl.String(),
				ChildId: url.String(),
			})
			if err != nil {
				log.Println("ERR: failed to update parent session with childID:", err)
			}
		}
	}
}

func _SendHtmlErrorPartial(err error, ctx context.Context) (InternalResponse, error) {
	layoutTemplate := partials.ErrorHTML(err, "web-handler")
	var buf bytes.Buffer
	renderErr := layoutTemplate.Render(ctx, &buf)
	if renderErr != nil {
		return serverError(renderErr)
	}

	return InternalResponse{
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
		Body:       buf.String(),
	}, nil
}

func SendMessage(sessionID string, message string) (string, error) {
	client := &http.Client{}

	payload := services.SendMessagePayload{
		Messages: []services.Message{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf(os.Getenv("OPENAI_API_BASE_URL")+"/chat/completions/%s/messages", sessionID), bytes.NewBuffer(payloadBytes))

	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil

}

// TODO: this should share with the gateway handler, though the
// function signature typing is different
func clientError(status int) (InternalResponse, error) {
	return InternalResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

// // TODO: this should share with the gateway handler

func serverError(err error) (InternalResponse, error) {
	log.Println(err.Error())
	return InternalResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func parseAndValidatePayload(payloadBody string, payload any) error {
	if err := json.Unmarshal([]byte(payloadBody), payload); err != nil {
		log.Printf("Invalid JSON payload: %v", err)
		return fmt.Errorf("unprocessable")
	}
	if err := validate.Struct(payload); err != nil {
		log.Printf("Invalid payload struct: %v", err)
		return fmt.Errorf("badrequest")
	}
	return nil
}
