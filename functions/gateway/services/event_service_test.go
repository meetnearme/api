package services

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ganeshdipdumbare/marqo-go"
	"github.com/meetnearme/api/functions/gateway/helpers"
)

func init () {
    os.Setenv("GO_ENV", helpers.GO_TEST_ENV)
}

func TestUpsertEventToMarqo(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := "http://localhost:8999"
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("x-api-key")
		expectedAuthHeader := testMarqoApiKey
		if authHeader != expectedAuthHeader {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Mock the response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	// Test data
	eventTime := time.Date(2030, 5, 1, 12, 0, 0, 0, time.UTC)
	createEvent := Event{
		Name:        "New Event",
		Description: "New Description",
		StartTime:   eventTime.Format(time.RFC3339),
		Address:     "New Address",
		Lat:         float64(51.5074),
		Long:        float64(-0.1278),
	}

	// Create Marqo client and call the function
	client, err := marqo.NewClient()
	if err != nil {
		t.Fatalf("Failed to create Marqo client: %v", err)
	}
	newEvent, err := UpsertEventToMarqo(client, createEvent)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if newEvent == nil {
		t.Fatal("Expected newEvent to be non-nil")
	}
}
