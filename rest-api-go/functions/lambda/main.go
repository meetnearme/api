package main

import ( 

	"github.com/aws/aws-lambda-go/lambda"
)


// func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
//     fmt.Println("Hello World")
//     log.Println("Hello world")

//     resp := events.APIGatewayProxyResponse{
//         StatusCode: 200,
//         Body: "I am returning from the server",
//     }

//     return resp, nil
// }

func main() {
    lambda.Start(router)
} 


