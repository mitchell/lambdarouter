package lambdarouter

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	radix "github.com/armon/go-radix"
	"github.com/aws/aws-lambda-go/events"
)

// Method constants.
const (
	POST   = http.MethodPost
	GET    = http.MethodGet
	PUT    = http.MethodPut
	DELETE = http.MethodDelete
)

// HandlerRequest ...
type HandlerRequest struct {
	Claims  map[string]string
	Path    map[string]string
	QryStr  map[string]string
	Request *events.APIGatewayProxyRequest
}

// HandlerResponse ...
type HandlerResponse struct {
	Status int
	Body   []byte
	Err    error
}

// Handler ...
type Handler func(req *HandlerRequest, res *HandlerResponse)

// Router ...
type Router struct {
	request   *events.APIGatewayProxyRequest
	endpoints map[string]*radix.Tree
	params    map[string]string
	svcprefix string
}

// NOTE: Begin router methods.

// New ...
func New(r *events.APIGatewayProxyRequest, svcprefix string) *Router {
	return &Router{
		request: r,
		endpoints: map[string]*radix.Tree{
			POST:   radix.New(),
			GET:    radix.New(),
			PUT:    radix.New(),
			DELETE: radix.New(),
		},
		params:    map[string]string{},
		svcprefix: svcprefix,
	}
}

// Get ...
func (r *Router) Get(route string, handler Handler) {
	r.addEndpoint(GET, route, handler)
}

// Post ...
func (r *Router) Post(route string, handler Handler) {
	r.addEndpoint(POST, route, handler)
}

// Put ...
func (r *Router) Put(route string, handler Handler) {
	r.addEndpoint(PUT, route, handler)
}

// Delete ...
func (r *Router) Delete(route string, handler Handler) {
	r.addEndpoint(DELETE, route, handler)
}

// Respond ...
func (r *Router) Respond() events.APIGatewayProxyResponse {
	var (
		handlerInterface interface{}
		ok               bool

		endpointTree = r.endpoints[r.request.HTTPMethod]
		path         = strings.TrimPrefix(r.request.Path, "/"+r.svcprefix)
		response     = events.APIGatewayProxyResponse{}
	)
	log.Printf("path: %+v", path)

	for k := range r.params {
		p := strings.TrimPrefix(k, "{")
		p = strings.TrimSuffix(p, "}")
		if r.request.PathParameters[p] != "" {
			path = strings.Replace(path, r.request.PathParameters[p], k, -1)
		}
	}
	log.Printf("path: %+v", path)

	if handlerInterface, ok = endpointTree.Get(path); !ok {
		respbody, _ := json.Marshal(map[string]string{"error": "no route matching path found"})

		response.StatusCode = http.StatusNotFound
		response.Body = string(respbody)
		return response
	}

	handler := handlerInterface.(Handler)

	req := &HandlerRequest{
		Claims:  r.request.RequestContext.Authorizer["claims"].(map[string]string),
		Path:    r.request.PathParameters,
		QryStr:  r.request.QueryStringParameters,
		Request: r.request,
	}
	res := &HandlerResponse{}

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

func (res *HandlerResponse) deconstruct() (int, []byte, error) {
	return res.Status, res.Body, res.Err
}

func (r *Router) addEndpoint(method string, route string, handler Handler) {
	if _, overwrite := r.endpoints[method].Insert(route, handler); overwrite {
		panic("endpoint already existent")
	}

	rtearr := stripSlashesAndSplit(route)
	for _, v := range rtearr {
		if strings.HasPrefix(v, "{") {
			r.params[v] = "" // adding params as keys with {brackets}
		}
	}

	log.Printf("router: %+v", *r)
}
