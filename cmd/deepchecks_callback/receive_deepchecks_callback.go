package main

import (
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type deepchecksCallbackContent struct {
	UserInteractionId string `json:"user_interaction_id"`
}

func receiveEvaluation(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	span := oteltrace.SpanFromContext(currentContext)

	span.SetAttributes(attribute.String("request.body", request.Body))

	callbackContent := deepchecksCallbackContent{}
	err := json.Unmarshal([]byte(request.Body), &callbackContent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error unmarshalling request body")
		return instrumentation.ErrorResponse("Error unmarshalling request body", 400), nil
	}

	return events.APIGatewayV2HTTPResponse{
		Body:       `{"wow": "such response"}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
