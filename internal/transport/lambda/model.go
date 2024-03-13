package transport

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// aliasing the types to keep lines short
type Request = events.APIGatewayV2HTTPRequest
type Response = events.APIGatewayV2HTTPResponse

// HandlerFunc is a generic JSON Lambda handler used to chain middleware.
type HandlerFunc func(context.Context, Request) (interface{}, error)
