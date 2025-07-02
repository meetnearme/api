package interfaces

import (
	"context"
	"errors"

	"github.com/meetnearme/api/functions/gateway/types"
)

type GeoServiceInterface interface {
	GetGeo(location, baseUrl string) (string, string, string, error)
}

type SeshuServiceInterface interface {
	GetSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionGet) (*types.SeshuSession, error)
	InsertSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionInput) (*types.SeshuSessionInsert, error)
	UpdateSeshuSession(ctx context.Context, db types.DynamoDBAPI, seshuPayload types.SeshuSessionUpdate) (*types.SeshuSessionUpdate, error)
}

type PostgresServiceInterface interface {
	GetSeshuJobs(ctx context.Context) ([]types.SeshuJob, error)
	CreateSeshuJob(ctx context.Context, job types.SeshuJob) error
	UpdateSeshuJob(ctx context.Context, job types.SeshuJob) error
	DeleteSeshuJob(ctx context.Context, id string) error
}

var ErrInvalidLocation = errors.New("location is not valid")
