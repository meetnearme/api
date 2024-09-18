package rds_handlers

import (
	"log"
	"os"
	"testing"

	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
	"github.com/meetnearme/api/functions/gateway/transport"
	// rds_types "github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
)

func TestMain(m *testing.M) {
	log.Println("Setting up test environment for rds handlers package")

	// Set GO_ENV to "test" to trigger test-specific behavior
	os.Setenv("GO_ENV", helpers.GO_TEST_ENV)

	mockRds := test_helpers.NewMockRdsDataClientWithJSONRecords([]map[string]interface{}{
		{
			"id":          "123",
			"name":        "Test Event",
			"description": "This is a test event",
			"datetime":    "2023-05-01T12:00:00Z",
			"address":     "123 Test St",
			"zip_code":    "12345",
			"country":     "Test Country",
			"lat":         "51.5074",
			"long":        "-0.1278",
		},
	})

	transport.SetTestRdsDB(mockRds)

	log.Println("Running tests for handlers package")
	exitCode := m.Run()

	log.Println("Cleaning up test environment for handlers package")
	// Perform any necessary cleanup here

	os.Exit(exitCode)
}

