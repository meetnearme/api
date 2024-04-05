package transport

import (
	"context"
	"fmt"
)

func LogRequest(ctx context.Context, req Request) (context.Context, Request, error) {
	reqMethod := req.RequestContext.HTTP.Method
	reqPath := req.RequestContext.HTTP.Path

	fmt.Println("Received request: " + reqMethod + " " + reqPath)
	return ctx, req, nil
}
