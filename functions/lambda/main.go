package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
	"github.com/meetnearme/api/functions/lambda/shared"
	"github.com/meetnearme/api/functions/lambda/views"
)

var validate *validator.Validate = validator.New()

type CreateEvent struct {
    Name string `json:"name" validate:"required"`
    Description string  `json:"description" validate:"required"`
    Datetime string  `json:"datetime" validate:"required"`
    Address string  `json:"address" validate:"required"`
    ZipCode string  `json:"zip_code" validate:"required"`
    Country string  `json:"country" validate:"required"`
}


var Pages = []shared.Page{
	{
		Name:     "My Account (test)",
		Desc:     "My Account page",
		Slug:     "my-account",
		// // Handlers: csrf.Handlers,
	},
	{
		Name:     "My Events",
		Desc:     "List of events I'm attending",
		Slug:     "my-events",
		// // Handlers: clicktoedit.Handlers,
	},
}

func Router(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    switch req.RequestContext.HTTP.Method {
    case "GET":
        component := views.Home(Pages)
        var buf bytes.Buffer
        err := component.Render(ctx, &buf)
        if err != nil {
            return serverError(err)
        }
        return events.APIGatewayV2HTTPResponse{
            Headers: map[string]string{"Content-Type": "text/html"},
            StatusCode: http.StatusOK,
            IsBase64Encoded: false,
            Body: buf.String(),
        }, nil
        // return processGetEvents(ctx)
    case "POST":
        return processPost(ctx, req)
    default:
        return clientError(http.StatusMethodNotAllowed)
    }
}

func processGetEvents(ctx context.Context) (events.APIGatewayV2HTTPResponse, error) {
    log.Print("Received GET events request")

    eventList, err := listItems(ctx)
    if err != nil {
        return serverError(err)
    }

    json, err := json.Marshal(eventList)
    if err != nil {
        return serverError(err)
    }
    log.Printf("Successfully fetched todos: %s", json)

    return events.APIGatewayV2HTTPResponse{
        StatusCode: http.StatusOK,
        Body: string(json),
    }, nil
}

func processPost(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    var createEvent CreateEvent
    err := json.Unmarshal([]byte(req.Body), &createEvent)
    if err != nil {
        log.Printf("Cannot unmarshal body: %v", err)
        return clientError(http.StatusUnprocessableEntity)
    }

    err = validate.Struct(&createEvent)
    if err != nil {
        log.Printf("Invalid body: %v", err)
        return clientError(http.StatusBadRequest)
    }
    log.Printf("Received POST request with item: %+v", createEvent)

    res, err := insertItem(ctx, createEvent)
    if err != nil {
        return serverError(err)
    }
    log.Printf("Inserted new user: %+v", res)

    json, err := json.Marshal(res)
    if err != nil {
        return serverError(err)
    }

    return events.APIGatewayV2HTTPResponse{
        StatusCode: http.StatusCreated,
        Body: string(json),
        Headers: map[string]string{
            "Location": fmt.Sprintf("/user/%s", "hello res"),
        },
    }, nil
}

func clientError(status int) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayV2HTTPResponse, error) {
	log.Println(err.Error())
    log.Println("Hitting server error in routes")

	return events.APIGatewayV2HTTPResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func main() {
    lambda.Start(Router)
}
