package transport

import (
	"bytes"
	"context"
	"log"
	"net/http"

	"github.com/meetnearme/api/functions/gateway/templates/partials"
)

func SendServerError(err error) (Response, error) {
	log.Println(err.Error())

	return Response{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func SendClientError(status int, message string) (Response, error) {
	return Response{
		StatusCode: status,
		Body:       message,
	}, nil
}

func SendHTMLError(err error, ctx context.Context, req Request) (Response, error) {
	layoutTemplate := partials.ErrorHTML(err, req.RequestContext.RequestID)
	var buf bytes.Buffer
	err = layoutTemplate.Render(ctx, &buf)
	if err != nil {
		return SendServerError(err)
	}

	return Response{
		Headers: map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
		Body: buf.String(),
	}, nil
}
