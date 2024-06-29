package transport

import (
	"log"
	"net/http"
)

func SendHtmlSuccess(w http.ResponseWriter, body []byte, status int) http.HandlerFunc {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write(body)

	return http.HandlerFunc(nil)
}

func SendHtmlError(w http.ResponseWriter, body []byte, status int) http.HandlerFunc {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write(body)

	return http.HandlerFunc(nil)
}


func SendServerRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
	log.Println(err.Error())

	w.WriteHeader(status)
	w.Write(body)

	return http.HandlerFunc(nil)
}
