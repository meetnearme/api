package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

type PostgresService struct {
	DB *pgxpool.Pool
}

func NewPostgresService(db *pgxpool.Pool) *PostgresService {
	return &PostgresService{DB: db}
}

func GetPostgresClient(ctx context.Context) (*pgxpool.Pool, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return nil, fmt.Errorf("missing required environment variables for Postgres")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	return pgxpool.New(ctx, dsn)
}

func (s *PostgresService) GetSeshuJobs(ctx context.Context) ([]internal_types.SeshuJob, error) {
	rows, err := s.DB.Query(ctx, "SELECT * FROM seshujobs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []internal_types.SeshuJob
	for rows.Next() {
		var (
			scheduledScrapeTime time.Time
			lastScrapeSuccess   *time.Time
			lastScrapeFailure   *time.Time
		)

		var job internal_types.SeshuJob
		err := rows.Scan(
			&job.NormalizedURLKey, &job.LocationLatitude, &job.LocationLongitude, &job.LocationAddress,
			&scheduledScrapeTime, &job.TargetNameCSSPath, &job.TargetLocationCSSPath,
			&job.TargetStartTimeCSSPath, &job.TargetDescriptionCSSPath, &job.TargetHrefCSSPath,
			&job.Status, &lastScrapeSuccess, &lastScrapeFailure, &job.LastScrapeFailureCount,
			&job.OwnerID, &job.KnownScrapeSource,
		)
		if err != nil {
			return nil, err
		}

		job.ScheduledScrapeTime = scheduledScrapeTime.Unix()
		if lastScrapeSuccess != nil {
			job.LastScrapeSuccess = lastScrapeSuccess.Unix()
		}
		if lastScrapeFailure != nil {
			job.LastScrapeFailure = lastScrapeFailure.Unix()
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (s *PostgresService) CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	_, err := s.DB.Exec(ctx, `
		INSERT INTO seshujobs (
			normalized_url_key, location_latitude, location_longitude, location_address,
			scheduled_scrape_time, target_name_css_path, target_location_css_path,
			target_start_time_css_path, target_description_css_path, target_href_css_path,
			status, last_scrape_success, last_scrape_failure, last_scrape_failure_count,
			owner_id, known_scrape_source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15, $16
		)`,
		job.NormalizedURLKey, job.LocationLatitude, job.LocationLongitude, job.LocationAddress,
		time.Unix(job.ScheduledScrapeTime, 0),
		job.TargetNameCSSPath, job.TargetLocationCSSPath, job.TargetStartTimeCSSPath,
		job.TargetDescriptionCSSPath, job.TargetHrefCSSPath,
		job.Status,
		time.Unix(job.LastScrapeSuccess, 0),
		time.Unix(job.LastScrapeFailure, 0),
		job.LastScrapeFailureCount, job.OwnerID, job.KnownScrapeSource,
	)
	return err
}

func (s *PostgresService) UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	_, err := s.DB.Exec(ctx, `
		UPDATE seshujobs SET
			location_latitude = $1,
			location_longitude = $2,
			location_address = $3,
			scheduled_scrape_time = $4,
			target_name_css_path = $5,
			target_location_css_path = $6,
			target_start_time_css_path = $7,
			target_description_css_path = $8,
			target_href_css_path = $9,
			status = $10,
			last_scrape_success = $11,
			last_scrape_failure = $12,
			last_scrape_failure_count = $13,
			owner_id = $14,
			known_scrape_source = $15
		WHERE normalized_url_key = $16
	`,
		job.LocationLatitude, job.LocationLongitude, job.LocationAddress,
		time.Unix(job.ScheduledScrapeTime, 0), job.TargetNameCSSPath, job.TargetLocationCSSPath,
		job.TargetStartTimeCSSPath, job.TargetDescriptionCSSPath, job.TargetHrefCSSPath,
		job.Status, time.Unix(job.LastScrapeSuccess, 0), time.Unix(job.LastScrapeFailure, 0),
		job.LastScrapeFailureCount, job.OwnerID, job.KnownScrapeSource, job.NormalizedURLKey,
	)
	return err
}

func (s *PostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	_, err := s.DB.Exec(ctx, "DELETE FROM seshujobs WHERE normalized_url_key = $1", id)
	return err
}
