# lambdarouter
[![GoDoc Reference](https://godoc.org/github.com/mitchell/lambdarouter?status.svg)](https://godoc.org/github.com/mitchell/lambdarouter)
[![Build Status](https://travis-ci.org/mitchell/lambdarouter.svg?branch=master)](https://travis-ci.org/mitchell/lambdarouter)
[![Test Coverage](https://api.codeclimate.com/v1/badges/7270c6c4017b36d07360/test_coverage)](https://codeclimate.com/github/mitchelljfs/lambdarouter/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/7270c6c4017b36d07360/maintainability)](https://codeclimate.com/github/mitchelljfs/lambdarouter/maintainability)
[![Go Report Card](https://goreportcard.com/badge/github.com/mitchell/lambdarouter)](https://goreportcard.com/report/github.com/mitchell/lambdarouter)

This package contains a router capable of routing many AWS Lambda API gateway requests to anything
that implements the aws-lambda-go/lambda.Handler interface, all in one Lambda function. It plays
especially well with go-kit's awslambda transport package. Get started by reading below and visiting
the [GoDoc reference](https://godoc.org/github.com/mitchell/lambdarouter).

## Initializing a Router
```
r := lambdarouter.New("prefix/")

r.Get("hello/{name}", helloHandler)
r.Post("hello/server", helloHandler)
r.Delete("hello", lambda.NewHandler(func() (events.APIGatewayProxyResponse, error) {
        return events.APIGatewayProxyResponse{
                Body: "nothing to delete",
        }, nil
}))

lambda.StartHandler(r)
```

Check out the `examples/` folder for more fleshed out examples in the proper context.
