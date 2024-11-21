package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gorilla/mux"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetHomePage(t *testing.T) {
	// Save original environment variables
	originalMarqoApiKey := os.Getenv("MARQO_API_KEY")
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"Hits": []map[string]interface{}{
				{
					"id":             "123",
					"eventOwners":    []interface{}{"789"},
					"eventOwnerName": "First Event Host",
					"name":           "First Test Event",
					"description":    "Description of the first event",
					"startTime":      "2099-05-01T12:00:00Z",
				},
				{
					"id":             "456",
					"eventOwnerName": "Second Event Host",
					"eventOwners":    []interface{}{"012"},
					"name":           "Second Test Event",
					"description":    "Description of the second event",
					"startTime":      "2099-05-01T17:00:00Z",
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
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"Hits": []map[string]interface{}{
				{
					"id":             "123",
					"eventOwners":    []interface{}{"789"},
					"eventOwnerName": "Event Host Test",
					"name":           "First Test Event",
					"description":    "Description of the first event",
				},
				{
					"id":             "456",
					"eventOwners":    []interface{}{"012"},
					"eventOwnerName": "Event Host Test",
					"name":           "Second Test Event",
					"description":    "Description of the second event",
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

func TestGetProfilePage(t *testing.T) {
	req, err := http.NewRequest("GET", "/profile", nil)
	if err != nil {
		t.Fatal(err)
	}

	mockUserInfo := helpers.UserInfo{
		Email:             "test@domain.com",
		EmailVerified:     true,
		GivenName:         "Demo",
		FamilyName:        "User",
		Name:              "Demo User",
		PreferredUsername: "test@domain.com",
		Sub:               "testID",
		UpdatedAt:         123234234,
	}

	mockRoleClaims := []helpers.RoleClaim{
		{
			Role:        "orgAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "superAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
		{
			Role:        "sysAdmin",
			ProjectID:   "project-id",
			ProjectName: "myapp.zitadel.cloud",
		},
	}

	ctx := context.WithValue(req.Context(), "userInfo", mockUserInfo)
	ctx = context.WithValue(ctx, "roleClaims", mockRoleClaims)
	req = req.WithContext(ctx)

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
	originalMarqoEndpoint := os.Getenv("DEV_MARQO_API_BASE_URL")
	originalMarqoIndexName := os.Getenv("DEV_MARQO_INDEX_NAME")

	// Set test environment variables
	testMarqoApiKey := "test-marqo-api-key"
	testMarqoEndpoint := fmt.Sprintf("http://localhost:%d", test_helpers.GetNextPort())
	testMarqoIndexName := "testing-index"

	os.Setenv("MARQO_API_KEY", testMarqoApiKey)
	os.Setenv("DEV_MARQO_API_BASE_URL", testMarqoEndpoint)
	os.Setenv("DEV_MARQO_INDEX_NAME", testMarqoIndexName)

	// Defer resetting environment variables
	defer func() {
		os.Setenv("MARQO_API_KEY", originalMarqoApiKey)
		os.Setenv("DEV_MARQO_API_BASE_URL", originalMarqoEndpoint)
		os.Setenv("DEV_MARQO_INDEX_NAME", originalMarqoIndexName)
	}()

	// Create a mock HTTP server for Marqo
	mockMarqoServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the response
		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"_id":            "123",
					"eventOwners":    []interface{}{"789"},
					"eventOwnerName": "Event Host Test",
					"name":           "Test Event",
					"description":    "This is a test event",
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
	req, err := http.NewRequest("GET", "/event/"+eventID, nil)
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
	router.HandleFunc("/event/{"+helpers.EVENT_ID_KEY+"}", func(w http.ResponseWriter, r *http.Request) {
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

func TestGetSearchParamsFromReq(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		cfRay          string
		expectedQuery  string
		expectedLoc    []float64
		expectedRadius float64
		expectedStart  int64
		expectedEnd    int64
		expectedCfLoc  helpers.CdnLocation
	}{
		{
			name: "All parameters provided",
			queryParams: map[string]string{
				"start_time": "4070908800",
				"end_time":   "4071808800",
				"lat":        "40.7128",
				"lon":        "-74.0060",
				"radius":     "1000",
				"q":          "test query",
			},
			cfRay:          "1234567890000-EWR",
			expectedQuery:  "test query",
			expectedLoc:    []float64{40.7128, -74.0060},
			expectedRadius: 1000,
			expectedStart:  4070908800,
			expectedEnd:    4071808800,
			expectedCfLoc:  helpers.CfLocationMap["EWR"],
		},
		{
			name: "Lat + lon params with no radius",
			queryParams: map[string]string{
				"start_time": "4070908800",
				"end_time":   "4071808800",
				"lat":        "40.7128",
				"lon":        "-74.0060",
				"radius":     "",
				"q":          "",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{40.7128, -74.0060},
			expectedRadius: 150,
			expectedStart:  4070908800,
			expectedEnd:    4071808800,
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name:           "No parameters provided",
			queryParams:    map[string]string{},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{39.8283, -98.5795},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name: "Only location parameters",
			queryParams: map[string]string{
				"lat":    "35.6762",
				"lon":    "139.6503",
				"radius": "500",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{35.6762, 139.6503},
			expectedRadius: 500,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be one month from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name: "Only time parameters",
			queryParams: map[string]string{
				"start_time": "this_week",
				"end_time":   "",
			},
			cfRay:          "",
			expectedQuery:  "",
			expectedLoc:    []float64{39.8283, -98.5795},
			expectedRadius: 2500.0,
			expectedStart:  0, // This will be the current time in Unix seconds
			expectedEnd:    0, // This will be 7 days from now in Unix seconds
			expectedCfLoc:  helpers.CdnLocation{},
		},
		{
			name:           "Only CF-Ray header",
			queryParams:    map[string]string{},
			cfRay:          "1234567890000-LAX",
			expectedLoc:    []float64{helpers.CfLocationMap["LAX"].Lat, helpers.CfLocationMap["LAX"].Lon}, // Los Angeles coordinates
			expectedCfLoc:  helpers.CfLocationMap["LAX"],
			expectedRadius: 150.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL:    &url.URL{RawQuery: encodeParams(tt.queryParams)},
				Header: make(http.Header),
			}
			if tt.cfRay != "" {
				req.Header.Set("cf-ray", tt.cfRay)
			}

			// TODO: need to test `categories` and `ownerIds` returned here
			query, loc, radius, start, end, cfLoc, _, _, _, _ := GetSearchParamsFromReq(req)

			if query != tt.expectedQuery {
				t.Errorf("Expected query %s, got %s", tt.expectedQuery, query)
			}

			if !floatSliceEqual(loc, tt.expectedLoc, 0.0001) {
				t.Errorf("Expected location %v, got %v", tt.expectedLoc, loc)
			}

			if math.Abs(radius-tt.expectedRadius) > 0.0001 {
				t.Errorf("Expected radius %f, got %f", tt.expectedRadius, radius)
			}

			if tt.expectedStart != 0 {
				if start != tt.expectedStart {
					t.Errorf("Expected start time %d, got %d", tt.expectedStart, start)
				}
			} else {
				if start <= 0 {
					t.Errorf("Expected start time to be greater than 0, got %d", start)
				}
			}

			if tt.expectedEnd != 0 {
				if end != tt.expectedEnd {
					t.Errorf("Expected end time %d, got %d", tt.expectedEnd, end)
				}
			} else {
				if end <= start {
					t.Errorf("Expected end time to be greater than start time, got start: %d, end: %d", start, end)
				}
			}

			if cfLoc != tt.expectedCfLoc {
				t.Errorf("Expected CF location %v, got %v", tt.expectedCfLoc, cfLoc)
			}
		})
	}
}

func encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	return values.Encode()
}

func floatSliceEqual(a, b []float64, epsilon float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > epsilon {
			return false
		}
	}
	return true
}
