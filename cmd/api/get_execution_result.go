package main

import (
	"context"
	"encoding/json"
	"observaquiz_lambda/pkg/instrumentation"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var getExecutionResultEndpoint = apiEndpoint{
	"GET",
	"/api/results",
	regexp.MustCompile("^/api/results$"),
	getExecutionResult,
	true,
}

func getExecutionResult(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	headerInfo := getHeaderInfo(request)

	executionId := headerInfo.ExecutionId

	executionResult, err := resultsTable.GetExecutionResultFromDynamo(currentContext, executionId)
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

type QuestionResult struct {
	QuestionId string `json:"question_id"`
}
