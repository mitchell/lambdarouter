package lambdarouter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	radix "github.com/armon/go-radix"
	"github.com/aws/aws-lambda-go/events"
)

const (
	post   = http.MethodPost
	get    = http.MethodGet
	put    = http.MethodPut
	patch  = http.MethodPatch
	delete = http.MethodDelete
)

// APIGContext is used as the input and output of handler functions.
// The Body, Claims, Path, and QryStr will be populated by the the APIGatewayProxyRequest.
// The Request itself is also passed through if you need further access.
// Fill the Status and Body, or Status and Error to respond.
type APIGContext struct {
	Claims  map[string]interface{}
	Context context.Context
	Path    map[string]string
	QryStr  map[string]string
	Request *events.APIGatewayProxyRequest
	Status  int
	Body    []byte
	Err     error
}

// APIGHandler is the type a handler function must implement to be used
// with Get, Post, Put, Patch, and Delete.
type APIGHandler func(ctx *APIGContext)

// APIGMiddleware is the function type that must me implemented to be appended
// to a route or to the APIGRouterConfig.Middleware attribute.
type APIGMiddleware func(APIGHandler) APIGHandler

// APIGRouter is the object that handlers build upon and is used in the end to respond.
type APIGRouter struct {
	request    *events.APIGatewayProxyRequest
	routes     map[string]*radix.Tree
	params     map[string]interface{}
	prefix     string
	headers    map[string]string
	context    context.Context
	middleware []APIGMiddleware
}

// APIGRouterConfig is used as the input to NewAPIGRouter, request is your incoming
// apig request and prefix will be stripped of all incoming request paths. Headers
// will be sent with all responses.
type APIGRouterConfig struct {
	Context    context.Context
	Request    *events.APIGatewayProxyRequest
	Prefix     string
	Headers    map[string]string
	Middleware []APIGMiddleware
}

// NOTE: Begin router methods.

// NewAPIGRouter creates a new router using the given router config.
func NewAPIGRouter(cfg *APIGRouterConfig) *APIGRouter {
	return &APIGRouter{
		request: cfg.Request,
		routes: map[string]*radix.Tree{
			post:   radix.New(),
			get:    radix.New(),
			put:    radix.New(),
			patch:  radix.New(),
			delete: radix.New(),
		},
		params:     map[string]interface{}{},
		prefix:     cfg.Prefix,
		headers:    cfg.Headers,
		context:    cfg.Context,
		middleware: cfg.Middleware,
	}
}

// Get creates a new get endpoint.
func (r *APIGRouter) Get(path string, handler APIGHandler, middleware ...APIGMiddleware) {
	functions := routeFunctions{
		handler:    handler,
		middleware: middleware,
	}
	r.addRoute(get, path, functions)
}

// Post creates a new post endpoint.
func (r *APIGRouter) Post(path string, handler APIGHandler, middleware ...APIGMiddleware) {
	functions := routeFunctions{
		handler:    handler,
		middleware: middleware,
	}
	r.addRoute(post, path, functions)
}

// Put creates a new put endpoint.
func (r *APIGRouter) Put(path string, handler APIGHandler, middleware ...APIGMiddleware) {
	functions := routeFunctions{
		handler:    handler,
		middleware: middleware,
	}
	r.addRoute(put, path, functions)
}

// Patch creates a new patch endpoint
func (r *APIGRouter) Patch(path string, handler APIGHandler, middleware ...APIGMiddleware) {
	functions := routeFunctions{
		handler:    handler,
		middleware: middleware,
	}
	r.addRoute(patch, path, functions)
}

// Delete creates a new delete endpoint.
func (r *APIGRouter) Delete(path string, handler APIGHandler, middleware ...APIGMiddleware) {
	functions := routeFunctions{
		handler:    handler,
		middleware: middleware,
	}
	r.addRoute(delete, path, functions)
}

// Respond returns an APIGatewayProxyResponse to respond to the lambda request.
func (r *APIGRouter) Respond() events.APIGatewayProxyResponse {
	var (
		ok             bool
		respbytes      []byte
		response       events.APIGatewayProxyResponse
		routeInterface interface{}

		routeTrie = r.routes[r.request.HTTPMethod]
		path      = strings.TrimPrefix(r.request.Path, r.prefix)
		splitPath = stripSlashesAndSplit(path)
		ctx       = &APIGContext{
			Body:    []byte(r.request.Body),
			Path:    r.request.PathParameters,
			QryStr:  r.request.QueryStringParameters,
			Request: r.request,
			Context: r.context,
		}
	)
	if r.request.RequestContext.Authorizer["claims"] != nil {
		ctx.Claims = r.request.RequestContext.Authorizer["claims"].(map[string]interface{})
	}

	for p := range r.params {
		if r.request.PathParameters[p] != "" {
			pval := r.request.PathParameters[p]
			for i, v := range splitPath {
				if v == pval {
					splitPath[i] = "{" + p + "}"
					break
				}
			}
		}
	}
	path = "/" + strings.Join(splitPath, "/")

	if routeInterface, ok = routeTrie.Get(path); !ok {
		respbytes, _ = json.Marshal(map[string]string{"error": "no route matching path found"})

		response.StatusCode = http.StatusNotFound
		response.Body = string(respbytes)
		response.Headers = r.headers
		return response
	}

	functions := routeInterface.(routeFunctions)

	for _, m := range functions.middleware {
		functions.handler = m(functions.handler)
	}
	for _, m := range r.middleware {
		functions.handler = m(functions.handler)
	}
	functions.handler(ctx)

	response.StatusCode = ctx.Status
	response.Body = string(ctx.Body)
	response.Headers = r.headers
	return response
}

// NOTE: Begin helper functions.
type routeFunctions struct {
	handler    APIGHandler
	middleware []APIGMiddleware
}

func stripSlashesAndSplit(s string) []string {
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, "/")
	return strings.Split(s, "/")
}

func (r *APIGRouter) addRoute(method string, path string, functions routeFunctions) {
	if _, overwrite := r.routes[method].Insert(path, functions); overwrite {
		panic("endpoint already existent")
	}

	rtearr := stripSlashesAndSplit(path)
	for _, v := range rtearr {
		if strings.HasPrefix(v, "{") {
			v = strings.TrimPrefix(v, "{")
			v = strings.TrimSuffix(v, "}")
			r.params[v] = nil // adding params as *unique* keys
		}
	}

}
