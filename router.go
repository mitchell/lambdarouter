package lambdarouter

import (
	"context"
	"encoding/json"
	"log"
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

// APIGHandler is the interface a handler function must implement to be used
// with Get, Post, Put, Patch, and Delete.
type APIGHandler func(ctx *APIGContext)

// APIGRouter is the object that handlers build upon and is used in the end to respond.
type APIGRouter struct {
	request   *events.APIGatewayProxyRequest
	endpoints map[string]*radix.Tree
	params    map[string]interface{}
	prefix    string
	headers   map[string]string
	context   context.Context
}

// APIGRouterConfig is used as the input to NewAPIGRouter, request is your incoming
// apig request and prefix will be stripped of all incoming request paths. Headers
// will be sent with all responses.
type APIGRouterConfig struct {
	Context context.Context
	Request *events.APIGatewayProxyRequest
	Prefix  string
	Headers map[string]string
}

// NOTE: Begin router methods.

// NewAPIGRouter creates a new router using the given router config.
func NewAPIGRouter(cfg *APIGRouterConfig) *APIGRouter {
	return &APIGRouter{
		request: cfg.Request,
		endpoints: map[string]*radix.Tree{
			post:   radix.New(),
			get:    radix.New(),
			put:    radix.New(),
			patch:  radix.New(),
			delete: radix.New(),
		},
		params:  map[string]interface{}{},
		prefix:  cfg.Prefix,
		headers: cfg.Headers,
		context: cfg.Context,
	}
}

// Get creates a new get endpoint.
func (r *APIGRouter) Get(route string, handlers ...APIGHandler) {
	r.addEndpoint(get, route, handlers)
}

// Post creates a new post endpoint.
func (r *APIGRouter) Post(route string, handlers ...APIGHandler) {
	r.addEndpoint(post, route, handlers)
}

// Put creates a new put endpoint.
func (r *APIGRouter) Put(route string, handlers ...APIGHandler) {
	r.addEndpoint(put, route, handlers)
}

// Patch creates a new patch endpoint
func (r *APIGRouter) Patch(route string, handlers ...APIGHandler) {
	r.addEndpoint(patch, route, handlers)
}

// Delete creates a new delete endpoint.
func (r *APIGRouter) Delete(route string, handlers ...APIGHandler) {
	r.addEndpoint(delete, route, handlers)
}

// Respond returns an APIGatewayProxyResponse to respond to the lambda request.
func (r *APIGRouter) Respond() events.APIGatewayProxyResponse {
	var (
		handlersInterface interface{}
		ok                bool
		status            int
		respbody          []byte
		err               error

		endpointTree = r.endpoints[r.request.HTTPMethod]
		path         = strings.TrimPrefix(r.request.Path, r.prefix)
		inPath       = path
		response     = events.APIGatewayProxyResponse{}
		splitPath    = stripSlashesAndSplit(path)
	)

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

	if handlersInterface, ok = endpointTree.Get(path); !ok {
		respbody, _ = json.Marshal(map[string]string{"error": "no route matching path found"})

		response.StatusCode = http.StatusNotFound
		response.Body = string(respbody)
		response.Headers = r.headers
		return response
	}

	handlers := handlersInterface.([]APIGHandler)

	for _, handler := range handlers {
		ctx := &APIGContext{
			Body:    []byte(r.request.Body),
			Path:    r.request.PathParameters,
			QryStr:  r.request.QueryStringParameters,
			Request: r.request,
			Context: r.context,
		}
		if r.request.RequestContext.Authorizer["claims"] != nil {
			ctx.Claims = r.request.RequestContext.Authorizer["claims"].(map[string]interface{})
		}

		handler(ctx)
		status, respbody, err = ctx.respDeconstruct()

		if err != nil {
			respbody, _ = json.Marshal(map[string]string{"error": err.Error()})
			if strings.Contains(err.Error(), "record not found") {
				status = 204
			} else if status != 204 && status < 400 {
				status = 400
			}

			log.Printf("%v %v %v error: %v \n", r.request.HTTPMethod, inPath, status, err.Error())
			log.Println("error causing body: " + r.request.Body)
			response.StatusCode = status
			response.Body = string(respbody)
			response.Headers = r.headers
			return response
		}
	}

	response.StatusCode = status
	response.Body = string(respbody)
	response.Headers = r.headers
	return response
}

// NOTE: Begin helper functions.
func stripSlashesAndSplit(s string) []string {
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, "/")
	return strings.Split(s, "/")
}

func (ctx *APIGContext) respDeconstruct() (int, []byte, error) {
	return ctx.Status, ctx.Body, ctx.Err
}

func (r *APIGRouter) addEndpoint(method string, route string, handlers []APIGHandler) {
	if _, overwrite := r.endpoints[method].Insert(route, handlers); overwrite {
		panic("endpoint already existent")
	}

	rtearr := stripSlashesAndSplit(route)
	for _, v := range rtearr {
		if strings.HasPrefix(v, "{") {
			v = strings.TrimPrefix(v, "{")
			v = strings.TrimSuffix(v, "}")
			r.params[v] = nil // adding params as *unique* keys
		}
	}

}
