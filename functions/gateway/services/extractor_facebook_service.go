package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/types"
)

type FacebookExtractor struct{}

func (f *FacebookExtractor) CanHandle(url string) bool {
	return IsFacebookEventsURL(url)
}

func (f *FacebookExtractor) Extract(ctx context.Context, seshuJob types.SeshuJob, scraper ScrapingService) ([]types.EventInfo, string, error) {

	mode := ctx.Value("MODE").(string)
	knownScrapeSource := ""

	validate := func(content string) bool {
		return strings.Contains(content, `"__typename":"Event"`)
	}

	html, err := scraper.GetHTMLFromURLWithRetries(seshuJob, 7500, true, "script[data-sjs][data-content-len]", 7, validate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get HTML from Facebook URL: %w", err)
	}

	if mode != constants.SESHU_MODE_ONBOARD {

		childScrapeQueue := []types.EventInfo{}
		urlToIndex := make(map[string]int)

		eventsFound, err := FindFacebookEventListData(html, seshuJob.NormalizedUrlKey, seshuJob.LocationTimezone)
		if err != nil {
			log.Printf("ERR: Failed to extract Facebook event list data: %v", err)
			return nil, "", err
		}

		for i, event := range eventsFound {
			eventsFound[i].KnownScrapeSource = knownScrapeSource
			// TODO: we could arguably this any time we have a URL,
			// searching even for things like Title, StartTime, etc.
			// but for now we're only assuming these missing fields have a
			// chance of triggering a child scrape

			// If existing seshujobs has location data, means seshujob used "all events at the target URL located in the same geography?"
			f.applyLocationData(&eventsFound[i], seshuJob)

			if event.EventDescription == "" || event.EventLocation == "" || event.EventTimezone == "" {
				childScrapeQueue = append(childScrapeQueue, event)
				urlToIndex[event.EventURL] = i
			}
		}

		if len(childScrapeQueue) <= 0 {
			return eventsFound, html, err
		}

		// Do we need child scrapes for facebook events?
		for _, event := range childScrapeQueue {
			childHtml, err := scraper.GetHTMLFromURLWithRetries(types.SeshuJob{NormalizedUrlKey: event.EventURL}, 7500, true, "script[data-sjs][data-content-len]", 7, validate)
			if err != nil {
				log.Printf("ERR: Failed to get child HTML from %s: %v", event.EventURL, err)
				continue
			}
			// Use the single event mode function for child pages
			childEvArrayOfOne, err := FindFacebookSingleEventData(childHtml, seshuJob.NormalizedUrlKey, seshuJob.LocationTimezone)
			if err != nil {
				log.Printf("ERR: Failed to extract single event data from child page: %v", err)
				continue
			}

			// Look up the original index using the URL
			originalIndex := urlToIndex[event.EventURL]
			if eventsFound[originalIndex].EventDescription == "" {
				eventsFound[originalIndex].EventDescription = childEvArrayOfOne[0].EventDescription
			}
			if eventsFound[originalIndex].EventLocation == "" {
				eventsFound[originalIndex].EventLocation = childEvArrayOfOne[0].EventLocation
			}

			// Store the event URL as the source URL for the child event
			childEvArrayOfOne[0].SourceUrl = event.EventURL

			// derive timezone from coordinates, without it all time data is unstable
			if eventsFound[originalIndex].EventTimezone == "" &&
				childEvArrayOfOne[0].EventLatitude != 0 &&
				childEvArrayOfOne[0].EventLongitude != 0 {
				derivedTimezone := DeriveTimezoneFromCoordinates(
					childEvArrayOfOne[0].EventLatitude,
					childEvArrayOfOne[0].EventLongitude,
				)
				if derivedTimezone != "" {
					eventsFound[originalIndex].EventTimezone = derivedTimezone
				}
			}
		}

		return eventsFound, html, err

	}

	// Else logic for ONBOARD mode
	eventsFound, err := FindFacebookEventListData(html, seshuJob.NormalizedUrlKey, seshuJob.LocationTimezone)
	if err != nil {
		log.Printf("ERR: Failed to extract Facebook event list data in %s mode: %v", mode, err)
		return nil, "", fmt.Errorf("failed to extract Facebook event list data: %w", err)
	}

	return eventsFound, html, nil
}

func (f *FacebookExtractor) applyLocationData(event *types.EventInfo, seshuJob types.SeshuJob) {
	if seshuJob.LocationLatitude != constants.INITIAL_EMPTY_LAT_LONG {
		event.EventLatitude = seshuJob.LocationLatitude
	}
	if seshuJob.LocationLongitude != constants.INITIAL_EMPTY_LAT_LONG {
		event.EventLongitude = seshuJob.LocationLongitude
	}
	if seshuJob.LocationAddress != "" {
		event.EventLocation = seshuJob.LocationAddress
	}
}
