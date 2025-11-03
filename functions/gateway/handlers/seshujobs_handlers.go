package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/helpers"
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
	db, err := services.GetPostgresService(ctx)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to initialize services: "+err.Error()), http.StatusInternalServerError)
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)
	}

	var job internal_types.SeshuJob
	err = json.Unmarshal(body, &job)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)
	}

	// Derive timezone from coordinates if available
	derivedTimezone := services.DeriveTimezoneFromCoordinates(job.LocationLatitude, job.LocationLongitude)
	if derivedTimezone != "" {
		job.LocationTimezone = derivedTimezone
	} else {
		job.LocationTimezone = ""
	}

	err = db.CreateSeshuJob(ctx, job)
	if err != nil {
		// Check if this is a duplicate key error
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return transport.SendHtmlErrorPartial([]byte("That URL is already owned by another user"), http.StatusConflict)
		}
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

	// Derive timezone from coordinates if available
	derivedTimezone := services.DeriveTimezoneFromCoordinates(job.LocationLatitude, job.LocationLongitude)
	if derivedTimezone != "" {
		job.LocationTimezone = derivedTimezone
	} else {
		job.LocationTimezone = ""
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
	userInfo := constants.UserInfo{}
	if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
		userInfo = ctx.Value("userInfo").(constants.UserInfo)
	}
	userId := userInfo.Sub
	if userId == "" {
		return transport.SendHtmlErrorPartial([]byte("Missing user ID"), http.StatusUnauthorized)
	}

	roleClaims := []constants.RoleClaim{}
	if claims, ok := ctx.Value("roleClaims").([]constants.RoleClaim); ok {
		roleClaims = claims
	}

	isSuperAdmin := helpers.HasRequiredRole(roleClaims, []string{constants.Roles[constants.SuperAdmin]})

	db, _ := services.GetPostgresService(ctx)
	id := r.URL.Query().Get("id")
	if id == "" {
		return transport.SendHtmlErrorPartial([]byte("Missing 'id' query parameter"), http.StatusBadRequest)
	}
	ctxWithTargetUrl := context.WithValue(ctx, "targetUrl", id)
	job, err := db.GetSeshuJobs(ctxWithTargetUrl)
	if err != nil || len(job) == 0 {
		return transport.SendHtmlErrorPartial([]byte("Failed to find Seshu job: "+id), http.StatusInternalServerError)
	}

	// only super admins can delete jobs that are not owned by them
	if !isSuperAdmin && job[0].OwnerID != userId {
		return transport.SendHtmlErrorPartial([]byte("You are not the owner of this job"), http.StatusForbidden)
	}

	err = db.DeleteSeshuJob(ctx, id)
	if err != nil {
		// NOTE: this should never leak error messages as they can be leveraged to know the underlying
		// database schema / structure
		return transport.SendHtmlErrorPartial([]byte("Failed to delete job "+id), http.StatusInternalServerError)
	}

	successPartial := partials.SuccessBannerHTML("Job deleted successfully")
	var buf bytes.Buffer

	err = successPartial.Render(ctx, &buf)
	if err != nil {
		return transport.SendHtmlErrorPartial([]byte("Failed to render html template"), http.StatusInternalServerError)
	}

	return transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)
}

func ProcessGatherSeshuJobs(ctx context.Context, triggerTime int64) (int, bool, int, error) {

	log.Printf("Received request to gather seshu jobs at time: %d", triggerTime)
	log.Printf("Last execution time: %d", lastExecutionTime)

	if triggerTime-lastExecutionTime <= 60 { // change this for HOUR
		return 0, true, http.StatusOK, nil
	}

	db, err := services.GetPostgresService(ctx)
	if err != nil {
		return 0, false, http.StatusInternalServerError, fmt.Errorf("failed to initialize Postgres service: %w", err)
	}
	if db == nil {
		return 0, false, http.StatusInternalServerError, fmt.Errorf("failed to initialize Postgres service")
	}

	nats, err := services.GetNatsService(ctx)
	if err != nil {
		return 0, false, http.StatusInternalServerError, fmt.Errorf("failed to initialize NATS service: %w", err)
	}
	if nats == nil {
		return 0, false, http.StatusInternalServerError, fmt.Errorf("failed to initialize NATS service")
	}

	lastExecutionTime = triggerTime
	currentHour := time.Unix(triggerTime, 0).UTC().Hour()

	topOfQueue, err := nats.PeekTopOfQueue(ctx)
	if err != nil {
		return 0, false, http.StatusBadRequest, fmt.Errorf("failed to get top of NATS queue: %w", err)
	}

	var jobs []internal_types.SeshuJob

	if topOfQueue == nil || len(topOfQueue.Data) == 0 {
		jobs, err = db.ScanSeshuJobsWithInHour(ctx, currentHour)
		if err != nil {
			return 0, false, http.StatusBadRequest, fmt.Errorf("unable to obtain Jobs: %w", err)
		}
	} else if topOfQueue.Data != nil && len(topOfQueue.Data) > 0 {
		var job internal_types.SeshuJob
		if err := json.Unmarshal(topOfQueue.Data, &job); err != nil {
			return 0, false, http.StatusBadRequest, fmt.Errorf("invalid JSON payload: %w", err)
		}

		if currentHour-job.ScheduledHour > 0 {
			jobs, err = db.ScanSeshuJobsWithInHour(ctx, currentHour)
			if err != nil {
				return 0, false, http.StatusBadRequest, fmt.Errorf("unable to obtain Jobs: %w", err)
			}
		}
	}

	published := 0
	for _, job := range jobs {
		if err := nats.PublishMsg(ctx, job); err != nil {
			jobKey := "unknown"
			if job.NormalizedUrlKey != "" {
				jobKey = job.NormalizedUrlKey
			}
			log.Printf("Failed to push job %s to NATS: %v", jobKey, err)
			continue
		}
		published++
	}

	return published, false, http.StatusOK, nil
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
				<p><strong>Target End Time Selector:</strong> %s</p>
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
			job.NormalizedUrlKey,
			job.LocationLatitude,
			job.LocationLongitude,
			job.LocationAddress,
			job.TargetNameCSSPath,
			job.TargetLocationCSSPath,
			job.TargetStartTimeCSSPath,
			job.TargetDescriptionCSSPath,
			job.TargetEndTimeCSSPath,
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

// Add these functions to expose the global variable for testing
func GetLastExecutionTime() int64 {
	return lastExecutionTime
}

func SetLastExecutionTime(t int64) {
	lastExecutionTime = t
}
