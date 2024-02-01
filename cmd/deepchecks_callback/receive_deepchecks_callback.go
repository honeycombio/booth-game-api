package main

import (
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type deepchecksCallbackContent struct {
	UserInteractionId string `json:"user_interaction_id"`
}

type deepchecksCallbackResponse struct {
	Wow               string `json:"wow"`
	Message           string `json:"message"`
	CreatedSpan       string `json:"created_span_in_honeycomb"`
	ThisExecutionSpan string `json:"span_describing_this_execution"`
}

func linkToTraceInLocalEnvironment(traceID string, spanID string) string {
	// the only time anybody ever looks at the response is during local testing.
	return fmt.Sprintf("https://ui.honeycomb.io/%s/environments/%s/datasets/%s/trace?trace_id=%s&span=%s",
		"modernity", "quiz-local", ServiceName, traceID, spanID)
}

func callbackReceivedResponse(currentContext context.Context, msg string) events.APIGatewayV2HTTPResponse {
	span := oteltrace.SpanFromContext(currentContext)
	response := deepchecksCallbackResponse{
		Wow:               "such response",
		Message:           msg,
		ThisExecutionSpan: linkToTraceInLocalEnvironment(span.SpanContext().TraceID().String(), span.SpanContext().TraceID().String()),
	}
	json, _ := json.Marshal(response)
	return events.APIGatewayV2HTTPResponse{
		Body:       string(json),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}
}

func receiveEvaluation(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, span := tracer.Start(currentContext, "Receive Evaluation")
	defer span.End()
	defer func() {
		if r := recover(); r != nil {
			response = instrumentation.RespondToPanic(span, r)
		}
	}()

	span.SetAttributes(attribute.String("request.body", request.Body))

	callbackContent := deepchecksCallbackContent{}
	err = json.Unmarshal([]byte(request.Body), &callbackContent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error unmarshalling request body")
		// OK they sent us some garbage
		return instrumentation.ErrorResponse("Error unmarshalling request body: "+err.Error(), 400), nil
	}
	span.SetAttributes(attribute.String("app.deepChecks.user_interaction_id", callbackContent.UserInteractionId))

	// TODO: start the span at the time deepchecks created the interaction - show how long evaluation took.
	spanRecordingResults, err := startSpanToRecordResults(currentContext, callbackContent.UserInteractionId)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// this wasn't their fault, so respond 200
		return callbackReceivedResponse(currentContext, "Failed to create result span: "+err.Error()), nil
	}
	defer spanRecordingResults.End()
	spanRecordingResults.SetAttributes(attribute.String("app.deepChecks.user_interaction_id", callbackContent.UserInteractionId),
		attribute.String("app.deepChecks.full_report", request.Body))

	// Add all the rest
	spanRecordingResults.End() // this should send it
	return callbackReceivedResponse(currentContext, "Hey, it worked!"), nil
}

func createSpanLink(currentContext context.Context, spanContext context.Context) {
	_, spanThatOnlyExistsBecauseWeCantAddLinks := tracer.Start(currentContext, "Link to created span", oteltrace.WithLinks(oteltrace.LinkFromContext(spanContext)))
	spanThatOnlyExistsBecauseWeCantAddLinks.End()
}

func startSpanToRecordResults(currentContext context.Context, userInteractionId string) (spanRecordingResults oteltrace.Span, err error) {
	span := oteltrace.SpanFromContext(currentContext)
	parts := strings.Split(userInteractionId, "-")
	if len(parts) != 2 {
		span.RecordError(errors.New("user_interaction_id is not in the expected format"))
		span.SetStatus(codes.Error, "user_interaction_id is not in the expected format")
		// our fault not theirs
		return nil, errors.New("user_interaction_id is not in the expected format")
	}
	traceID := parts[0]
	spanID := parts[1]
	span.SetAttributes(attribute.String("app.deepChecks.trace_id", traceID),
		attribute.String("app.deepChecks.span_id", spanID))
	traceIDfromHex, err := oteltrace.TraceIDFromHex(traceID)
	if err != nil {
		msg := "could not construct Trace ID from hex"
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, msg)
		return nil, errors.New(msg)
	}
	spanIDfromHex, err := oteltrace.SpanIDFromHex(spanID)
	if err != nil {
		msg := "could not construct Span ID from hex"
		span.RecordError(errors.New(msg))
		span.SetStatus(codes.Error, msg)
		return nil, errors.New(msg)
	}

	contextToPutALogIn := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(
		oteltrace.SpanContextConfig{
			TraceID:    traceIDfromHex,
			SpanID:     spanIDfromHex,
			TraceFlags: 0x1, // assuming sampled
		}))

	tracer := otel.Tracer("my-tracer")
	contextOfLog, spanRecordingResults := tracer.Start(contextToPutALogIn, "LLM Evaluation Results")

	createSpanLink(currentContext, contextOfLog)

	return spanRecordingResults, nil
}
