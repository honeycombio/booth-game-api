package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func Api(context context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path

	switch methodPath {
	case "GET /api/questions":
		return getQuestions(request)
	default:
		return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}, nil
	}
}

func main() {
	lambda.Start(Api)
}
