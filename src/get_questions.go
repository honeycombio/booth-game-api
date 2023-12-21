package main

import (
	"embed"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

//go:embed questions.json
var questionList embed.FS

func getQuestions(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// return a question struct as json

	jsonQuestions, err := questionList.ReadFile("questions.json")
	if err != nil {
		return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Error %v", err), StatusCode: 500}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		Body:       string(jsonQuestions),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
