package main

import (
	// "fmt"
	"context"
	"fmt"
	"log"

	// "net/http"

	// "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// func handleHelloTest(w http.ResponseWriter, r *http.Request) {
//     w.Write([]byte("Hello from the Go server!!"))
// }

// func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
//     fmt.Println("Hello World")
//     log.Println("Hello world")

//     resp := events.APIGatewayProxyResponse{
//         StatusCode: 200,
//     }

//     return resp, nil
// }

func handler(ctx context.Context, event Event) (string, error) {
        log.Print("value1 = ", event["key1"] )
        log.Print("value2 = ", event["key2"] )
        log.Print("value3 = ", event["key3"] )

        return fmt.Sprintf("Hello World"), nil
}



func main() {
    lambda.Start(handler)
    // server := http.NewServeMux()
    // server.HandleFunc("/", handleHelloTest)

    // fmt.Printf("Serving on PORT 3001")
    // if err := http.ListenAndServe(":3001", server); err != nil {
    //     log.Fatal(err)
    // } 
} 


