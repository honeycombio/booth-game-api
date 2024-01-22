package main

import (
	"booth_game_lambda/pkg/instrumentation"
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

	tracerProvider := instrumentation.CreateTracerProvider(currentContext, "deep-checks-callback")

	tracer = tracerProvider.Tracer("deep-checks-callback") // Is this even used?
	_, span := tracer.Start(currentContext, "callback lambda runs")
	defer span.End()

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
	instrumentation.AddHttpRequestAttributesToSpan(lambdaSpan, request)

	response, _ = receiveEvaluation(currentContext, request)

	instrumentation.AddHttpResponseAttributesToSpan(lambdaSpan, response)

	return response, nil
}
