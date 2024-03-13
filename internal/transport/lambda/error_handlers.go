package transport

import (
	"log"
	"net/http"
)

func SendServerError(err error) (Response, error) {
	log.Println(err.Error())

	return Response{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func SendClientError(status int) (Response, error) {
	return Response{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}
