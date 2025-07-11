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

func GetSeshuJobs(db *services.PostgresService) func(http.ResponseWriter, *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			jobs, err := db.GetSeshuJobs(ctx)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to retrieve jobs: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			buf := SeshuJobList(jobs)
			transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)(w, r)
		}
	}
}

func CreateSeshuJob(db *services.PostgresService) func(http.ResponseWriter, *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			var job internal_types.SeshuJob
			err = json.Unmarshal(body, &job)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)(w, r)
				return
			}

			err = db.CreateSeshuJob(ctx, job)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to insert job: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			successPartial := partials.SuccessBannerHTML("Job created successfully")
			var buf bytes.Buffer
			err = successPartial.Render(ctx, &buf)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			transport.SendHtmlRes(w, buf.Bytes(), http.StatusCreated, "partial", nil)(w, r)
		}
	}
}

func UpdateSeshuJob(db *services.PostgresService) func(http.ResponseWriter, *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			body, err := io.ReadAll(r.Body)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			var job internal_types.SeshuJob
			err = json.Unmarshal(body, &job)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)(w, r)
				return
			}

			err = db.UpdateSeshuJob(ctx, job)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to update job: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			successPartial := partials.SuccessBannerHTML("Job updated successfully")
			var buf bytes.Buffer
			err = successPartial.Render(ctx, &buf)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)(w, r)
		}
	}
}

func DeleteSeshuJob(db *services.PostgresService) func(http.ResponseWriter, *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			id := r.URL.Query().Get("id")
			if id == "" {
				transport.SendHtmlErrorPartial([]byte("Missing 'id' query parameter"), http.StatusBadRequest)(w, r)
				return
			}

			err := db.DeleteSeshuJob(ctx, id)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to delete job: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			successPartial := partials.SuccessBannerHTML("Job deleted successfully")
			var buf bytes.Buffer
			err = successPartial.Render(ctx, &buf)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to render template: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			transport.SendHtmlRes(w, buf.Bytes(), http.StatusOK, "partial", nil)(w, r)
		}
	}
}

func GatherSeshuJobsHandler(db *services.PostgresService, nats *services.NatsService) func(http.ResponseWriter, *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			var req TriggerRequest

			body, err := io.ReadAll(r.Body)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Failed to read request body: "+err.Error()), http.StatusInternalServerError)(w, r)
				return
			}

			err = json.Unmarshal(body, &req)
			if err != nil {
				transport.SendHtmlErrorPartial([]byte("Invalid JSON payload: "+err.Error()), http.StatusBadRequest)(w, r)
				return
			}

			log.Printf("Received request to gather seshu jobs at time: %d", req.Time)
			log.Printf("Last execution time: %d", lastExecutionTime)

			if req.Time-lastExecutionTime <= HOUR {
				transport.SendHtmlRes(w, []byte(""), http.StatusOK, "partial", nil)(w, r)
				return
			}

			lastExecutionTime = req.Time
			currentHour := time.Unix(req.Time, 0).UTC().Hour()

			// Call NATS to look at the top of the queue for jobs
			log.Println("Checking top of NATS queue...")
			topOfQueue, err := nats.GetTopOfQueue(r.Context())

			if err != nil {
				log.Printf("Failed to get top of NATS queue: %v", err)
			}

			var jobs []internal_types.SeshuJob

			// If the top of the queue is not within the last 60 minutes or query is empty, do a full DB scan of seshu jobs
			if topOfQueue == nil || len(topOfQueue.Data) == 0 {

				log.Println("NATS queue is empty or top message has no data.")

				// The scan will be index scan on index seshu_jobs.scheduled_scrape_time
				jobs, err = db.ScanSeshuJobsWithInHour(r.Context(), currentHour)
				if err != nil {
					log.Printf("DB scan failed: %v", err)
					transport.SendHtmlErrorPartial([]byte("Unable to obtain Jobs: "+err.Error()), http.StatusBadRequest)(w, r)
					return
				}
				log.Printf("Retrieved %d jobs from DB - 193 ", len(jobs))
			} else if topOfQueue != nil {

				log.Println("Found job at top of NATS queue.")
				var job internal_types.SeshuJob

				err := json.Unmarshal(topOfQueue.Data, &job)
				if err != nil {
					log.Println("Failed to unmarshal job from NATS queue:", err)
				}

				log.Printf("Top job: %+v", job)

				if currentHour-job.ScheduledHour > 0 {
					log.Println("Top job is older than 60 minutes.")

					// The scan will be index scan on index seshu_jobs.scheduled_scrape_time
					jobs, err = db.ScanSeshuJobsWithInHour(r.Context(), currentHour)
					if err != nil {
						log.Printf("DB scan failed: %v  - 201", err)
						transport.SendHtmlErrorPartial([]byte("Unable to obtain Jobs: "+err.Error()), http.StatusBadRequest)(w, r)
						return
					}
				}
				log.Printf("Retrieved %d jobs from DB - 215", len(jobs))
			}

			// Push Found DB items onto the NATS queue
			for _, job := range jobs {
				log.Printf("Pushing job %s to NATS...", job.NormalizedURLKey)
				if err := nats.PublishMsg(r.Context(), job); err != nil {
					log.Printf("Failed to push job %s to NATS: %v", job.NormalizedURLKey, err)
				}
			}

			transport.SendHtmlRes(w, []byte("successful"), http.StatusOK, "partial", nil)(w, r)
		}
	}
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
