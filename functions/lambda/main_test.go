package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/meetnearme/api/functions/lambda/shared"
)

// Mocks

var mockEvents = []shared.Event{
    {
        Name: "Chess Tournament",
        Description: "Join the junior chess tournament to test your abilities",
        Datetime: "2024-03-13T15:07:00",
        Address: "15 Chess Street",
        ZipCode: "84322", 
        Country: "USA",
        Id: uuid.NewString(),
    },
    {
        Name: "Dancing",
        Description: "Dance with us",
        Datetime: "2024-03-13T15:07:00",
        Address: "Dance town",
        ZipCode: "84322", 
        Country: "USA",
        Id: uuid.NewString(),
    },
}

var createEvent = CreateEvent{
            Name: "Eating ice cream",
            Description: "Dance with us",
            Datetime: "2024-03-13T15:07:00",
            Address: "Dance town",
            ZipCode: "84322", 
            Country: "USA",
}

// Mock of list Items for events

func TestRouter(t *testing.T) {
    // Test various methods for router
    ctx := context.Background()

    // Mock DB functions 
    listItems := func(ctx context.Context) ([]shared.Event, error) {
        return mockEvents, nil
    } 

    insertItem := func(ctx context.Context, event CreateEvent) (shared.Event, error) {
        return shared.Event{
            Name: "Eating ice cream",
            Description: "Dance with us",
            Datetime: "2024-03-13T15:07:00",
            Address: "Dance town",
            ZipCode: "84322", 
            Country: "USA",
            Id: uuid.NewString(),
        }, nil
    } 

    // Call mocks to satisfy compiler
    _, _ = listItems(ctx)
    _, _ = insertItem(ctx, createEvent)



    // Get request
    req := events.APIGatewayV2HTTPRequest{
        RequestContext: events.APIGatewayV2HTTPRequestContext{
            HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
                Method: http.MethodGet,
            },
        },
    } 

    resp, err := Router(ctx, req)
    if err != nil {
        t.Errorf("Error in Router function: %v", err)
    } 

    if resp.StatusCode != http.StatusOK {
        t.Errorf("Unexpected status code for GET request: %d", resp.StatusCode)
    } 

    // Post request 
    req.RequestContext.HTTP.Method = http.MethodPost

    resp, err = Router(ctx, req)
    if err != nil {
        t.Errorf("Error in Router function: %v", err)
    } 

    if resp.StatusCode != http.StatusCreated {
        t.Errorf("Unexpected status code for GET request: %d", resp.StatusCode)
    } 

    // Unsupported method 
    req.RequestContext.HTTP.Method = http.MethodPut

    resp, err = Router(ctx, req)
    if err != nil {
        t.Errorf("Error in Router function: %v", err)
    } 

    if resp.StatusCode != http.StatusMethodNotAllowed {
        t.Errorf("Unexpected status code for GET request: %d", resp.StatusCode)
    } 

} 
