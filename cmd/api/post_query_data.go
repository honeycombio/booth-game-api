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
	oteltrace "go.opentelemetry.io/otel/trace"
)

var queryDataEndpoint = apiEndpoint{
	"POST",
	"/api/queryData",
	regexp.MustCompile("^/api/queryData$"),
	postQueryData,
	true,
}

func postQueryData(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {

	currentSpan := oteltrace.SpanFromContext(currentContext)

	if settings.QueryDataApiKey != "" {
		currentSpan.RecordError(err)
		return instrumentation.ErrorResponse(err.Error(), 500), nil
	}
	/* Parse what they sent */
	currentSpan.SetAttributes(
		attribute.String("request.body", request.Body),
		attribute.Bool("app.query_data_apikey_populated", settings.QueryDataApiKey != ""))

	queryRequest := queryData.QueryDataRequest{}

	err = json.Unmarshal([]byte(request.Body), &queryRequest)
	if err != nil {
		newErr := fmt.Errorf("error unmarshalling answer: %w\n request body: %s", err, request.Body)
		currentSpan.RecordError(newErr)
		return instrumentation.ErrorResponse("Bad request. Expected format: { 'query': 'query as a string of escaped json' }", 400), nil
	}

	questionResponse, err := queryData.RunHoneycombQuery(currentContext, settings.QueryDataApiKey, queryRequest)
	if err != nil {
		currentSpan.RecordError(err)
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
