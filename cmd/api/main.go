package main

import (
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	default_event                  = "devopsdays_whenever"
	ATTENDEE_API_KEY_HEADER        = "x-honeycomb-api-key"
	ATTENDEE_API_KEY_ATTRIBUTE_KEY = "app.honeycomb_api_key"
)

func RouterWithSpan(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, cleanup := context.WithTimeout(currentContext, 30*time.Second)
	defer cleanup()
	lambdaSpan := oteltrace.SpanFromContext(currentContext)
	defer func() {
		if r := recover(); r != nil {
			response = RepondToPanic(lambdaSpan, r)
		}
	}()

	var attendeeApiKey string
	for k, v := range request.Headers {
		if strings.ToLower(k) == ATTENDEE_API_KEY_HEADER {
			attendeeApiKey = v
			break
		}
	}
	if attendeeApiKey != "" {
		currentContext, err = SetApiKeyInBaggage(currentContext, attendeeApiKey)
		if err != nil {
			lambdaSpan.SetAttributes(attribute.String("error.message", fmt.Sprintf("failed at setting api key in baggage")))
			lambdaSpan.RecordError(err)
		}
		lambdaSpan.SetAttributes(attribute.String(ATTENDEE_API_KEY_ATTRIBUTE_KEY, attendeeApiKey))
	}
	instrumentation.AddHttpRequestAttributesToSpan(lambdaSpan, request)

	response, err = ApiRouter(currentContext, request)

	instrumentation.AddHttpResponseAttributesToSpan(lambdaSpan, response)
	addSpanAttributesToResponse(lambdaSpan, &response)

	return response, err

}

func ApiRouter(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	lambdaSpan := oteltrace.SpanFromContext(currentContext)

	endpoint, endpointFound := api.findEndpoint(request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path)

	if !endpointFound {
		methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path
		response = events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}
	} else {
		lambdaSpan.SetName(fmt.Sprintf("%s %s", endpoint.method, endpoint.pathTemplate))
		response, err = getResponseFromHandler(currentContext, endpoint, request)
		if err != nil {
			lambdaSpan.RecordError(err)
		}
	}

	return response, err

}

func addSpanAttributesToResponse(lambdaSpan oteltrace.Span, response *events.APIGatewayV2HTTPResponse) {
	// traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
	/*
		version
		trace-id
		parent-id
		trace-flags
	*/
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}
	response.Headers["x-tracechild"] = fmt.Sprintf("%s-%s-%s-%s", "00", lambdaSpan.SpanContext().TraceID().String(), lambdaSpan.SpanContext().SpanID().String(), "01")
}

var settings struct {
	OpenAIKey string `env:"openai_key"`
}

func main() {
	flags.Parse(&settings)
	// print all the environment variables to the console
	settings.OpenAIKey = os.Getenv("openai_key")
	currentContext := context.Background()

	tracerProvider := instrumentation.CreateTracerProvider(currentContext, "observaquiz-bff")

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(RouterWithSpan,
			otellambda.WithFlusher(tracerProvider),
			otellambda.WithTracerProvider(tracerProvider)),
		lambda.WithContext(currentContext),
	)
}

func getEventName(request events.APIGatewayV2HTTPRequest) string {
	eventName := request.Headers["event-name"]
	if eventName == "" {
		eventName = default_event
	}

	return eventName
}
