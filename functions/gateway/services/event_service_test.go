package services

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/meetnearme/api/functions/gateway/indexing"
	"github.com/meetnearme/api/functions/gateway/test_helpers"
)

func TestGetEventsZOrder(t *testing.T) {
    tests := []struct {
        name string
        mockScanOutput *dynamodb.ScanOutput
        mockScanError error
        startTime time.Time
        endTime time.Time
        lat float64
        lon float64
        radius float64
        expectedEvents []EventSelect
        expectedError error
    }{
         {
            name: "successful retrieval",
            mockScanOutput: &dynamodb.ScanOutput{
                Items: []map[string]types.AttributeValue{
                    // ... (mock items)
                },
            },
            startTime:      time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC),
            endTime:        time.Date(2023, 5, 2, 0, 0, 0, 0, time.UTC),
            lat:            51.5074,
            lon:            -0.1278,
            radius:         10.0,
            expectedEvents: []EventSelect{
                // ... (expected events)
            },
        },
        {
            name:          "database error",
            mockScanError: fmt.Errorf("database error"),
            expectedError: fmt.Errorf("database error"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockDB := &test_helpers.MockDynamoDBClient{
                ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
                    return tt.mockScanOutput, tt.mockScanError
                },
            }

            events, err := GetEventsZOrder(context.Background(), mockDB, tt.startTime, tt.endTime, tt.lat, tt.lon, tt.radius)

            if tt.expectedError != nil {
                if err == nil || err.Error() != tt.expectedError.Error() {
                    t.Errorf("expected error %v, got %v", tt.expectedError, err)
                }
            } else if err != nil {
                t.Errorf("unexpected error: %v", err)
            } else if !reflect.DeepEqual(events, tt.expectedEvents) {
                t.Errorf("expected events %+v, got %+v", tt.expectedEvents, events)
            }
        })
    }
}

func TestInsertEvent(t *testing.T) {
    var capturedInput *dynamodb.PutItemInput

    mockDB := &test_helpers.MockDynamoDBClient{
        PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
            capturedInput = params
            return &dynamodb.PutItemOutput{}, nil
        },
    }

    eventTime := time.Date(2030, 5, 1, 12, 0, 0, 0, time.UTC)
    createEvent := EventInsert{
        Name:        "New Event",
        Description: "New Description",
        Datetime:    eventTime.Format(time.RFC3339),
        Address:     "New Address",
        ZipCode:     "12345",
        Country:     "New Country",
        Latitude:    float32(51.5074),
        Longitude:   float32(-0.1278),
    }

    newEvent, err := InsertEvent(context.Background(), mockDB, createEvent)

    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }

    if newEvent == nil {
        t.Fatal("Expected newEvent to be non-nil")
    }

    if newEvent.Name != createEvent.Name {
        t.Errorf("Expected Name to be %s, got %s", createEvent.Name, newEvent.Name)
    }

    if capturedInput == nil {
        t.Fatal("Expected PutItemInput to be captured")
    }

    if *capturedInput.TableName != eventsTableName {
        t.Errorf("Expected TableName to be %s, got %s", eventsTableName, *capturedInput.TableName)
    }

    var insertedEvent EventSelect
    err = attributevalue.UnmarshalMap(capturedInput.Item, &insertedEvent)
    if err != nil {
        t.Fatalf("Failed to unmarshal PutItem input: %v", err)
    }

    if insertedEvent.Name != createEvent.Name {
        t.Errorf("Expected inserted event Name to be %s, got %s", createEvent.Name, insertedEvent.Name)
    }

    if len(insertedEvent.ZOrderIndex) == 0 {
        t.Error("Expected ZOrderIndex to be set")
    }

    expectedZOrder, err := indexing.CalculateZOrderIndex(eventTime, createEvent.Latitude, createEvent.Longitude, "normal")
    if err != nil {
        t.Fatalf("Failed to calculate expected ZOrderIndex: %v", err)
    }

    if !reflect.DeepEqual(insertedEvent.ZOrderIndex[:12], expectedZOrder[:12]) {
        t.Errorf("Unexpected ZOrderIndex: got %v, want %v", insertedEvent.ZOrderIndex[:12], expectedZOrder[:12])
    }
}
