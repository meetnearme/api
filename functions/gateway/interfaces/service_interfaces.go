package interfaces

import (
	"context"
	"errors"

	"github.com/meetnearme/api/functions/gateway/types"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stripe/stripe-go/v83"
)

type GeoServiceInterface interface {
	GetGeo(location, baseUrl string) (string, string, string, error)
}

type CityServiceInterface interface {
	GetCity(location string) (string, error)
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
	ScanSeshuJobsWithInHour(ctx context.Context, hours int) ([]types.SeshuJob, error)
	Close() error
}

type NatsServiceInterface interface {
	PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error)
	PublishMsg(ctx context.Context, job interface{}) error
	ConsumeMsg(ctx context.Context, workers int) error
	Close() error
}

type StripeSubscriptionServiceInterface interface {
	GetSubscriptionPlans() ([]*types.SubscriptionPlan, error)
	GetCustomerSubscriptions(customerID string) ([]*types.CustomerSubscription, error)
	CreateCustomerPortalSession(customerID, returnURL string, subscriptionID, flowType string) (*types.CustomerPortalSession, error)
	SearchCustomerByExternalID(externalID string) (*stripe.Customer, error)
	UpdateCustomerMetadata(customerID, externalID string) error
	CreateCustomer(externalID, email, name string) (*stripe.Customer, error)
	GetOrCreateCustomerByExternalID(externalID, email, name string) (*stripe.Customer, error)
}

var ErrInvalidLocation = errors.New("location is not valid")
