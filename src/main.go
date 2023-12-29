package main

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
)

var apiEndpoints = []apiEndpoint{
	{
		"GET",
		"/api/events",
		regexp.MustCompile("^/api/events$"),
		getEvents,
		false,
	},
	{
		"GET",
		"/api/questions",
		regexp.MustCompile("^/api/questions$"),
		getQuestions,
		true,
	},
	{
		"POST",
		"/api/questions/{questionId}/answer",
		regexp.MustCompile("^/api/questions/([^/]+)/answer$"),
		postAnswer,
		true,
	},
}

type apiEndpoint struct {
	method        string
	pathTemplate  string
	pathRegex     *regexp.Regexp
	handler       interface{}
	requiresEvent bool
}

const (
	default_event = "devopsdays_whenever"
)

func Api(context context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	for _, v := range apiEndpoints {
		if v.method == request.RequestContext.HTTP.Method &&
			v.pathRegex.MatchString(request.RequestContext.HTTP.Path) {

			if v.requiresEvent {
				eventName := getEventName(request)
				if _, eventFound := eventQuestions[eventName]; !eventFound {
					return events.APIGatewayV2HTTPResponse{
						Body:       fmt.Sprintf("Couldn't find event name %s", eventName),
						StatusCode: 404,
					}, nil
				}
			}

			return v.handler.(func(events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error))(request)
		}
	}
	methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path

	return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}, nil
}

var settings struct {
	OpenAIKey string `env:"openai_key"`
}

func main() {
	flags.Parse(&settings)
	// print all the environment variables to the console
	settings.OpenAIKey = os.Getenv("openai_key")

	println("OpenAI Key" + settings.OpenAIKey)
	lambda.Start(Api)
}

func getEventName(request events.APIGatewayV2HTTPRequest) string {
	eventName := request.Headers["event-name"]
	if eventName == "" {
		eventName = default_event
	}

	return eventName
}
