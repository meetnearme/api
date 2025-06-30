package services

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const basicHTMLresp = `<html><body><h1>Test Page</h1></body></html>`

func TestMain(m *testing.M) {
	log.Println("Running TestMain: Setup up for 'services' package")

	log.Println("Setting up auth flag mock values...")
	*domain = "meet-near-me-production-8baqim.ch1.zitadel.cloud"
	*key = "test-key"
	*clientID = "test-client-id"
	*clientSecret = "test-client-secret"
	*redirectURI = "https://test-redirect.com"

	InitAuth()
	log.Println("Auth service initialized with mock values.")

	// --- Part 2: Setup for Scraping Tests ---
	// This starts a mock server for any test that needs to scrape a URL.
	mockScrapingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(basicHTMLresp))
	}))
	defer mockScrapingServer.Close()

	exitCode := m.Run()

	log.Println("Tests have completed. Doing tear down.")

	os.Exit(exitCode)
}
