package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/constants"
	"github.com/meetnearme/api/functions/gateway/templates/pages"
	"github.com/meetnearme/api/functions/gateway/templates/partials"
	"github.com/meetnearme/api/functions/gateway/types"
)

// NOTE: `err` is passed in and logged if status is 400 or greater, but msg
func SendHtmlRes(w http.ResponseWriter, body []byte, status int, mode string, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers BEFORE any other headers or WriteHeader calls
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, HX-Request, HX-Trigger, HX-Trigger-Name, HX-Target, HX-Current-URL")

		msg := string(body)
		if status >= 400 {
			internalMsg := "ERR: " + msg
			if err != nil {
				log.Println(internalMsg + " || Internal error msg: " + err.Error())
			} else {
				log.Println(internalMsg + " || Internal error msg: <nil>")
			}
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
		userInfo := constants.UserInfo{}
		ctx := r.Context()
		if _, ok := ctx.Value("userInfo").(constants.UserInfo); ok {
			userInfo = ctx.Value("userInfo").(constants.UserInfo)
		}

		requestID := ""
		if apiGwV2ReqRaw := ctx.Value(constants.ApiGwV2ReqKey); apiGwV2ReqRaw != nil {
			if apiGwV2Req, ok := apiGwV2ReqRaw.(events.APIGatewayV2HTTPRequest); ok {
				requestID = apiGwV2Req.RequestContext.RequestID
			} else {
				log.Println("Warning: ApiGwV2ReqKey value is not of expected type")
			}
		} else {
			log.Println("Warning: No ApiGwV2ReqKey found in context")
		}

		errorPartial := pages.ErrorPage(body, requestID, hideError)
		layoutTemplate := pages.Layout(constants.SitePages["home"], userInfo, errorPartial, types.Event{}, false, ctx, []string{})
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
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// Include all headers that HTMX/json-enc might send
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, HX-Request, HX-Trigger, HX-Trigger-Name, HX-Target, HX-Current-URL")

		var buf bytes.Buffer
		ctx := r.Context()
		requestID := ""
		if raw := ctx.Value(constants.ApiGwV2ReqKey); raw != nil {
			if req, ok := raw.(events.APIGatewayV2HTTPRequest); ok {
				requestID = req.RequestContext.RequestID
			} else {
				log.Println("Warning: ApiGwV2ReqKey value is not of expected type")
			}
		} else {
			log.Println("Warning: No ApiGwV2ReqKey found in context")
		}
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

func SendHtmlErrorText(body []byte, status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		ctx := r.Context()
		requestID := ""
		if raw := ctx.Value(constants.ApiGwV2ReqKey); raw != nil {
			if req, ok := raw.(events.APIGatewayV2HTTPRequest); ok {
				requestID = req.RequestContext.RequestID
			} else {
				log.Println("Warning: ApiGwV2ReqKey value is not of expected type")
			}
		} else {
			log.Println("Warning: No ApiGwV2ReqKey found in context")
		}
		errorPartial := partials.ErrorHTMLText(body, fmt.Sprint(requestID))
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

// This is used for embed endpoints that need to be accessible from external websites
func SetCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	// For development, allow all origins. In production, this should be restricted
	// to specific domains or use environment variable to control allowed origins
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	// Include all headers that HTMX/json-enc might send
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, HX-Request, HX-Trigger, HX-Trigger-Name, HX-Target, HX-Current-URL")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
}
