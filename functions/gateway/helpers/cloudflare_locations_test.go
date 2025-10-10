package helpers

import (
	"testing"

	"github.com/meetnearme/api/functions/gateway/constants"
)

func TestCfLocationMap(t *testing.T) {
	expectedMap := map[string]constants.CdnLocation{
		"TIA": {IATA: "TIA", Lat: 41.4146995544, Lon: 19.7206001282, CCA2: "AL", Region: "Europe", City: "Tirana"},
		"ALG": {IATA: "ALG", Lat: 36.6910018921, Lon: 3.2154099941, CCA2: "DZ", Region: "Africa", City: "Algiers"},
	}

	for key, expectedLocation := range expectedMap {
		actualLocation, exists := CfLocationMap[key]
		if !exists {
			t.Errorf("Expected location with IATA %s not found in map", key)
			continue
		}

		if actualLocation != expectedLocation {
			t.Errorf("For IATA %s, expected location %v, got %v", key, expectedLocation, actualLocation)
		}
	}
}
