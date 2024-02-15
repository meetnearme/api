package db

import (
    "context"
    "github.com/meetnearme/api/functions/lambda/shared"
)

type DB interface {
    ListItems(ctx context.Context) ([]shared.Event, error)
    InsertItem(ctx context.Context, event shared.CreateEvent) (*shared.Event, error)
} 
