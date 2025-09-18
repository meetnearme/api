package services

import (
	"context"
	"os"
	"sync"

	"github.com/meetnearme/api/functions/gateway/interfaces"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

var (
	geoService     interfaces.GeoServiceInterface
	geoServiceOnce sync.Once

	cityService     interfaces.CityServiceInterface
	cityServiceOnce sync.Once

	seshuService     interfaces.SeshuServiceInterface
	seshuServiceOnce sync.Once
)

func GetGeoService() interfaces.GeoServiceInterface {
	geoServiceOnce.Do(func() {
		if os.Getenv("GO_ENV") == "test" {
			geoService = getMockGeoService()
		} else {
			geoService = &RealGeoService{}
		}
	})
	return geoService
}

func GetCityService() interfaces.CityServiceInterface {
	cityServiceOnce.Do(func() {
		if os.Getenv("GO_ENV") == "test" {
			cityService = getMockCityService()
		} else {
			cityService = &RealCityService{}
		}
	})
	return cityService
}

func ResetGeoService() {
	geoService = nil
	geoServiceOnce = sync.Once{}
}

func ResetCityService() {
	cityService = nil
	cityServiceOnce = sync.Once{}
}

func GetSeshuService() interfaces.SeshuServiceInterface {
	seshuServiceOnce.Do(func() {
		if os.Getenv("GO_ENV") == "test" {
			seshuService = getMockSeshuService()
		} else {
			seshuService = &RealSeshuService{}
		}
	})
	return seshuService
}

type RealGeoService struct {
	htmlFetcher HTMLFetcher // Can be nil for production use
}
type RealSeshuService struct{}

func (s *RealSeshuService) GetSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionGet) (*internal_types.SeshuSession, error) {
	return GetSeshuSession(ctx, db, seshuPayload)
}

func (s *RealSeshuService) InsertSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionInput) (*internal_types.SeshuSessionInsert, error) {
	return InsertSeshuSession(ctx, db, seshuPayload)
}

func (s *RealSeshuService) UpdateSeshuSession(ctx context.Context, db internal_types.DynamoDBAPI, seshuPayload internal_types.SeshuSessionUpdate) (*internal_types.SeshuSessionUpdate, error) {
	return UpdateSeshuSession(ctx, db, seshuPayload)
}
