package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

var (
	streamName  = os.Getenv("NATS_SESHU_STREAM_NAME")
	subjectName = os.Getenv("NATS_SESHU_STREAM_SUBJECT")
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

		_, err = js.CreateStream(ctx, jetstream.StreamConfig{ // create stream if it does not exist
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

// func (s *NatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
// 	stream, err := s.js.Stream(ctx, streamName)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get stream: %w", err)
// 	}

// 	msg, _ := stream.GetLastMsgForSubject(ctx, streamName)

// 	return msg, nil
// }

func (s *NatsService) PeekTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {
	stream, err := s.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Use a dedicated durable for peeking
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "peek-durable",
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Step 1: check if the queue is empty
	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	infoBytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal consumer info: %v\n", err)
		return nil, err
	}

	log.Printf("Consumer Info:\n%s\n", string(infoBytes))

	if info.NumPending == 0 {
		return nil, nil
	}

	msg, err := stream.GetLastMsgForSubject(ctx, subjectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	return msg, nil
}

func (s *NatsService) PublishMsg(ctx context.Context, job interface{}) error {

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	_, err = s.js.Publish(ctx, subjectName, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (s *NatsService) ConsumeMsg(ctx context.Context, workers int) error {

	consumer, err := s.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:        "consumer",
		Durable:     "seshu-consumer",
		Description: "Consumer for processing jobs",
		BackOff: []time.Duration{
			5 * time.Second,
		},
		FilterSubject: subjectName,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	return nil
}

func (s *NatsService) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}
