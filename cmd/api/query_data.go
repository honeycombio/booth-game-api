package main

import (
	"booth_game_lambda/cmd/api/queryData"
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
)

var queryDataEndpoint = apiEndpoint{
	"POST",
	"/api/queryData",
	regexp.MustCompile("^/api/queryData$"),
	fetchQueryData,
	true,
}

func fetchQueryData(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {

	tracer := instrumentation.TracerProvider.Tracer("app.query_data")
	currentContext, queryDataSpan := tracer.Start(currentContext, "Fetch Query Data from Honeycomb")
	defer queryDataSpan.End()
	defer func() {
		if r := recover(); r != nil {
			response = RepondToPanic(queryDataSpan, r)
		}
	}()

	/* Parse what they sent */
	queryDataSpan.SetAttributes(attribute.String("request.body", request.Body))
	queryRequest := queryData.QueryDataRequest{}
	err = json.Unmarshal([]byte(request.Body), &queryRequest)
	if err != nil {
		newErr := fmt.Errorf("error unmarshalling answer: %w\n request body: %s", err, request.Body)
		queryDataSpan.RecordError(newErr)
		return ErrorResponse("Bad request. Expected format: { 'query': 'query as a string of escaped json' }", 400), nil
	}

	questionResponse, err := queryData.RunHoneycombQuery(currentContext, queryRequest)

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
