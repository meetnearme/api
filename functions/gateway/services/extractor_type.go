package services

import (
	"context"

	"github.com/meetnearme/api/functions/gateway/types"
)

// For different event extractors
type EventExtractor interface {
	CanHandle(url string) bool
	Extract(ctx context.Context, seshuJob types.SeshuJob, scraper ScrapingService) ([]types.EventInfo, string, error)
}
