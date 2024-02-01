package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
	// import relative module for views handled by templ
    "github.com/meetnearme/api/views"
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

func main() {
    lambda.Start(Router)
}

// func hello(name string) templ.Component {
// 	return templ.FromGoHTML(func(ctx context.Context, name string) (interface{}, error) {
// 		return templ.ToGoHTML(ctx, fmt.Sprintf("Hello, %s!", name))
// 	})
// }

func hello(name string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var templ_7745c5c3_Var2 string
		templ_7745c5c3_Var2, templ_7745c5c3_Err = templ.JoinStringErrs(name)
		if templ_7745c5c3_Err != nil {
			return templ.Error{Err: templ_7745c5c3_Err, FileName: `functions/lambda/views/hello.templ`, Line: 3, Col: 12}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var2))
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}

func Router(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    switch req.RequestContext.HTTP.Method {
    case "GET":
        component := Hello("World")
        var buf bytes.Buffer
        err := component.Render(ctx, &buf)
        if err != nil {
            return serverError(err)
        }
        return events.APIGatewayV2HTTPResponse{
            StatusCode: http.StatusOK,
            Body: buf.String(),
        }, nil
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

