package transport

import (
	"log"
	"net/http"
)

// NOTE: `err` is passed in and logged if status is 400 or greater, but msg
func SendHtmlRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Println("In SendHtmlRes")
        defer log.Println("About to leave Send HTML")
        msg := string(body)
        if (status >= 400) {
            log.Println("Error occured:", err)
            msg = "ERR: "+msg
            log.Println(msg+ " || Internal error msg: "+err.Error())
            body = []byte(msg) 
        }
        w.Header().Set("Content-Type", "text/html")
        w.WriteHeader(status)
        _, writeErr := w.Write(body)
        if writeErr != nil {
            log.Println("Error writing response:", writeErr)
        }
    }
}

func SendHtmlError(w http.ResponseWriter, body []byte, status int) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.WriteHeader(status)
        w.Write(body)
    }
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
