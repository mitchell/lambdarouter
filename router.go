package lambdarouter

import (
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

// APIGRequest is used as the input of handler functions.
// The Claims, Path, and QryStr will be populated by the the APIGatewayProxyRequest.
// The Request itself is also passed through if you need further access.
type APIGRequest struct {
	Claims  map[string]interface{}
	Path    map[string]string
	QryStr  map[string]string
	Request *events.APIGatewayProxyRequest
}

// APIGResponse is used as the output of handler functions.
// Populate Status and Body with your http response or populate Err with your error.
type APIGResponse struct {
	Status int
	Body   []byte
	Err    error
}

// APIGHandler is the interface a handler function must implement to be used
// with Get, Post, Put, Patch, and Delete.
type APIGHandler func(req *APIGRequest, res *APIGResponse)

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
func (r *APIGRouter) Get(route string, handler APIGHandler) {
	r.addEndpoint(get, route, handler)
}

// Post creates a new post endpoint.
func (r *APIGRouter) Post(route string, handler APIGHandler) {
	r.addEndpoint(post, route, handler)
}

// Put creates a new put endpoint.
func (r *APIGRouter) Put(route string, handler APIGHandler) {
	r.addEndpoint(put, route, handler)
}

// Patch creates a new patch endpoint
func (r *APIGRouter) Patch(route string, handler APIGHandler) {
	r.addEndpoint(patch, route, handler)
}

// Delete creates a new delete endpoint.
func (r *APIGRouter) Delete(route string, handler APIGHandler) {
	r.addEndpoint(delete, route, handler)
}

// Respond returns an APIGatewayProxyResponse to respond to the lambda request.
func (r *APIGRouter) Respond() events.APIGatewayProxyResponse {
	var (
		handlerInterface interface{}
		ok               bool

		endpointTree = r.endpoints[r.request.HTTPMethod]
		path         = strings.TrimPrefix(r.request.Path, "/"+r.svcprefix)
		response     = events.APIGatewayProxyResponse{}
	)

	for k := range r.params {
		p := strings.TrimPrefix(k, "{")
		p = strings.TrimSuffix(p, "}")
		if r.request.PathParameters[p] != "" {
			path = strings.Replace(path, r.request.PathParameters[p], k, -1)
		}
	}

	if handlerInterface, ok = endpointTree.Get(path); !ok {
		respbody, _ := json.Marshal(map[string]string{"error": "no route matching path found"})

		response.StatusCode = http.StatusNotFound
		response.Body = string(respbody)
		return response
	}

	handler := handlerInterface.(APIGHandler)

	req := &APIGRequest{
		Path:    r.request.PathParameters,
		QryStr:  r.request.QueryStringParameters,
		Request: r.request,
	}
	if r.request.RequestContext.Authorizer["claims"] != nil {
		req.Claims = r.request.RequestContext.Authorizer["claims"].(map[string]interface{})
	}
	res := &APIGResponse{}

	handler(req, res)
	status, respbody, err := res.deconstruct()

	if err != nil {
		respbody, _ := json.Marshal(map[string]string{"error": err.Error()})
		if strings.Contains(err.Error(), "record not found") {
			status = 204
		} else if status < 400 {
			status = 400
		}

		response.StatusCode = status
		response.Body = string(respbody)
		return response
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

func (res *APIGResponse) deconstruct() (int, []byte, error) {
	return res.Status, res.Body, res.Err
}

func (r *APIGRouter) addEndpoint(method string, route string, handler APIGHandler) {
	if _, overwrite := r.endpoints[method].Insert(route, handler); overwrite {
		panic("endpoint already existent")
	}

	rtearr := stripSlashesAndSplit(route)
	for _, v := range rtearr {
		if strings.HasPrefix(v, "{") {
			r.params[v] = "" // adding params as keys with {brackets}
		}
	}

}
