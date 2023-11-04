package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate = validator.New()

type CreateUser struct {
    Name string `json:"name" validate:"required"`
    Kind string  `json:"kind" validate:"required"`
    Region string  `json:"region" validate:"required"`
} 

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    log.Printf("Received req %#v", req)

    switch req. HTTPMethod {
    case "GET":
        return processGetUsers(ctx)
    case "POST":
        return processPost(ctx, req)
    default:
        return clientError(http.StatusMethodNotAllowed)
    } 
}

func processGetUsers(ctx context.Context) (events.APIGatewayProxyResponse, error) {
    log.Print("Received GET Users request")

    users, err := listItems(ctx)
    if err != nil {
        return serverError(err)
    }

    json, err := json.Marshal(users)
    if err != nil {
        return serverError(err)
    } 
    log.Printf("Successfully fetched todos: %s", json)

    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusOK,
        Body: string(json),
    }, nil
} 

func processPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    var createUser CreateUser
    err := json.Unmarshal([]byte(req.Body), &createUser)
    if err != nil {
        log.Printf("Cannot unmarshal body: %v", err)
        return clientError(http.StatusUnprocessableEntity)
    } 

    err = validate.Struct(&createUser)
    if err != nil {
        log.Printf("Invalid body: %v", err)
        return clientError(http.StatusBadRequest)
    }
    log.Printf("Received POST request with item: %+v", createUser)

    res, err := insertItem(ctx, createUser)
    if err != nil {
        return serverError(err)
    }
    log.Printf("Inserted new user: %+v", res)

    json, err := json.Marshal(res)
    if err != nil {
        return serverError(err)
    }

    return events.APIGatewayProxyResponse{
        StatusCode: http.StatusCreated,
        Body: string(json),
        Headers: map[string]string{
            "Location": fmt.Sprintf("/user/%s", res.Id),
        },
    }, nil 
} 

func clientError(status int) (events.APIGatewayProxyResponse, error) {

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())
    log.Println("Hitting server error in routes")

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

