package helpers

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

func TestParseCities(t *testing.T) {
	// Sample JSON input
	jsonInput := `[
		{
			"city": "New York",
			"growth_from_2000_to_2013": "4.8%",
			"latitude": 40.7127837,
			"longitude": -74.0059413,
			"population": "8405837",
			"rank": "1",
			"state": "New York"
		},
		{
			"city": "Los Angeles",
			"growth_from_2000_to_2013": "4.8%",
			"latitude": 34.0522342,
			"longitude": -118.2436849,
			"population": "3884307",
			"rank": "2",
			"state": "California"
		},
		{
			"city": "Chicago",
			"growth_from_2000_to_2013": "-6.1%",
			"latitude": 41.8781136,
			"longitude": -87.6297982,
			"population": "2718782",
			"rank": "3",
			"state": "Illinois"
		},
		{
			"city": "Houston",
			"growth_from_2000_to_2013": "11.0%",
			"latitude": 29.7604267,
			"longitude": -95.3698028,
			"population": "2195914",
			"rank": "4",
			"state": "Texas"
		}
	]`

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	// Call ParseCities
	ParseCities(jsonInput)

	// Get the captured output
	output := buf.String()

	// Expected output fragments
	expectedOutputs := []string{
		`City{`,
		`City:                 "New York",`,
		`Latitude:             40.712784,`,
		`Longitude:            -74.005941,`,
		`Population:           8405837,`,
		`State:                "New York",`,
		`City:                 "Los Angeles",`,
		`Latitude:             34.052234,`,
		`Longitude:            -118.243685,`,
		`Population:           3884307,`,
		`State:                "California",`,
		`City:                 "Chicago",`,
		`Latitude:             41.878114,`,
		`Longitude:            -87.629798,`,
		`Population:           2718782,`,
		`State:                "Illinois",`,
		`City:                 "Houston",`,
		`Latitude:             29.760427,`,
		`Longitude:            -95.369803,`,
		`Population:           2195914,`,
		`State:                "Texas",`,
	}

	// Check if all expected outputs are present
	for _, expected := range expectedOutputs {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't", expected)
		}
	}

	// Check if the output starts and ends correctly
	if !strings.Contains(output, "var Cities = []City{") {
		t.Error("Expected output to start with 'var Cities = []City{'")
	}
	if !strings.Contains(output, "}") {
		t.Error("Expected output to end with '}'")
	}
}
