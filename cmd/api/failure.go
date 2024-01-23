package main

import (
	"fmt"
	"runtime/debug"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func RepondToPanic(span oteltrace.Span, r interface{}) events.APIGatewayV2HTTPResponse {
	span.SetStatus(codes.Error, "Panic caught")
	error, ok := r.(error)
	if ok {
		// r is an error
		span.RecordError(error)
		fmt.Printf("%s", debug.Stack())
		span.SetAttributes(attribute.String("error.stack", fmt.Sprintf("%s", debug.Stack())),
			attribute.String("error.type", "legit (error)"))
	} else {
		span.RecordError(fmt.Errorf("%v", r))
		span.SetAttributes(attribute.String("error.type", "some panic that is not (error)"))
	}
	return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("{ \"error\": \"Panic caught: %v\" }", r), StatusCode: 500}

}
