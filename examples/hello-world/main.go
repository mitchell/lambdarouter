package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mitchell/lambdarouter"
)

var r = lambdarouter.New("hellosrv")

func init() {
	r.Post("hello", lambda.NewHandler(func() (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusCreated,
			Body:       "hello world",
		}, nil
	}))

	r.Group("hello", func(r *lambdarouter.Router) {
		r.Get("{name}", lambda.NewHandler(func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       "hello " + req.PathParameters["name"],
			}, nil
		}))

		r.Put("french", lambda.NewHandler(func() (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       "bonjour le monde",
			}, nil
		}))

		r.Get("french/{prenom}", lambda.NewHandler(func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusOK,
				Body:       "bonjour " + req.PathParameters["prenom"],
			}, nil
		}))
	})
}

func main() {
	lambda.StartHandler(r)
}
