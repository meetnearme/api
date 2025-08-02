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
				// Check if context is cancelled
			}
			fmt.Printf("Processing msg: %s\n", string(msg.Data()))
			time.Sleep(time.Second * 60) // Simulate processing time
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
