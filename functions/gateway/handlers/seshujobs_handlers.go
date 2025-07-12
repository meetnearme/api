package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type TriggerRequest struct {
	Time int64 `json:"time"`
}

var lastExecutionTime int64 = 0
var HOUR int64 = 3600 // 1 hour in seconds

func GetSeshuJobs(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db, _ := services.GetPostgresService(ctx)
	jobs, err := db.GetSeshuJobs(ctx)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to retrieve jobs: "+err.Error()), http.StatusInternalServerError)
	}
	buf := SeshuJobList(jobs)
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func CreateSeshuJob(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db, _ := services.GetPostgresService(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	var job internal_types.SeshuJob
	err = json.Unmarshal(body, &job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	err = db.CreateSeshuJob(ctx, job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to insert job: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML("Job created successfully")
	var buf bytes.Buffer

	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusCreated, "partial", nil)
}

func UpdateSeshuJob(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db, _ := services.GetPostgresService(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	var job internal_types.SeshuJob
	err = json.Unmarshal(body, &job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	err = db.UpdateSeshuJob(ctx, job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to update job: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML("Job updated successfully")
	var buf bytes.Buffer

	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func DeleteSeshuJob(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	db, _ := services.GetPostgresService(ctx)

	id := r.URL.Query().Get("id")
	if id == "" {
		return transport.SendHtmlErrorPartial([]byte("Missing 'id' query parameter"), http.StatusBadRequest)
	}

	err := db.DeleteSeshuJob(ctx, id)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to delete job: "+err.Error()), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML("Job deleted successfully")
	var buf bytes.Buffer

	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func GatherSeshuJobsHandler(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	var req TriggerRequest
	db, _ := services.GetPostgresService(ctx)
	nats, _ := services.GetNatsService(ctx)

	if db == nil || nats == nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to initialize services"), http.StatusInternalServerError)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	log.Printf("Received request to gather seshu jobs at time: %d", req.Time)
	log.Printf("Last execution time: %d", lastExecutionTime)

	if req.Time-lastExecutionTime <= 60 { // change this for HOUR
		return transport.SendHtmlRes(w, []byte(""), http.StatusOK, "partial", nil)
	}

	lastExecutionTime = req.Time
	currentHour := time.Unix(req.Time, 0).UTC().Hour()

	// Call NATS to look at the top of the queue for jobs
	log.Println("Checking top of NATS queue...")
	topOfQueue, err := nats.GetTopOfQueue(r.Context())

	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to get top of NATS queue:"+err.Error()), http.StatusBadRequest)
	}

	var jobs []internal_types.SeshuJob

	// If the top of the queue is not within the last 60 minutes or query is empty, do a full DB scan of seshu jobs
	if topOfQueue == nil || len(topOfQueue.Data) == 0 {

		log.Println("NATS queue is empty or top message has no data.")

		// The scan will be index scan on index seshu_jobs.scheduled_scrape_time
		jobs, err = db.ScanSeshuJobsWithInHour(r.Context(), currentHour)
		if err != nil {
			return transport.SendHtmlErrorPartial([]byte("Unable to obtain Jobs: "+err.Error()), http.StatusBadRequest)
		}
		log.Printf("Retrieved %d jobs from DB - 193 ", len(jobs))

	} else if topOfQueue != nil {

		var job internal_types.SeshuJob

		err := json.Unmarshal(topOfQueue.Data, &job)
		if err != nil {
			log.Println("Failed to unmarshal job from NATS queue:", err)
			return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
		}

		log.Printf("Top job: %+v", job)

		if currentHour-job.ScheduledHour > 0 {
			log.Println("Top job is older than 60 minutes.")

			// The scan will be index scan on index seshu_jobs.scheduled_scrape_time
			jobs, err = db.ScanSeshuJobsWithInHour(r.Context(), currentHour)
			if err != nil {
				log.Printf("DB scan failed: %v  - 201", err)
				return transport.SendHtmlErrorPartial([]byte("Unable to obtain Jobs: "+err.Error()), http.StatusBadRequest)
			}
		}
		log.Printf("Retrieved %d jobs from DB - 215", len(jobs))
	}

	// Push Found DB items onto the NATS queue
	for _, job := range jobs {
		if err := nats.PublishMsg(r.Context(), job); err != nil {
			log.Printf("Failed to push job %s to NATS: %v", job.NormalizedURLKey, err)
		}
	}

	return transport.SendHtmlRes(w, []byte("successful"), http.StatusOK, "partial", nil)

}

func SeshuJobList(jobs []internal_types.SeshuJob) *bytes.Buffer { // temporary
	var buf bytes.Buffer
	for _, job := range jobs {
		buf.WriteString(fmt.Sprintf(`
			<div class="job-card">
				<p><strong>Key:</strong> %s</p>
				<p><strong>Latitude:</strong> %f</p>
				<p><strong>Longitude:</strong> %f</p>
				<p><strong>Address:</strong> %s</p>
				<p><strong>Target Name Selector:</strong> %s</p>
				<p><strong>Target Location Selector:</strong> %s</p>
				<p><strong>Target Start Time Selector:</strong> %s</p>
				<p><strong>Target Description Selector:</strong> %s</p>
				<p><strong>Target Href Selector:</strong> %s</p>
				<p><strong>Status:</strong> %s</p>
				<p><strong>Last Scrape Success:</strong> %d</p>
				<p><strong>Last Scrape Failure:</strong> %d</p>
				<p><strong>Failure Count:</strong> %d</p>
				<p><strong>Scheduled Hour:</strong> %d</p>
				<p><strong>Owner ID:</strong> %s</p>
				<p><strong>Source:</strong> %s</p>
				<hr/>
			</div>
		`,
			job.NormalizedURLKey,
			job.LocationLatitude,
			job.LocationLongitude,
			job.LocationAddress,
			job.TargetNameCSSPath,
			job.TargetLocationCSSPath,
			job.TargetStartTimeCSSPath,
			job.TargetDescriptionCSSPath,
			job.TargetHrefCSSPath,
			job.Status,
			job.LastScrapeSuccess,
			job.LastScrapeFailure,
			job.LastScrapeFailureCount,
			job.ScheduledHour,
			job.OwnerID,
			job.KnownScrapeSource,
		))
	}
	return &buf
}
