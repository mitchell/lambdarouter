package lambdarouter

import (
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
	params    map[string]string
	svcprefix string
}

// NOTE: Begin router methods.

// NewAPIGRouter creates a new router using the request and a prefix to strip from your incoming requests.
func NewAPIGRouter(r *events.APIGatewayProxyRequest, svcprefix string) *APIGRouter {
	return &APIGRouter{
		request: r,
		endpoints: map[string]*radix.Tree{
			post:   radix.New(),
			get:    radix.New(),
			put:    radix.New(),
			patch:  radix.New(),
			delete: radix.New(),
		},
		params:    map[string]string{},
		svcprefix: svcprefix,
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
		path         = strings.TrimPrefix(r.request.Path, "/"+r.svcprefix)
		response     = events.APIGatewayProxyResponse{}
		splitPath    = stripSlashesAndSplit(path)
	)

	for k := range r.params {
		pname := strings.TrimPrefix(k, "{")
		pname = strings.TrimSuffix(pname, "}")
		if r.request.PathParameters[pname] != "" {
			pval := r.request.PathParameters[pname]
			for i, v := range splitPath {
				if v == pval {
					splitPath[i] = k
				}
			}

		}
	}
	path = "/" + strings.Join(splitPath, "/")

	if handlersInterface, ok = endpointTree.Get(path); !ok {
		respbody, _ := json.Marshal(map[string]string{"error": "no route matching path found"})

		response.StatusCode = http.StatusNotFound
		response.Body = string(respbody)
		return response
	}

	handlers := handlersInterface.([]APIGHandler)

	for _, handler := range handlers {
		ctx := &APIGContext{
			Body:    []byte(r.request.Body),
			Path:    r.request.PathParameters,
			QryStr:  r.request.QueryStringParameters,
			Request: r.request,
		}
		if r.request.RequestContext.Authorizer["claims"] != nil {
			ctx.Claims = r.request.RequestContext.Authorizer["claims"].(map[string]interface{})
		}

		handler(ctx)
		status, respbody, err = ctx.respDeconstruct()

		if err != nil {
			respbody, _ := json.Marshal(map[string]string{"error": err.Error()})
			if strings.Contains(err.Error(), "record not found") {
				status = 204
			} else if status < 400 {
				status = 400
			}

			log.Printf("%v error: %v", status, err.Error())
			response.StatusCode = status
			response.Body = string(respbody)
			return response
		}
	}

	response.StatusCode = status
	response.Body = string(respbody)
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
			r.params[v] = "" // adding params as keys with {brackets}
		}
	}

}
