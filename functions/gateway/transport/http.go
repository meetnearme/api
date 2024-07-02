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


func SendServerRes(w http.ResponseWriter, body []byte, status int) http.HandlerFunc {
	msg := string(body)
	if (status >= 400) {
		msg = "ERR: "+msg
	}
	log.Println(msg)

	w.WriteHeader(status)
	w.Write([]byte(msg))

	return http.HandlerFunc(nil)
}
