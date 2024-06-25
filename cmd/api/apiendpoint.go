package main

import (
	"context"
	"fmt"
	"observaquiz_lambda/pkg/instrumentation"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
)

var api = ApiHolder{
	apiEndpoints: []apiEndpoint{
		getEventsEndpoint,
		getQuestionsEndpoint,
		postAnswerEndpoint,
		queryDataEndpoint,
		postOpinionEndpoint,
		getExecutionResultEndpoint,
		getEventResultEndpoint,
	},
}

type apiEndpoint struct {
	method        string
	pathTemplate  string
	pathRegex     *regexp.Regexp
	handler       func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)
	requiresEvent bool
}

type ApiHolder struct {
	apiEndpoints []apiEndpoint
}

func getResponseFromHandler(currentContext context.Context, endpoint apiEndpoint, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if endpoint.requiresEvent {
		eventName := getEventName(request)
		if _, eventFound := eventQuestions[eventName]; !eventFound {
			return instrumentation.ErrorResponse(fmt.Sprintf("Couldn't find event name %s", eventName), 404), nil
		}
	}

	return endpoint.handler(currentContext, request)
}

func (api ApiHolder) findEndpoint(method string, path string) (apiEndpoint, bool) {
	for _, v := range api.apiEndpoints {
		if v.method == method && v.pathRegex.MatchString(path) {
			return v, true
		}
	}

	return apiEndpoint{}, false
}
