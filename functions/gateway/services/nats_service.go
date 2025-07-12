package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

		fmt.Printf("Stream %s does not exist, creating it...\n", "SESJU_JOBS_STREAM")

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

func (s *NatsService) GetTopOfQueue(ctx context.Context) (*jetstream.RawStreamMsg, error) {

	stream, _ := s.js.Stream(ctx, streamName)
	msg, _ := stream.GetLastMsgForSubject(ctx, subjectName)

	return msg, nil
}

func (s *NatsService) PublishMsg(ctx context.Context, job interface{}) error {

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	_, err = s.js.PublishMsg(ctx, &nats.Msg{
		Data:    data,
		Subject: subjectName,
	})

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (s *NatsService) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}
