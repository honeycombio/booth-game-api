package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jessevdk/go-flags"
)

var variableRegex = regexp.MustCompile("\\{.*?\\}")

var apiEndpoints = []apiEndpoint{
	{"GET", "/api/questions", regexp.MustCompile("^/api/questions$"), getQuestions},
	{"POST", "/api/questions/{questionId}/answer", regexp.MustCompile("^/api/questions/([^/]+)/answer$"), postAnswer},
}
var apis = map[string]interface{}{
	"GET /api/questions": getQuestions,
	"POST /api/questions/{questionId}/answer": postAnswer,
}

type apiEndpoint struct {
	method string
	pathTemplate string
	pathRegex *regexp.Regexp
	handler interface{}
}

type router struct { 
	regexApiRoutes map[string]interface{}
}

var router = router{}

func Api(context context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path

	for k, v := range apis {

	
	switch methodPath {
	case "GET /api/questions":
		return getQuestions(request)
	case "POST /api/questions/"
	default:
		return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}, nil
	}
}

var settings struct {
	OpenAIKey string `json:"openai_key"`
}

func main() {
	flags.ParseArgs(&settings, os.Environ())
	for k, v := range apis {
		k = variableRegex.ReplaceAllString(k, "[^/]+")
	}
	lambda.Start(Api)
}
