package lambdarouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	iradix "github.com/hashicorp/go-immutable-radix"
)

// Router holds the defined routes for use upon invocation.
type Router struct {
	events *iradix.Tree
	prefix string
}

// New initializes an empty router. The prefix parameter may be of any length.
func New(prefix string) Router {
	if len(prefix) > 0 {
		if prefix[0] != '/' {
			prefix = "/" + prefix
		}
		if prefix[len(prefix)-1] != '/' {
			prefix += "/"
		}
	}

	return Router{
		events: iradix.New(),
		prefix: prefix,
	}
}

// Get adds a new GET method route to the router. The path parameter is the route path you wish to
// define. The handler parameter is a lambda.Handler to invoke if an incoming path matches the
// route.
func (r *Router) Get(path string, handler lambda.Handler) {
	r.addEvent(prepPath(http.MethodGet, r.prefix, path), event{h: handler})
}

// Post adds a new POST method route to the router. The path parameter is the route path you wish to
// define. The handler parameter is a lambda.Handler to invoke if an incoming path matches the
// route.
func (r *Router) Post(path string, handler lambda.Handler) {
	r.addEvent(prepPath(http.MethodPost, r.prefix, path), event{h: handler})
}

// Put adds a new PUT method route to the router. The path parameter is the route path you wish to
// define. The handler parameter is a lambda.Handler to invoke if an incoming path matches the
// route.
func (r *Router) Put(path string, handler lambda.Handler) {
	r.addEvent(prepPath(http.MethodPut, r.prefix, path), event{h: handler})
}

// Patch adds a new PATCH method route to the router. The path parameter is the route path you wish
// to define. The handler parameter is a lambda.Handler to invoke if an incoming path matches the
// route.
func (r *Router) Patch(path string, handler lambda.Handler) {
	r.addEvent(prepPath(http.MethodPatch, r.prefix, path), event{h: handler})
}

// Delete adds a new DELETE method route to the router. The path parameter is the route path you
// wish to define. The handler parameter is a lambda.Handler to invoke if an incoming path matches
// the route.
func (r *Router) Delete(path string, handler lambda.Handler) {
	r.addEvent(prepPath(http.MethodDelete, r.prefix, path), event{h: handler})
}

// Invoke implements the lambda.Handler interface for the Router type.
func (r Router) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	var req events.APIGatewayProxyRequest

	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	path := req.Path

	for param, value := range req.PathParameters {
		path = strings.Replace(path, value, "{"+param+"}", -1)
	}

	i, found := r.events.Get([]byte(req.HTTPMethod + path))

	if !found {
		return json.Marshal(events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Body:       "not found",
		})
	}

	e := i.(event)
	return e.h.Invoke(ctx, payload)
}

// Group allows you to define many routes with the same prefix. The prefix parameter will be applied
// to all routes defined in the function. The fn parameter is a function in which the grouped
// routes should be defined.
func (r *Router) Group(prefix string, fn func(r *Router)) {
	validatePathPart(prefix)

	if prefix[0] == '/' {
		prefix = prefix[1:]
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	original := r.prefix
	r.prefix += prefix
	fn(r)
	r.prefix = original
}

type event struct {
	h lambda.Handler
}

func (r *Router) addEvent(key string, e event) {
	if r.events == nil {
		panic("router not initialized")
	}

	routes, _, overwrite := r.events.Insert([]byte(key), e)

	if overwrite {
		panic(fmt.Sprintf("event '%s' already exists", key))
	}

	r.events = routes
}

func prepPath(method, prefix, path string) string {
	validatePathPart(path)

	if path[0] == '/' {
		path = path[1:]
	}
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	return method + prefix + path
}

func validatePathPart(part string) {
	if len(part) == 0 {
		panic("path was empty")
	}
}
