package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func receiveEvaluation(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	span := oteltrace.SpanFromContext(currentContext)

	/**
	  OK, I went to implement this, and couldn't find the place to hook it up
	  -- turns out DeepChecks hasn't implemented this webhook stuff yet. So this is gonna sit.
	*/

	span.SetAttributes(attribute.String("response.body", request.Body))

	return events.APIGatewayV2HTTPResponse{
		Body:       `{"wow": "such response"}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
