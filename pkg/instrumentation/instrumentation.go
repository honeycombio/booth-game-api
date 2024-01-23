package instrumentation

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var TracerProvider *sdktrace.TracerProvider

func CreateTracerProvider(currentContext context.Context, serviceName string) *sdktrace.TracerProvider {
	resource, _ := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.0.1"),
		))

	httpExporter, _ := otlptracehttp.New(currentContext)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(httpExporter),
		sdktrace.WithResource(resource))

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetTracerProvider(tracerProvider)

	TracerProvider = tracerProvider
	return tracerProvider
}

func AddHttpRequestAttributesToSpan(span trace.Span, request events.APIGatewayV2HTTPRequest) {
	span.SetAttributes(
		semconv.URLPath(request.RequestContext.HTTP.Path),
		semconv.HTTPMethod(request.RequestContext.HTTP.Method),
		semconv.ServerAddress(request.RequestContext.DomainName),
		semconv.ClientAddress(request.RequestContext.HTTP.SourceIP),
		semconv.URLScheme(request.RequestContext.HTTP.Protocol),
		semconv.UserAgentOriginal(request.RequestContext.HTTP.UserAgent),
	)
}

func AddHttpResponseAttributesToSpan(span trace.Span, response events.APIGatewayV2HTTPResponse) {
	span.SetAttributes(
		semconv.HTTPResponseStatusCode(int(response.StatusCode)),
		semconv.HTTPResponseBodySize(len(response.Body)),
	)
}

func InjectTraceParentToResponse(span trace.Span, response *events.APIGatewayV2HTTPResponse) {
	response.Headers[`traceparent`] = fmt.Sprintf("%s-%s-%s-%s", "00", span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String(), "01")
}
