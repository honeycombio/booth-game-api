package main

import (
	"embed"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

type Question struct {
	Id          uuid.UUID `json:"id"`
	Question    string    `json:"question"`
	PromptCheck string    `json:"prompt_check"`
}

//go:embed questions.json
var questionList embed.FS

var questions, _ = questionList.ReadFile("questions.json")

func getQuestions(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// return a question struct as json

	return events.APIGatewayV2HTTPResponse{
		Body:       string(questions),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
