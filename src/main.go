package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var apiEndpoints = []apiEndpoint{
	{
		"GET",
		"/api/events",
		regexp.MustCompile("^/api/events$"),
		getEvents,
		false,
	},
	{
		"GET",
		"/api/questions",
		regexp.MustCompile("^/api/questions$"),
		getQuestions,
		true,
	},
	{
		"POST",
		"/api/questions/{questionId}/answer",
		regexp.MustCompile("^/api/questions/([^/]+)/answer$"),
		postAnswer,
		true,
	},
}

type apiEndpoint struct {
	method        string
	pathTemplate  string
	pathRegex     *regexp.Regexp
	handler       interface{}
	requiresEvent bool
}

const (
	default_event = "devopsdays_whenever"
)

func Api(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	span := oteltrace.SpanFromContext(currentContext)
	span.SetAttributes(
		semconv.HTTPURL(request.RequestContext.HTTP.Path),
		semconv.HTTPMethod(request.RequestContext.HTTP.Method))

	for _, v := range apiEndpoints {
		if v.method == request.RequestContext.HTTP.Method &&
			v.pathRegex.MatchString(request.RequestContext.HTTP.Path) {

			span.SetName(fmt.Sprintf("%s %s", v.method, v.pathTemplate))
			if v.requiresEvent {
				eventName := getEventName(request)
				if _, eventFound := eventQuestions[eventName]; !eventFound {
					return events.APIGatewayV2HTTPResponse{
						Body:       fmt.Sprintf("Couldn't find event name %s", eventName),
						StatusCode: 404,
					}, nil
				}
			}

			return v.handler.(func(events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error))(request)
		}
	}
	methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path

	return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}, nil
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

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(Api,
			otellambda.WithFlusher(tracerProvider),
			otellambda.WithTracerProvider(tracerProvider)),
		lambda.WithContext(currentContext),
	)
}

func createTracerProvider(currentContext context.Context) *trace.TracerProvider {
	resource, _ := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("booth-game-backend"),
			semconv.ServiceVersion("0.0.1"),
		))

	httpExporter, _ := otlptracehttp.New(currentContext)

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(httpExporter),
		trace.WithResource(resource))

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider
}

func getEventName(request events.APIGatewayV2HTTPRequest) string {
	eventName := request.Headers["event-name"]
	if eventName == "" {
		eventName = default_event
	}

	return eventName
}
