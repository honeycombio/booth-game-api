package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

func createTracerProvider(currentContext context.Context) *sdktrace.TracerProvider {
	resource, _ := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("booth-game-backend"),
			semconv.ServiceVersion("0.0.1"),
		))

	httpExporter, _ := otlptracehttp.New(currentContext)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(httpExporter),
		sdktrace.WithResource(resource))

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider
}

func addHttpRequestAttributesToSpan(span trace.Span, request events.APIGatewayV2HTTPRequest) {
	span.SetAttributes(
		semconv.URLPath(request.RequestContext.HTTP.Path),
		semconv.HTTPMethod(request.RequestContext.HTTP.Method),
		semconv.ServerAddress(request.RequestContext.DomainName),
		semconv.ClientAddress(request.RequestContext.HTTP.SourceIP),
		semconv.URLScheme(request.RequestContext.HTTP.Protocol),
		semconv.UserAgentOriginal(request.RequestContext.HTTP.UserAgent),
	)
}

func addHttpResponseAttributesToSpan(span trace.Span, response events.APIGatewayV2HTTPResponse) {
	span.SetAttributes(
		semconv.HTTPResponseStatusCode(int(response.StatusCode)),
		semconv.HTTPResponseBodySize(len(response.Body)),
	)
}
