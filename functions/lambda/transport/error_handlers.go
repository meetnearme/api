package transport

import (
	"bytes"
	"context"
	"net/http"

	"github.com/a-h/templ"
)

type HTTPError struct {
	Status          int
	Message         string
	ErrorComponent  templ.Component
	ResponseHeaders map[string]string
}

// Writes message and status of HTTPError to the response
// Returns false if httpError is nil, true otherwise
func SendHTTPError(httpError *HTTPError) Response {
	return Response{
		Headers:         httpError.ResponseHeaders,
		StatusCode:      httpError.Status,
		IsBase64Encoded: false,
		Body:            httpError.Message,
	}
}

func DisplayHTTPError(ctx context.Context, httpError *HTTPError) Response {
	var buf bytes.Buffer
	err := httpError.ErrorComponent.Render(ctx, &buf)
	if err != nil {
		SendHTTPError(&HTTPError{
			Status:         http.StatusInternalServerError,
			Message:        err.Error(),
			ErrorComponent: nil,
		})
	}
	return Response{
		Headers:         httpError.ResponseHeaders,
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}
}
