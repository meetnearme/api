package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/types"
)

// NOTE: `err` is passed in and logged if status is 400 or greater, but msg
func SendHtmlRes(w http.ResponseWriter, body []byte, status int, mode string, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := string(body)
		if status >= 400 {
			internalMsg := "ERR: " + msg
			log.Println(internalMsg + " || Internal error msg: " + err.Error())
			body = []byte(msg)
			if mode == "partial" {
				handler := SendHtmlErrorPartial(body, status)
				handler.ServeHTTP(w, r)
			} else {
				handler := SendHtmlErrorPage(body, status, false)
				handler.ServeHTTP(w, r)
			}
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		_, writeErr := w.Write(body)
		if writeErr != nil {
			log.Println("ERR:Error writing response:", writeErr)
		}
	}
}

func SendHtmlErrorPage(body []byte, status int, hideError bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userInfo := helpers.UserInfo{}
		ctx := r.Context()
		if _, ok := ctx.Value("userInfo").(helpers.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(helpers.UserInfo)
		}

		requestID := ""
		if apiGwV2ReqRaw := ctx.Value(helpers.ApiGwV2ReqKey); apiGwV2ReqRaw != nil {
			if apiGwV2Req, ok := apiGwV2ReqRaw.(events.APIGatewayV2HTTPRequest); ok {
				requestID = apiGwV2Req.RequestContext.RequestID
			} else {
				log.Println("Warning: ApiGwV2ReqKey value is not of expected type")
			}
		} else {
			log.Println("Warning: No ApiGwV2ReqKey found in context")
		}

		errorPartial := pages.ErrorPage(body, requestID, hideError)
		layoutTemplate := pages.Layout(helpers.SitePages["home"], userInfo, errorPartial, types.Event{}, ctx, []string{})
		var buf bytes.Buffer
		err := layoutTemplate.Render(ctx, &buf)
		if err != nil {
			log.Println("Error rendering HTML error page:", err)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

func SendHtmlErrorPartial(body []byte, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		ctx := r.Context()
		apiGwV2Req := ctx.Value(helpers.ApiGwV2ReqKey).(events.APIGatewayV2HTTPRequest).RequestContext
		requestID := apiGwV2Req.RequestID
		errorPartial := partials.ErrorHTMLAlert(body, fmt.Sprint(requestID))
		err := errorPartial.Render(r.Context(), &buf)
		if err != nil {
			log.Println("Error rendering error partial:", err)
		}
		log.Println("ERR ("+fmt.Sprint(status)+"): ", string(body))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

// Helper function to create error JSON
func createErrorJSON(message []byte) []byte {
	messageString := string(message)
	errorResponse := struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}{
		Error: struct {
			Message string `json:"message"`
		}{
			Message: messageString,
		},
	}

	jsonBytes, err := json.Marshal(errorResponse)
	if err != nil {
		// Fallback in case marshaling fails
		return []byte(`{"error":{"message":"Internal server error"}}`)
	}
	return jsonBytes
}

func SendServerRes(w http.ResponseWriter, body []byte, status int, err error) http.HandlerFunc {
	msg := string(body)
	if status >= 400 {
		msg = "ERR: " + msg
		msg = string(createErrorJSON([]byte(msg)))
		log.Println(msg)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(msg))

	// NOTE: we don't want to return `nil` to a handler function in general, but we need to clean
	// up a bunch of other code connected to this

	return http.HandlerFunc(nil)
}
