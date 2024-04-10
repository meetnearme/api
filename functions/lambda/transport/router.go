package transport

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Request = events.APIGatewayV2HTTPRequest
type Response = events.APIGatewayV2HTTPResponse

type lambdaHandlerFunc func(ctx context.Context, r Request, db *dynamodb.Client) (Response, error)

type Middleware func(ctx context.Context, r Request) (context.Context, Request, error)

type route struct {
	method       string
	pattern      *regexp.Regexp
	innerHandler lambdaHandlerFunc
	paramKeys    []string
	middleware   []Middleware
}

type Router struct {
	routes []*route
}

func GetRouter() *Router {
	return &Router{routes: []*route{}}
}

func (r *Router) addRoute(method string, pathDef string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	// handle path parameters
	pathParamMatcher := regexp.MustCompile(":([a-zA-Z]+)")
	matches := pathParamMatcher.FindAllStringSubmatch(pathDef, -1)
	paramKeys := []string{}
	pattern := pathDef
	if len(matches) > 0 {
		// replace path parameter definition with regex pattern to capture any string
		pattern = pathParamMatcher.ReplaceAllLiteralString(pathDef, "([^/]+)")
		// store the names of path parameters, to later be used as context keys
		for i := 0; i < len(matches); i++ {
			paramKeys = append(paramKeys, matches[i][1])
		}
	}
	// check for duplicates: same method and regex pattern
	regex := regexp.MustCompile("^" + pattern + "$")
	for _, route := range r.routes {
		if route.method == method && route.pattern.String() == regex.String() {
			panic(fmt.Sprintf("Route already exists: %s %s", method, pathDef))
		}
	}

	newRoute := &route{
		method,
		regex,
		handler,
		paramKeys,
		middleware,
	}
	r.routes = append(r.routes, newRoute)
	return newRoute
}

func (r *Router) GET(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodGet, pattern, handler, middleware...)
}

func (r *Router) POST(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodPost, pattern, handler, middleware...)
}

func (r *Router) PUT(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodPut, pattern, handler, middleware...)
}

func (r *Router) PATCH(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodPatch, pattern, handler, middleware...)
}

func (r *Router) DELETE(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodDelete, pattern, handler, middleware...)
}

func (r *Router) OPTIONS(pattern string, handler lambdaHandlerFunc, middleware ...Middleware) *route {
	return r.addRoute(http.MethodOptions, pattern, handler, middleware...)
}

func (r *Router) ServeHTTP(ctx context.Context, req Request, db *dynamodb.Client) (Response, error) {
	var allow []string
	for _, route := range r.routes {
		reqPath := req.RequestContext.HTTP.Path
		reqMethod := req.RequestContext.HTTP.Method

		matches := route.pattern.FindStringSubmatch(reqPath)

		if len(matches) > 0 {
			if reqMethod != route.method {
				allow = append(allow, route.method)
				continue
			}

			values := matches[1:]
			if len(values) != len(route.paramKeys) {
				message := "unexpected number of path parameters in request"
				// Log Error message
				return SendClientError(http.StatusBadRequest, message)
			}
			for idx, key := range route.paramKeys {
				ctx = context.WithValue(ctx, key, values[idx])
			}

			return route.handler(ctx, req, db)
		}
	}
	if len(allow) > 0 {
		return SendClientError(http.StatusMethodNotAllowed, "")
	}
	return SendClientError(http.StatusNotFound, "")
}

// A wrapper around a route's handler for request middleware
func (r *route) handler(ctx context.Context, req Request, db *dynamodb.Client) (Response, error) {
	// Middleware
	var error error
	for _, middleware := range r.middleware {
		if ctx, req, error = middleware(ctx, req); error != nil {
			return SendServerError(error)
		}
	}

	return r.innerHandler(ctx, req, db)
}
