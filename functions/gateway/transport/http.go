package transport

import (
	"log"
	"net/http"
)

// NOTE: `err` is passed in and logged if status is 400 or greater, but msg
func SendHtmlRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
	msg := string(body)
	if (status >= 400) {
		msg = "ERR: "+msg
		log.Println(msg+ " || Internal error msg: "+err.Error())
	}
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
	msg := string(body)
	if (status >= 400) {
		msg = "ERR: "+msg
		log.Println(msg)
	}

	w.WriteHeader(status)
	w.Write([]byte(msg))

	return http.HandlerFunc(nil)
}
