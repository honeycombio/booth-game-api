package main

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
)

var api = ApiHolder{
	apiEndpoints: []apiEndpoint{
		getEventsEndpoint,
		getQuestionsEndpoint,
		postAnswerEndpoint,
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
			return events.APIGatewayV2HTTPResponse{
				Body:       fmt.Sprintf("Couldn't find event name %s", eventName),
				StatusCode: 404,
			}, nil
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
