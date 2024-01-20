package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

func receiveEvaluation(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	return events.APIGatewayV2HTTPResponse{
		Body:       `{"wow": "such response"}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
