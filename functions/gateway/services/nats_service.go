package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/meetnearme/api/functions/gateway/constants"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	streamName  = os.Getenv("NATS_SESHU_STREAM_NAME")
	subjectName = os.Getenv("NATS_SESHU_STREAM_SUBJECT")
	durableName = os.Getenv("NATS_SESHU_STREAM_DURABLE_NAME")
)

// abs returns the absolute value of an int64
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

type NatsService struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

type eventChecker struct {
	EventTitle     string
	EventLocation  string
	EventStartTime string
	EventId        string
}

func NewNatsService(ctx context.Context, conn *nats.Conn) (*NatsService, error) {

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	//Create stream if it does not exist
	_, err = js.Stream(ctx, streamName)

	if err != nil {

		fmt.Printf("Stream %s does not exist, creating it...\n", streamName)

		_, err = js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     streamName,
			Subjects: []string{subjectName},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create stream: %w", err)
		}
	}

	return &NatsService{
		conn: conn,
		js:   js,
	}, nil
}

func GetNatsClient() (*nats.Conn, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		return nil, fmt.Errorf("NATS_URL environment variable is required")
	}
	return nats.Connect(url)
}

func (s *NatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	stream, err := s.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          durableName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		FilterSubject: subjectName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	nextStreamSeq := info.AckFloor.Stream + 1

	streamInfo, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	if nextStreamSeq > streamInfo.State.LastSeq {
		log.Printf("No message at seq %d (stream ends at %d)", nextStreamSeq, streamInfo.State.LastSeq)
		return nil, nil
	}

	msg, err := stream.GetMsg(ctx, nextStreamSeq)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream msg at seq %d: %w", nextStreamSeq, err)
	}

	return msg, nil
}

func (s *NatsService) PublishMsg(ctx context.Context, job interface{}) error {

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	ack, err := s.js.Publish(ctx, subjectName, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf("Published msg with sequence number %d on stream %q\n", ack.Sequence, ack.Stream)

	return nil
}

func (s *NatsService) ConsumeMsg(ctx context.Context, workers int) error {

	cons, err := s.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable:       durableName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: subjectName,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update consumer: %w", err)
	}

	iter, err := cons.Messages(jetstream.PullMaxMessages(1))
	if err != nil {
		return fmt.Errorf("failed to get iterator: %w", err)
	}

	// Open PostgresService for DB operations
	db, err := GetPostgresService(ctx)
	if err != nil {
		log.Printf("Failed to get PostgresService: %v", err)
	}

	sem := make(chan struct{}, workers)

	for {
		sem <- struct{}{}
		go func() {
			defer func() {
				<-sem
			}()
			msg, err := iter.Next()
			if err != nil {
				// Check if context is cancelled or iterator is exhausted
				if err == jetstream.ErrMsgIteratorClosed || err == jetstream.ErrNoMessages {
					log.Printf("Iterator closed or no messages: %v", err)
					return
				}
				log.Printf("Error getting next message: %v", err)
				return
			}

			// Check if msg is nil before using it
			if msg == nil {
				log.Printf("Received nil message from iterator")
				return
			}

			// Unmarshal the SeshuJob from the message
			var seshuJob internal_types.SeshuJob
			if err := json.Unmarshal(msg.Data(), &seshuJob); err != nil {
				log.Printf("Failed to unmarshal SeshuJob: %v", err)
				msg.Ack() // Acknowledge to remove from queue even if processing failed
				return
			}

			if seshuJob.Status != "SCANNING" {
				seshuJob.Status = "SCANNING"
				err = db.UpdateSeshuJob(ctx, seshuJob)
				if err != nil {
					log.Printf("Failed to update SeshuJob after scrape failure: %v", err)
				}
			}

			log.Printf("Processing scraping job for URL: %s", seshuJob.NormalizedUrlKey)

			var scrapeMode string
			if seshuJob.IsRecursive {
				scrapeMode = "rs"
			} else {
				scrapeMode = "init"
			}

			events, _, err := ExtractEventsFromHTML(seshuJob, constants.SESHU_MODE_SCRAPE, scrapeMode, &RealScrapingService{})
			if err != nil {
				log.Printf("Failed to extract events from %s: %v", seshuJob.NormalizedUrlKey, err)
				// Update job status to reflect failure in database
				seshuJob.LastScrapeFailureCount++
				seshuJob.LastScrapeFailure = time.Now().Unix()
				seshuJob.Status = "FAILING"
				err = db.UpdateSeshuJob(ctx, seshuJob)
				if err != nil {
					log.Printf("Failed to update SeshuJob after scrape failure: %v", err)
				}
				msg.Ack()
				return
			}

			// Smart update: preserve events that still exist at the source URL
			// Only delete events that are no longer present at the source
			if len(events) > 0 {

				// Deduplicate newly scraped events before processing
				events, _ = deduplicateEvents(events)

				weaviateClient, err := GetWeaviateClient()
				if err != nil {
					log.Printf("Failed to get Weaviate client for %s: %v", seshuJob.NormalizedUrlKey, err)
					// Update job status to reflect failure in database
					seshuJob.LastScrapeFailureCount++
					seshuJob.LastScrapeFailure = time.Now().Unix()
					seshuJob.Status = "FAILING"
					err = db.UpdateSeshuJob(ctx, seshuJob)
					if err != nil {
						log.Printf("Failed to update SeshuJob after Weaviate client failure: %v", err)
					}
					msg.Ack()
					return
				} // Step 1: Gather all existing events in DB using EventSourceId
				currentTime := time.Now().Unix()
				eventSourceId := seshuJob.NormalizedUrlKey

				var existingdeduplicatedEvents []constants.Event
				var duplicateIds []string
				existingEvents := []constants.Event{}
				if eventSourceId != "" {
					searchResponse, err := SearchWeaviateEvents(
						context.Background(),
						weaviateClient,
						"",                                  // no text query
						nil,                                 // no location filter
						0,                                   // no distance filter
						currentTime,                         // startTime = current time (future events only)
						0,                                   // endTime = 0 (no end time limit)
						nil,                                 // no owner filter
						"",                                  // no category filter
						"",                                  // no address filter
						"",                                  // no date parsing
						[]string{constants.ES_SINGLE_EVENT}, // eventSourceTypes filter
						[]string{eventSourceId},             // eventSourceIds filter
					)

					if err != nil {
						log.Printf("Failed to search for existing future events with EventSourceId %s: %v", eventSourceId, err)
						// Update job status to reflect failure in database
						seshuJob.LastScrapeFailureCount++
						seshuJob.LastScrapeFailure = time.Now().Unix()
						seshuJob.Status = "FAILING"
						err = db.UpdateSeshuJob(ctx, seshuJob)
						if err != nil {
							log.Printf("Failed to update SeshuJob after Weaviate search failure: %v", err)
						}
						// Continue with processing even if search fails
						msg.Nak() // Requeue for retry
						return
					} else {
						existingEvents = searchResponse.Events
						// Deduplicate existing events based on Name + Location + Time
						// This prevents issues when Weaviate has duplicate entries
						existingdeduplicatedEvents, duplicateIds = deduplicateEvents(existingEvents)
					}
				}
				// Step 3: Scrape target URL (already done - events are scraped)
				log.Printf("Scraped %d new events from URL", len(events))
				preservedEventIds := make(map[string]bool)
				newEventIndicesToSkip := make(map[int]bool) // Track which new events to skip (already exist)

				for _, existingEvent := range existingdeduplicatedEvents {
					for j, newEvent := range events {
						// Skip if this new event already matched a previous existing event
						if newEventIndicesToSkip[j] {
							continue
						}

						// Match criteria: Name, Location, and Time
						nameMatch := existingEvent.Name == newEvent.EventTitle
						locationMatch := existingEvent.Address == newEvent.EventLocation

						// For time matching, parse the new event's time IN THE EVENT'S TIMEZONE
						// The scraped time string is in local time (same timezone as the existing event)
						timeMatch := false
						var timeDiff int64
						if newEvent.EventStartTime != "" {
							// Use the existing event's timezone to parse the new event time
							// This ensures we're comparing apples to apples (both in the same timezone context)
							newEventTime, err := time.ParseInLocation("2006-01-02T15:04:05", newEvent.EventStartTime, &existingEvent.Timezone)
							if err == nil {
								// Compare Unix timestamps (both in UTC now) - exact match required
								timeDiff = abs(existingEvent.StartTime - newEventTime.Unix())
								timeMatch = timeDiff == 0 // Exact time match
							} else {
								log.Printf("Time parse error: %v", err)
							}
						}

						if nameMatch && locationMatch && timeMatch {
							// This is a match - preserve the existing event and skip inserting the new one
							preservedEventIds[existingEvent.Id] = true
							newEventIndicesToSkip[j] = true
							break
						} else {
							continue
						}
					}
				}

				// Step 5 & 6: Collect all IDs to delete (duplicates + obsolete events)
				// Combine duplicate IDs and obsolete event IDs for single batch deletion
				allIdsToDelete := make([]string, 0)

				// Add duplicate IDs from deduplication
				if eventSourceId != "" && len(existingEvents) > 0 {
					allIdsToDelete = append(allIdsToDelete, duplicateIds...)
				}
				duplicateCount := len(allIdsToDelete)

				// Add obsolete event IDs (events not preserved - no longer at source)
				for _, existingEvent := range existingdeduplicatedEvents {
					if !preservedEventIds[existingEvent.Id] {
						allIdsToDelete = append(allIdsToDelete, existingEvent.Id)
					}
				}
				obsoleteCount := len(allIdsToDelete) - duplicateCount

				// Single batch delete operation for both duplicates and obsolete events
				if len(allIdsToDelete) > 0 {
					log.Printf("Deleting %d total events from Weaviate (%d duplicates + %d obsolete)",
						len(allIdsToDelete), duplicateCount, obsoleteCount)
					_, err = BulkDeleteEventsFromWeaviate(context.Background(), weaviateClient, allIdsToDelete)
					if err != nil {
						log.Printf("Failed to delete events: %v", err)
					} else {
						log.Printf("Successfully deleted %d events", len(allIdsToDelete))
					}
				} else {
					log.Printf("No events to delete for: %s", seshuJob.NormalizedUrlKey)
				}

				// Filter out new events that match existing events
				eventsToInsert := make([]internal_types.EventInfo, 0)
				for i, event := range events {
					if !newEventIndicesToSkip[i] {
						eventsToInsert = append(eventsToInsert, event)
					}
				}

				// Now insert ONLY new events (not matches)
				if len(eventsToInsert) == 0 {
					log.Printf("No new events to insert for %s (all events already exist)", seshuJob.NormalizedUrlKey)
				} else {
					log.Printf("Inserting %d new events for %s", len(eventsToInsert), seshuJob.NormalizedUrlKey)
					err = PushExtractedEventsToDB(eventsToInsert, seshuJob, make(map[string]string))
					if err != nil {
						log.Println("Error pushing new events to DB:", err)
						// Update job status to reflect failure in database
						seshuJob.LastScrapeFailureCount++
						seshuJob.LastScrapeFailure = time.Now().Unix()
						seshuJob.Status = "FAILING"
						err = db.UpdateSeshuJob(ctx, seshuJob)
						if err != nil {
							log.Printf("Failed to update SeshuJob after event insertion failure: %v", err)
						}
						msg.Ack()
						return
					}
				}

				log.Printf("Successfully processed %d events for %s (%d preserved, %d deleted, %d inserted)",
					len(events), seshuJob.NormalizedUrlKey, len(preservedEventIds), len(allIdsToDelete), len(eventsToInsert))
				msg.Ack()
			} else {
				log.Printf("No events scraped from %s", seshuJob.NormalizedUrlKey)
				msg.Ack()
			}

			// update job status accordingly
			seshuJob.LastScrapeFailureCount = 0
			seshuJob.Status = "HEALTHY"
			seshuJob.LastScrapeSuccess = time.Now().Unix()
			err = db.UpdateSeshuJob(ctx, seshuJob)
			if err != nil {
				log.Printf("Failed to update SeshuJob after scrape failure: %v", err)
			}
		}()
	}
}

func (s *NatsService) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}

// deduplicateEvents removes duplicate events based on Name + Location + StartTime
// Works with both constants.Event and internal_types.EventInfo
func deduplicateEvents[T any](events []T) ([]T, []string) {
	if len(events) == 0 {
		return events, nil
	}

	seen := make(map[string]string) // map[key]firstEventID
	uniqueEvents := make([]T, 0, len(events))
	duplicateIds := make([]string, 0)
	duplicateCount := 0

	for _, event := range events {
		var eventChecker eventChecker

		// Type switch to extract fields based on event type
		switch e := any(event).(type) {
		case constants.Event:
			eventChecker.EventTitle = e.Name
			eventChecker.EventLocation = e.Address
			eventChecker.EventStartTime = fmt.Sprintf("%d", e.StartTime)
			eventChecker.EventId = e.Id
		case internal_types.EventInfo:
			eventChecker.EventTitle = e.EventTitle
			eventChecker.EventLocation = e.EventLocation
			eventChecker.EventStartTime = e.EventStartTime
			eventChecker.EventId = "" // New events don't have IDs
		}

		if eventChecker.EventTitle == "" || eventChecker.EventLocation == "" || eventChecker.EventStartTime == "" {
			// Skip events with missing critical fields
			continue
		}

		key := fmt.Sprintf("%s|%s|%s", eventChecker.EventTitle, eventChecker.EventLocation, eventChecker.EventStartTime)

		if firstID, exists := seen[key]; !exists {
			seen[key] = eventChecker.EventId
			uniqueEvents = append(uniqueEvents, event)
		} else {
			duplicateCount++
			if eventChecker.EventId != "" {
				log.Printf("WARNING: Duplicate event detected (will be deleted): Key=%s, DuplicateID=%s (keeping FirstID=%s)",
					key, eventChecker.EventId, firstID)
				duplicateIds = append(duplicateIds, eventChecker.EventId)
			} else {
				log.Printf("WARNING: Duplicate new event detected (skipping): Key=%s", key)
			}
		}
	}

	return uniqueEvents, duplicateIds
}
