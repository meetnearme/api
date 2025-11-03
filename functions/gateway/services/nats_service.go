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

type NatsService struct {
	conn *nats.Conn
	js   jetstream.JetStream
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
				// TODO: Update job status to reflect failure in database
				msg.Ack()
				return
			}

			// query Weaviate for existing FUTRE events with a matching
			// `EventSourceId`, if they exist, delete all of them
			// we do this because we are updating our database based on
			// the URL scraping target as a source of truth, if that URL
			// target has no events, we leave PAST events alone, but we
			// delete future events since they conflict with the URL and
			// it's current content as the source of truth for events

			// Delete existing FUTURE events with matching EventSourceId
			if len(events) > 0 {
				weaviateClient, err := GetWeaviateClient()
				if err != nil {
					log.Printf("Failed to get Weaviate client for %s: %v", seshuJob.NormalizedUrlKey, err)
					// TODO: Update job status to reflect failure in database
					msg.Ack()
					return
				}

				// Get current timestamp for future event filtering
				currentTime := time.Now().Unix()

				// Search for existing FUTURE events with matching EventSourceId
				// We'll use the first event's EventSourceId as they should all be the same from the same source
				eventSourceId := events[0].EventURL
				if eventSourceId != "" {
					// Search for future events with matching EventSourceId
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
						// Continue with processing even if search fails
					} else if len(searchResponse.Events) > 0 {
						// Extract event IDs for deletion
						eventIdsToDelete := make([]string, 0, len(searchResponse.Events))
						for _, event := range searchResponse.Events {
							if event.Id != "" {
								eventIdsToDelete = append(eventIdsToDelete, event.Id)
							}
						}

						if len(eventIdsToDelete) > 0 {
							log.Printf("Found %d existing future events with EventSourceId %s, deleting them", len(eventIdsToDelete), eventSourceId)

							// Delete the existing future events
							_, err = BulkDeleteEventsFromWeaviate(context.Background(), weaviateClient, eventIdsToDelete)
							if err != nil {
								log.Printf("Failed to delete existing future events with EventSourceId %s: %v", eventSourceId, err)
								// Continue with processing even if deletion fails
							} else {
								log.Printf("Successfully deleted %d existing future events with EventSourceId %s", len(eventIdsToDelete), eventSourceId)
							}
						}
					}
				}

				// TODO: bring this back with properly extracted events

				if len(events) == 0 {
					log.Printf("No events extracted from %s", seshuJob.NormalizedUrlKey)
				} else {
					log.Printf("Extracted %d events from %s", len(events), seshuJob.NormalizedUrlKey)
				}

				err = PushExtractedEventsToDB(events, seshuJob)
				if err != nil {
					log.Println("Error pushing ingested events to DB:", err)
				}

				log.Printf("Successfully stored %d events in Weaviate for %s", len(events), seshuJob.NormalizedUrlKey)
				msg.Ack()
				return
			}

			// TODO: Update the SeshuJob status and timestamps in database
			// - Set LastScrapeSuccess to current timestamp
			// - Update Status to "HEALTHY" if successful
			// - Update to other status if not successful
			// - Reset LastScrapeFailureCount to 0

			log.Printf("Successfully processed scraping job for URL: %s", seshuJob.NormalizedUrlKey)
			msg.Ack()
		}()
	}
}

func (s *NatsService) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}
