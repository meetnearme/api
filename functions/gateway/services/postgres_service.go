package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresService struct {
	DB *gorm.DB
}

func NewPostgresService(db *gorm.DB) *PostgresService {
	return &PostgresService{DB: db}
}

func GetPostgresClient(ctx context.Context) (*gorm.DB, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return nil, fmt.Errorf("missing required environment variables for Postgres")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, dbname)
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return gormDB, nil
}

func (s *PostgresService) GetSeshuJobs(ctx context.Context) ([]internal_types.SeshuJob, error) {
	userInfo := ctx.Value("userInfo").(constants.UserInfo)
	userId := userInfo.Sub

	targetUrl := ctx.Value("targetUrl").(string)

	var jobs []internal_types.SeshuJob
	query := s.DB.WithContext(ctx).Model(&internal_types.SeshuJob{})

	if userId != "" {
		query = query.Where("owner_id = ?", userId)
	}

	if targetUrl != "" {
		query = query.Where("normalized_url_key = ?", targetUrl)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *PostgresService) CreateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	return s.DB.WithContext(ctx).Create(&job).Error
}

func (s *PostgresService) UpdateSeshuJob(ctx context.Context, job internal_types.SeshuJob) error {
	return s.DB.WithContext(ctx).
		Model(&internal_types.SeshuJob{}).
		Where("normalized_url_key = ?", job.NormalizedUrlKey).
		Select("*").
		Updates(job).
		Error
}

func (s *PostgresService) DeleteSeshuJob(ctx context.Context, id string) error {
	return s.DB.WithContext(ctx).
		Where("normalized_url_key = ?", id).
		Delete(&internal_types.SeshuJob{}).
		Error
}

func (s *PostgresService) ScanSeshuJobsWithInHour(ctx context.Context, currentHour int) ([]internal_types.SeshuJob, error) {
	var jobs []internal_types.SeshuJob
	if err := s.DB.WithContext(ctx).
		Where("scheduled_hour = ?", currentHour).
		Find(&jobs).
		Error; err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *PostgresService) Close() error {
	if s.DB != nil {
		sqlDB, err := s.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
