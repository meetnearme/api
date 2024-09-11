package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
)

func TestGetHomePage(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

	// Set test environment variables
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", getNextPort())
	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

	testMarqoApiKey := "test-marqo-api-key"
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"Hits": []map[string]interface{}{
				{
					"id":          "123",
					"eventOwners": []interface{}{"789"},
					"name":        "First Test Event",
					"description": "Description of the first event",
				},
				{
					"id":          "456",
					"eventOwners": []interface{}{"012"},
					"name":        "Second Test Event",
					"description": "Description of the second event",
				},
			},
		}
		responseBytes, err := json.Marshal(response)

		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()

	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
		t.Log("Started mock Marqo server")
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()


	// Create a request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
			t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := GetHomePage(rr, req)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body (you might want to add more specific checks)
	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}

	if !strings.Contains(rr.Body.String(), ">First Test Event") {
		t.Errorf("First event title is missing from the page")
	}

	if !strings.Contains(rr.Body.String(), ">Second Test Event") {
		t.Errorf("First event title is missing from the page")
	}
}

func TestGetHomePageWithCFLocationHeaders(t *testing.T) {

	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

	// Set test environment variables
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", getNextPort())
	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

	testMarqoApiKey := "test-marqo-api-key"
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"Hits": []map[string]interface{}{
				{
					"id":          "123",
					"eventOwners": []interface{}{"789"},
					"name":        "First Test Event",
					"description": "Description of the first event",
				},
				{
					"id":          "456",
					"eventOwners": []interface{}{"012"},
					"name":        "Second Test Event",
					"description": "Description of the second event",
				},
			},
		}
		responseBytes, err := json.Marshal(response)

		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
		t.Log("Started mock Marqo server")
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()


		// Create a request
    req, err := http.NewRequest("GET", "/", nil)
    if err != nil {
        t.Fatal(err)
    }

		// Set up context with APIGatewayV2HTTPRequest
		ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
			Headers: map[string]string{"cf-ray": "8aebbd939a781f45-DEN"},
		})

		req = req.WithContext(ctx)

    // Create a ResponseRecorder to record the response
    rr := httptest.NewRecorder()

    // Call the handler
    handler := GetHomePage(rr, req)
    handler.ServeHTTP(rr, req)

    // Check the status code
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    // Check the response body (you might want to add more specific checks)
    if rr.Body.String() == "" {
        t.Errorf("Handler returned empty body")
    }
}

func TestGetLoginPage(t *testing.T) {
	req, err := http.NewRequest("GET", "/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := GetLoginPage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}

func TestGetProfilePage(t *testing.T) {
	req, err := http.NewRequest("GET", "/profile", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := GetProfilePage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}

func TestGetMapEmbedPage(t *testing.T) {
	req, err := http.NewRequest("GET", "/map-embed?address=New York", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up context with APIGatewayV2HTTPRequest
	ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		QueryStringParameters: map[string]string{"address": "New York"},
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := GetMapEmbedPage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}


func TestGetEventDetailsPage(t *testing.T) {

	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("MARQO_API_BASE_URL")

	// Set test environment variables
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", getNextPort())
	os.Setenv("MARQO_API_BASE_URL", testMarqoEndpoint)

	testMarqoApiKey := "test-marqo-api-key"
	os.Setenv("MARQO_API_KEY", testMarqoApiKey)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("MARQO_API_BASE_URL", originalMarqoEndpoint)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":          "123",
					"eventOwners": []interface{}{"789"},
					"name":        "Test Event",
					"description": "This is a test event",
				},
			},
		}
		responseBytes, err := json.Marshal(response)

		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}))

	// Set the mock Marqo server URL
	mockMarqoServer.Listener.Close()
	var err error
	mockMarqoServer.Listener, err = net.Listen("tcp", testMarqoEndpoint[len("http://"):])
	if err != nil {
		t.Fatalf("Failed to start mock Marqo server: %v", err)
	} else {
		t.Log("Started mock Marqo server")
	}
	mockMarqoServer.Start()
	defer mockMarqoServer.Close()

	const eventID = "123"
	req, err := http.NewRequest("GET", "/events/" + eventID, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up context with APIGatewayV2HTTPRequest
	ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
    PathParameters: map[string]string{
        helpers.EVENT_ID_KEY: eventID,
    },
	})
	req = req.WithContext(ctx)

	// Set up router to extract variables
	router := mux.NewRouter()
	router.HandleFunc("/events/{" + helpers.EVENT_ID_KEY + "}", func(w http.ResponseWriter, r *http.Request) {
		GetEventDetailsPage(w, r).ServeHTTP(w, r)
	})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}

	if !strings.Contains(rr.Body.String(), ">Test Event") {
		t.Errorf("Event title is missing from the page")
	}

	if !strings.Contains(rr.Body.String(), ">This is a test event") {
		t.Errorf("Event description is missing from the page")
	}
}

func TestGetAddEventSourcePage(t *testing.T) {
	req, err := http.NewRequest("GET", "/admin", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := GetAddEventSourcePage(rr, req)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if rr.Body.String() == "" {
		t.Errorf("Handler returned empty body")
	}
}
