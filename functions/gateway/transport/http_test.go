package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/meetnearme/api/functions/gateway/helpers"
)

func TestSendHtmlRes(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		status         int
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			body:           []byte("<h1>Hello</h1>"),
			status:         http.StatusOK,
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "<h1>Hello</h1>",
		},
		{
			name:           "Error",
			body:           []byte("Not Found"),
			status:         http.StatusNotFound,
			err:            errors.New("page not found"),
			expectedStatus: http.StatusOK,
			expectedBody:   "Not Found</h1>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := SendHtmlRes(rr, tt.body, tt.status, tt.err)

			req := httptest.NewRequest("GET", "/", nil)
			// Set up context with APIGatewayV2HTTPRequest
			ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
				RequestContext: events.APIGatewayV2HTTPRequestContext{
					RequestID: "test-request-id",
				},
			})
			req = req.WithContext(ctx)

			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
			}
			if contentType := rr.Header().Get("Content-Type"); contentType != "text/html" {
				t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/html")
			}
		})
	}
}

func TestSendHtmlErrorPartial(t *testing.T) {
	rr := httptest.NewRecorder()
	body := []byte("This error has been logged with Request ID: ")
	status := http.StatusOK

	req := httptest.NewRequest("GET", "/", nil)
	// Set up context with APIGatewayV2HTTPRequest
	ctx := context.WithValue(req.Context(), helpers.ApiGwV2ReqKey, events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "test-request-id",
		},
	})
	req = req.WithContext(ctx)

	handler := SendHtmlErrorPartial(rr, body, status)
	handler.ServeHTTP(rr, req)

	if rr.Code != status {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, status)
	}

	if !strings.Contains(rr.Body.String(), string(body)) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), string(body))
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/html")
	}
}

func TestSendServerRes(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		status         int
		err            error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			body:           []byte("Success"),
			status:         http.StatusOK,
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "Success",
		},
		{
			name:           "Error",
			body:           []byte("Bad Request"),
			status:         http.StatusBadRequest,
			err:            errors.New("bad request"),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ERR: Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := SendServerRes(rr, tt.body, tt.status, tt.err)
			if handler != nil {
				handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
			}

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}
