package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/meetnearme/api/functions/gateway/services"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/transport"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type TriggerRequest struct {
	Time int64 `json:"time"`
}

var lastExecutionTime int64 = 0

func GetSeshuJobs(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	service := services.GetPostgresService(ctx)

	jobs, err := service.GetSeshuJobs(ctx)

	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to retrieve jobs: "+err.Error()), http.StatusInternalServerError)
	}

	buf := SeshuJobList(jobs)
	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func CreateSeshuJob(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	ctx := r.Context()
	service := services.GetPostgresService(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	var job internal_types.SeshuJob
	err = json.Unmarshal(body, &job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	err = service.CreateSeshuJob(ctx, job)
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
	service := services.GetPostgresService(ctx)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	var job internal_types.SeshuJob
	err = json.Unmarshal(body, &job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	err = service.UpdateSeshuJob(ctx, job)
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
	service := services.GetPostgresService(ctx)

	id := r.URL.Query().Get("id")
	if id == "" {
		return transport.SendHtmlErrorPartial([]byte("Missing 'id' query parameter"), http.StatusBadRequest)
	}

	err := service.DeleteSeshuJob(ctx, id)
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

	var req TriggerRequest

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

	if req.Time-lastExecutionTime <= 120 {
		return transport.SendHtmlRes(w, []byte(""), http.StatusOK, "partial", nil)
	}

	lastExecutionTime = req.Time

	// Call NATS to look at the top of the queue for jobs

	// If the top of the queue is not within the last 60 seconds or query is empty, do a full DB scan of seshu jobs

	// The scan will be index scan on index seshu_jobs.scheduled_scrape_time

	// Push Found DB items onto the NATS queue

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
				<p><strong>Scheduled Scrape Time:</strong> %d</p>
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
			job.ScheduledScrapeTime,
			job.OwnerID,
			job.KnownScrapeSource,
		))
	}
	return &buf
}
