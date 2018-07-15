package lambdarouter

import (
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRouterSpec(t *testing.T) {

	Convey("Given an instantiated router", t, func() {
		request := events.APIGatewayProxyRequest{}
		rtr := NewAPIGRouter(&request, "shipping")

		Convey("When the handler func does NOT return an error", func() {
			hdlrfunc := func(req *APIGRequest, res *APIGResponse) {
				res.Status = http.StatusOK
				res.Body = []byte("hello")
				res.Err = nil

			}

			Convey("And a Get handler expecting the pattern /orders/filter/by_user/{id} is defined", func() {
				rtr.Get("/orders/filter/by_user/{id}", hdlrfunc)
				rtr.Post("/orders", func(req *APIGRequest, res *APIGResponse) {})
				rtr.Put("/orders", func(req *APIGRequest, res *APIGResponse) {})
				rtr.Patch("/orders", func(req *APIGRequest, res *APIGResponse) {})
				rtr.Delete("/orders/{id}", func(req *APIGRequest, res *APIGResponse) {})

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/orders/filter/by_user/4d50ff90-66e3-4047-bf37-0ca25837e41d"
					request.PathParameters = map[string]string{
						"id": "4d50ff90-66e3-4047-bf37-0ca25837e41d",
					}
					request.RequestContext.Authorizer = map[string]interface{}{
						"claims": map[string]interface{}{
							"cognito:username": "mitchell",
						},
					}

					Convey("The router will return the expected status and body", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusOK)
						So(response.Body, ShouldEqual, "hello")
					})
				})

				Convey("And the request does NOT match the pattern", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/orders/filter"

					Convey("The router will return and error body and a not found status", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusNotFound)
						So(response.Body, ShouldEqual, "{\"error\":\"no route matching path found\"}")
					})
				})

				Convey("And a Get handler expecting the pattern /orders/filter/by_user/{id} is defined AGAIN", func() {
					So(func() {
						rtr.Get("/orders/filter/by_user/{id}", hdlrfunc)
					}, ShouldPanicWith, "endpoint already existent")
				})
			})

		})

		Convey("When the handler func does return a record not found", func() {
			hdlrfunc := func(req *APIGRequest, res *APIGResponse) {
				res.Status = http.StatusBadRequest
				res.Body = []byte("hello")
				res.Err = errors.New("record not found")

			}

			Convey("And a Get handler expecting the pattern /orders/filter/by_user/{id} is defined", func() {
				rtr.Get("/orders/filter/by_user/{id}", hdlrfunc)

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/orders/filter/by_user/4d50ff90-66e3-4047-bf37-0ca25837e41d"
					request.PathParameters = map[string]string{
						"id": "4d50ff90-66e3-4047-bf37-0ca25837e41d",
					}

					Convey("The router will return the expected status and body", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusNoContent)
						So(response.Body, ShouldEqual, "{\"error\":\"record not found\"}")
					})
				})
			})
		})

		Convey("When the handler func does return a status < 400", func() {
			hdlrfunc := func(req *APIGRequest, res *APIGResponse) {
				res.Status = http.StatusOK
				res.Body = []byte("hello")
				res.Err = errors.New("bad request")

			}

			Convey("And a Get handler expecting the pattern /orders/filter/by_user/{id} is defined", func() {
				rtr.Get("/orders/filter/by_user/{id}", hdlrfunc)

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/orders/filter/by_user/4d50ff90-66e3-4047-bf37-0ca25837e41d"
					request.PathParameters = map[string]string{
						"id": "4d50ff90-66e3-4047-bf37-0ca25837e41d",
					}

					Convey("The router will return the expected status and body", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusBadRequest)
						So(response.Body, ShouldEqual, "{\"error\":\"bad request\"}")
					})
				})
			})
		})
	})
}
