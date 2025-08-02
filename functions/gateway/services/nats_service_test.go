package services

import (
	"context"
	"testing"

	"github.com/meetnearme/api/functions/gateway/test_helpers"
	internal_types "github.com/meetnearme/api/functions/gateway/types"
)

func TestPublishMsg(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey:         "example-event-1",
		LocationLatitude:         1.352222231,
		LocationLongitude:        103.8198,
		LocationAddress:          "123 Orchard Road, Singapore",
		ScheduledHour:            15,
		TargetNameCSSPath:        ".event-title",
		TargetLocationCSSPath:    ".event-location",
		TargetStartTimeCSSPath:   ".start-time",
		TargetEndTimeCSSPath:     ".end-time",
		TargetDescriptionCSSPath: ".description",
		TargetHrefCSSPath:        "a.more-info",
		Status:                   "HEALTHY",
		LastScrapeSuccess:        1727385600,
		LastScrapeFailure:        0,
		LastScrapeFailureCount:   0,
		OwnerID:                  "user_abc123",
		KnownScrapeSource:        "MEETUP",
	}

	err := mockQueue.PublishMsg(ctx, payload)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockQueue.PublishedMsgs) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(mockQueue.PublishedMsgs))
	}
}

func TestPeekTopOfQueue(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "example-event-1",
	}

	_ = mockQueue.PublishMsg(ctx, payload)

	top, err := mockQueue.PeekTopOfQueue(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if top == nil {
		t.Error("Expected non-nil top message, got nil")
	}
}

func TestPeekFromEmptyQueue(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	top, err := mockQueue.PeekTopOfQueue(ctx)
	if err == nil {
		t.Error("Expected error when peeking from empty queue, got nil")
	}
	if top != nil {
		t.Error("Expected nil message when queue is empty")
	}
}

func TestConsumeMsg(t *testing.T) {
	ctx := context.Background()
	mockQueue := test_helpers.NewMockNatsService()

	payload := internal_types.SeshuJob{
		NormalizedUrlKey: "example-event-1",
	}

	_ = mockQueue.PublishMsg(ctx, payload)

	err := mockQueue.ConsumeMsg(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	top, _ := mockQueue.PeekTopOfQueue(ctx)
	if top != nil {
		t.Error("Expected queue to be empty after ConsumeMsg, but it's not")
	}
}
