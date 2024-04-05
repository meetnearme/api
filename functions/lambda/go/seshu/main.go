package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator"
)

var validate *validator.Validate = validator.New()
var converter = md.NewConverter("", true, nil)

type InputPayload struct {
    Url string `json:"url" validate:"required"`
}

func Router(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
    switch req.RequestContext.HTTP.Method {
    case "POST":
				return handlePost(ctx, req)
    default:
        return clientError(http.StatusMethodNotAllowed)
    }
}

func handlePost(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	var inputPayload InputPayload
	err := json.Unmarshal([]byte(req.Body), &inputPayload)
	if err != nil {
			log.Printf("Invalid JSON payload: %v", err)
			return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&inputPayload)
	if err != nil {
			log.Printf("Invalid body: %v", err)
			return clientError(http.StatusBadRequest)
	}

	log.Println("URL: ", inputPayload.Url)
	if err != nil {
			return serverError(err)
	}

	// start of ZR code
	client := &http.Client{}
	zrReq, zrErr := http.NewRequest("GET", "https://api.zenrows.com/v1/?apikey=" + os.Getenv("ZENROWS_API_KEY") + "&url=" + url.QueryEscape(inputPayload.Url) + "&js_render=true&wait=2500", nil)
	if zrErr != nil {
		log.Fatalln(zrErr)
	}
	zrRes, zrErr := client.Do(zrReq)
	if zrErr != nil {
			log.Fatalln(zrErr)
	}
	defer zrRes.Body.Close()

	zrBody, zrErr := io.ReadAll(zrRes.Body)
	if zrErr != nil {
			log.Fatalln(zrErr)
	}



	zrBodyString := string(zrBody)
	// avoid extra parsing work for <head> content outside of <body>
	re := regexp.MustCompile(`<body>(.*?)<\/body>`)
	matches := re.FindStringSubmatch(zrBodyString)
	if len(matches) > 1 {
		zrBodyString = matches[1]
	}
	markdown, err := converter.ConvertString(zrBodyString)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: consider log levels / log volume
	return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusCreated,
			Body: string(markdown),
			Headers: map[string]string{
					"Location": fmt.Sprintf("/user/%s", "hello res"),
			},
	}, nil
}


// TODO: this should share with the gateway handler, though the
// function signature typing is different
func clientError(status int) (events.LambdaFunctionURLResponse, error) {
	return events.LambdaFunctionURLResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

// TODO: this should share with the gateway handler
func serverError(err error) (events.LambdaFunctionURLResponse, error) {
	log.Println(err.Error())

	return events.LambdaFunctionURLResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func main() {
    lambda.Start(Router)
}
