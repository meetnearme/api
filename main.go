package main

import (
	// "fmt"
	"log"
	// "net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// func handleHelloTest(w http.ResponseWriter, r *http.Request) {
//     w.Write([]byte("Hello from the Go server!!"))
// }

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    log.Println("Hello world")

    resp := events.APIGatewayProxyResponse{
        StatusCode: 200, 
    }

    return resp, nil 
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


