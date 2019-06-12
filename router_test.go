package lambdarouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	a := assert.New(t)

	desc(t, 0, "Intialize Router and")
	r := New("prefix")
	handler := lambda.NewHandler(handler)
	ctx := context.Background()

	desc(t, 2, "Get|Post|Put|Patch|Delete method should")
	{
		desc(t, 4, "insert a new route succesfully")
		a.NotPanics(func() {
			r.Get("thing/{id}", handler)
			r.Delete("/thing/{id}/", handler)
			r.Put("thing", handler)
		})

		desc(t, 4, "panic when inserting the same route")
		a.Panics(func() {
			r.Put("thing", handler)
		})

		desc(t, 4, "panic when router is uninitalized")
		var r2 Router
		a.Panics(func() {
			r2.Patch("panic", handler)
		})

		desc(t, 4, "panic when when given an empty path")
		a.Panics(func() {
			r.Post("", handler)
		})
	}

	desc(t, 2, "PrefixGroup method should")
	{
		desc(t, 4, "insert routes with the specified prefix succesfully")
		r.Group("/ding", func(r *Router) {
			r.Post("dong/{door}", handler)
		})
	}

	desc(t, 2, "Invoke method should")
	{
		e := events.APIGatewayProxyRequest{
			Path:           "/prefix/ding/dong/mitchell",
			HTTPMethod:     http.MethodPost,
			PathParameters: map[string]string{"door": "mitchell"},
		}

		desc(t, 4, "should succesfully route and invoke a defined route")
		ejson, _ := json.Marshal(e)

		res, err := r.Invoke(ctx, ejson)

		a.NoError(err)
		a.Exactly("null", string(res))

		desc(t, 4, "return the expected response when a route is not found")
		e.Path = "thing"
		e.PathParameters = nil
		ejson2, _ := json.Marshal(e)
		eres := events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Body:       "not found",
		}
		eresjson, _ := json.Marshal(eres)

		res, err = r.Invoke(ctx, ejson2)

		a.NoError(err)
		a.ElementsMatch(eresjson, res)

		desc(t, 4, "return an error when the there is an issue with the incoming event")
		_, err = r.Invoke(ctx, nil)

		a.Error(err)
	}

}

func handler() error {
	return nil
}

func desc(t *testing.T, depth int, str string, args ...interface{}) {
	for i := 0; i < depth; i++ {
		str = " " + str
	}

	t.Log(fmt.Sprintf(str, args...))
}
