package services

import (
	"github.com/meetnearme/api/functions/gateway/interfaces"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

var getMockGeoService = func() interfaces.GeoServiceInterface {
    return  &test_helpers.MockGeoService{}
}

func getMockSeshuService() interfaces.SeshuServiceInterface {
    return &test_helpers.MockSeshuService{}
}
