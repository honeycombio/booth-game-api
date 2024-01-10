package main

import (
	"context"
	"fmt"
	"os"
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
	ATTENDEE_API_KEY_HEADER        = "X-Attendee-Api-Key"
	ATTENDEE_API_KEY_ATTRIBUTE_KEY = "app.honeycomb_api_key"
)

var tracer oteltrace.Tracer

func ApiRouter(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, cleanup := context.WithTimeout(currentContext, 30*time.Second)
	defer cleanup()

	attendeeApiKey := request.Headers[ATTENDEE_API_KEY_HEADER]
	currentContext = SetApiKeyInBaggage(currentContext, attendeeApiKey)

	lambdaSpan := oteltrace.SpanFromContext(currentContext)
	lambdaSpan.SetAttributes(attribute.String(ATTENDEE_API_KEY_ATTRIBUTE_KEY, attendeeApiKey))
	addHttpRequestAttributesToSpan(lambdaSpan, request)

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

	addHttpResponseAttributesToSpan(lambdaSpan, response)

	return response, err

}

var settings struct {
	OpenAIKey string `env:"openai_key"`
}

func main() {
	flags.Parse(&settings)
	// print all the environment variables to the console
	settings.OpenAIKey = os.Getenv("openai_key")
	currentContext := context.Background()

	tracerProvider := createTracerProvider(currentContext)

	tracer = tracerProvider.Tracer("booth-game-backend")
	lambda.StartWithOptions(
		otellambda.InstrumentHandler(ApiRouter,
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
