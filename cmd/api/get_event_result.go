package main

import (
	"context"
	"encoding/json"
	"observaquiz_lambda/pkg/instrumentation"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var getEventResultEndpoint = apiEndpoint{
	"GET",
	"/api/events/{eventName}/results",
	regexp.MustCompile("^/api/events/([^/]+)/results$"),
	getEventResults,
	true,
}

func getEventResults(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {

	requestedEventName := strings.Split(request.RawPath, "/")[3]

	executionResult, err := resultsTable.GetAllResultsForEvent(currentContext, requestedEventName)
	if err != nil {
		return instrumentation.ErrorResponse("Error getting from dynamodb", 500), err
	}

	jsonData, err := json.Marshal(executionResult)
	if err != nil {
		var currentSpan = oteltrace.SpanFromContext(currentContext)

		currentSpan.RecordError(err, oteltrace.WithAttributes(attribute.String("error.message", "Failure marshalling JSON")))
		return instrumentation.ErrorResponse("wtaf", 500), nil
	}

	return events.APIGatewayV2HTTPResponse{Body: string(jsonData), StatusCode: 200}, nil
}
