package transport

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
)

// NOTE: `err` is passed in and logged if status is 400 or greater, but msg
func SendHtmlRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := string(body)
		if status >= 400 {
			msg = "ERR: " + msg
			log.Println(msg + " || Internal error msg: " + err.Error())
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
		var buf bytes.Buffer
		ctx := r.Context()
		apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest).RequestContext
		requestID := apiGwV2Req.RequestID
		errorPartial := partials.ErrorHTML(body, fmt.Sprint(requestID))
		err := errorPartial.Render(r.Context(), &buf)
		if err != nil {
			log.Println("Error rendering error partial:", err)
		}
		log.Println("ERR ("+fmt.Sprint(status)+"): ", string(body))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write(buf.Bytes())
	}
}

func SendServerRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
	msg := string(body)
	if status >= 400 {
		msg = "ERR: " + msg
		log.Println(msg)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(msg))

	return http.HandlerFunc(nil)
}
