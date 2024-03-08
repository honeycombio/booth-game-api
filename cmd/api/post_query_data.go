package main

import (
	"context"
	"encoding/json"
	"fmt"
	"observaquiz_lambda/cmd/api/queryData"
	"observaquiz_lambda/pkg/instrumentation"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var queryDataEndpoint = apiEndpoint{
	"POST",
	"/api/queryData",
	regexp.MustCompile("^/api/queryData$"),
	postQueryDataProxy,
	true,
}

func postQueryDataProxy(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {

	currentSpan := oteltrace.SpanFromContext(currentContext)

	if settings.QueryDataApiKey == "" {
		err = fmt.Errorf("QueryDataApiKey is not set")
		currentSpan.RecordError(err)
		currentSpan.SetStatus(codes.Error, "QueryDataApiKey is not set")
		return instrumentation.ErrorResponse(err.Error(), 500), nil
	}

	/* Parse what they sent */
	currentSpan.SetAttributes(attribute.String("observaquiz.qd.query", request.Body))

	queryRequest := queryData.QueryDataRequest{}

	err = json.Unmarshal([]byte(request.Body), &queryRequest)
	if err != nil {
		newErr := fmt.Errorf("error unmarshalling answer: %w\n request body: %s", err, request.Body)
		currentSpan.RecordError(newErr)
		currentSpan.SetStatus(codes.Error, err.Error())
		return instrumentation.ErrorResponse("Bad request. Expected format: { 'query': 'query as a string of escaped json' }", 400), nil
	}

	questionResponse, err := queryData.CreateAndRunHoneycombQuery(currentContext, settings.QueryDataApiKey, queryRequest)
	if err != nil {
		currentSpan.RecordError(err)
		currentSpan.SetStatus(codes.Error, err.Error())
		return instrumentation.ErrorResponse(err.Error(), 500), nil
	}

	questionsJson, err := json.Marshal(questionResponse)
	if err != nil {
		fmt.Printf("Error marshalling questions: %v\n", err)
		return events.APIGatewayV2HTTPResponse{Body: "Internal Server Error", StatusCode: 500}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		Body:       string(questionsJson),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
