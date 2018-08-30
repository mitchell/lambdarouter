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
		rtr := NewAPIGRouter(&APIGRouterConfig{
			Request: &request,
			Prefix:  "/shipping",
			Headers: map[string]string{
				"Access-Control-Allow-Origin":      "*",
				"Access-Control-Allow-Credentials": "true",
			},
		})

		Convey("When the handler func does NOT return an error", func() {
			hdlrfunc := func(ctx *APIGContext) {
				ctx.Status = http.StatusOK
				ctx.Body = []byte("hello")
				ctx.Err = nil
			}

			Convey("And a Get handler expecting the pattern /listings/{id}/state/{event} is defined", func() {
				rtr.Get("/listings/{id}/state/{event}", hdlrfunc)
				rtr.Post("/orders", func(ctx *APIGContext) {})
				rtr.Put("/orders", func(ctx *APIGContext) {})
				rtr.Patch("/orders", func(ctx *APIGContext) {})
				rtr.Delete("/orders/{id}", func(ctx *APIGContext) {})

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/listings/57/state/list"
					request.PathParameters = map[string]string{
						"id":    "57",
						"event": "list",
					}
					request.RequestContext.Authorizer = map[string]interface{}{
						"claims": map[string]interface{}{
							"cognito:username": "mitchell",
						},
					}

					Convey("The router will return the expected status, body, and headers", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusOK)
						So(response.Body, ShouldEqual, "hello")
						So(response.Headers, ShouldResemble, map[string]string{
							"Access-Control-Allow-Origin":      "*",
							"Access-Control-Allow-Credentials": "true",
						})
					})
				})

				Convey("And the request does NOT match the pattern", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/orders/filter"

					Convey("The router will return an error body and a status not found", func() {
						response := rtr.Respond()

						So(response.StatusCode, ShouldEqual, http.StatusNotFound)
						So(response.Body, ShouldEqual, "{\"error\":\"no route matching path found\"}")
						So(response.Headers, ShouldResemble, map[string]string{
							"Access-Control-Allow-Origin":      "*",
							"Access-Control-Allow-Credentials": "true",
						})
					})
				})

				Convey("And a Get handler expecting the pattern /listings/{id}/state/{event} is defined AGAIN", func() {
					So(func() {
						rtr.Get("/listings/{id}/state/{event}", hdlrfunc)
					}, ShouldPanicWith, "endpoint already existent")
				})

				Convey("And a Get handler expecting the pattern /orders/filter", func() {
					rtr.Get("/orders/filter", hdlrfunc)

					Convey("And the request matches the pattern and the path params are filled", func() {
						request.HTTPMethod = http.MethodGet
						request.Path = "/shipping/orders/filter"

						Convey("The router will return the expected status and body", func() {
							response := rtr.Respond()

							So(response.StatusCode, ShouldEqual, http.StatusOK)
							So(response.Body, ShouldEqual, "hello")
						})
					})

					Convey("And the request does NOT match either of the patterns", func() {
						request.HTTPMethod = http.MethodGet
						request.Path = "/shipping/orders/filter/by_user"

						Convey("The router will return an error body and a status not found", func() {
							response := rtr.Respond()

							So(response.StatusCode, ShouldEqual, http.StatusNotFound)
							So(response.Body, ShouldEqual, "{\"error\":\"no route matching path found\"}")
						})
					})
				})
			})

		})

		Convey("When the handler func does return a record not found", func() {
			hdlrfunc := func(ctx *APIGContext) {
				ctx.Status = http.StatusNoContent
				ctx.Body = []byte("hello")
				ctx.Err = errors.New("record not found")

			}

			Convey("And a Get handler expecting the pattern /listings/{id}/state/{event} is defined", func() {
				rtr.Get("/listings/{id}/state/{event}", hdlrfunc)

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/listings/57/state/list"
					request.PathParameters = map[string]string{
						"id":    "57",
						"event": "list",
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
			middlefunc1 := func(ctx *APIGContext) {
				ctx.Status = http.StatusOK
				ctx.Body = []byte("hello")
				ctx.Err = nil
			}
			middlefunc2 := func(ctx *APIGContext) {
				ctx.Status = http.StatusOK
				ctx.Body = []byte("hello")
				ctx.Err = errors.New("bad request")
			}
			hdlrfunc := func(ctx *APIGContext) {
				ctx.Status = http.StatusOK
				ctx.Body = []byte("hello")
				ctx.Err = nil
			}

			Convey("And a Get handler expecting the pattern /listings/{id}/state/{event} is defined", func() {
				rtr.Get("/listings/{id}/state/{event}", middlefunc1, middlefunc2, hdlrfunc)

				Convey("And the request matches the pattern and the path params are filled", func() {
					request.HTTPMethod = http.MethodGet
					request.Path = "/shipping/listings/57/state/list"
					request.PathParameters = map[string]string{
						"id":    "57",
						"event": "list",
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
