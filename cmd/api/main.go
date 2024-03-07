package main

import (
	"context"
	"fmt"
	"observaquiz_lambda/pkg/instrumentation"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	default_event           = "devopsdays_whenever"
	ATTENDEE_API_KEY_HEADER = "x-honeycomb-api-key"
	EXECUTION_ID_HEADER     = "x-observaquiz-execution-id"
	ServiceName             = "observaquiz-bff"
)

const LocalTraceLink = true // feature flag, enable locally and turn off in prod ideally

func RouterWithSpan(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, cleanup := context.WithTimeout(currentContext, 30*time.Second)
	defer cleanup()
	lambdaSpan := oteltrace.SpanFromContext(currentContext)
	defer func() {
		if r := recover(); r != nil {
			response = instrumentation.RespondToPanic(lambdaSpan, r)
		}
	}()

	currentContext, _ = setAttributesOnSpanAndBaggageFromHeaders(currentContext, request)
	instrumentation.AddHttpRequestAttributesToSpan(lambdaSpan, request)

	response, err = getResponseFromAPIRouter(currentContext, request)

	instrumentation.AddHttpResponseAttributesToSpan(lambdaSpan, response)
	addSpanAttributesToResponse(currentContext, &response)

	return response, err

}

func getResponseFromAPIRouter(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
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

func addSpanAttributesToResponse(currentContext context.Context, response *events.APIGatewayV2HTTPResponse) {
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}

	carrier := propagation.MapCarrier{}

	otel.GetTextMapPropagator().Inject(currentContext, carrier)
	response.Headers["x-tracechild"] = carrier["traceparent"]
}

func setAttributesOnSpanAndBaggageFromHeaders(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (context.Context, error) {
	var currentSpan = oteltrace.SpanFromContext(currentContext)
	var attendeeApiKey string
	var executionId string = "unset"
	for k, v := range request.Headers {
		if strings.ToLower(k) == ATTENDEE_API_KEY_HEADER {
			attendeeApiKey = v
		}
		if strings.ToLower(k) == EXECUTION_ID_HEADER {
			executionId = v
		}
	}
	currentSpan.SetAttributes(attribute.String(instrumentation.ATTENDEE_API_KEY_ATTRIBUTE_KEY, attendeeApiKey))
	currentSpan.SetAttributes(attribute.String(instrumentation.EXECUTION_ID_ATTRIBUTE_KEY, executionId))
	return instrumentation.SetApiKeyInBaggage(currentContext, attendeeApiKey, executionId)
}

var settings struct {
	OpenAIKey        string `env:"openai_key"`
	QueryDataApiKey  string `env:"query_data_api_key"`
	DeepchecksApiKey string `env:"deepchecks_api_key"`
}

func main() {
	flags.Parse(&settings)
	// print all the environment variables to the console
	settings.OpenAIKey = os.Getenv("openai_key")
	settings.QueryDataApiKey = os.Getenv("query_data_api_key") // whatever, if it works
	settings.DeepchecksApiKey = os.Getenv("deepchecks_api_key")
	currentContext := context.Background()

	tracerProvider := instrumentation.CreateTracerProvider(currentContext, ServiceName)

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
