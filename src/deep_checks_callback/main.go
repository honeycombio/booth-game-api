package main

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer oteltrace.Tracer

var settings struct {
}

func main() {
	flags.Parse(&settings)
	currentContext := context.Background()

	tracerProvider := createTracerProvider(currentContext)

	tracer = tracerProvider.Tracer("deep-checks-callback") // Is this even used?
	_, span := tracer.Start(currentContext, "hello from callback")
	span.End()
	lambda.StartWithOptions(
		otellambda.InstrumentHandler(ApiRouter,
			otellambda.WithFlusher(tracerProvider),
			otellambda.WithTracerProvider(tracerProvider)),
		lambda.WithContext(currentContext),
	)
}

func ApiRouter(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, cleanup := context.WithTimeout(currentContext, 30*time.Second)
	defer cleanup()

	lambdaSpan := oteltrace.SpanFromContext(currentContext)
	addHttpRequestAttributesToSpan(lambdaSpan, request)

	response = events.APIGatewayV2HTTPResponse{
		StatusCode: 400,
		Body:       "Not Implemented... yet",
	}

	addHttpResponseAttributesToSpan(lambdaSpan, response)

	return response, nil
}
