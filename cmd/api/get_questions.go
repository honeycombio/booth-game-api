package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

var getQuestionsEndpoint = apiEndpoint{
	"GET",
	"/api/questions",
	regexp.MustCompile("^/api/questions$"),
	getQuestions,
	true,
}

type QuestionsResponse struct {
	QuestionSet string     `json:"question_set"`
	Questions   []Question `json:"questions"`
}

type Question struct {
	Id                   uuid.UUID            `json:"id"`
	Question             string               `json:"question"`
	AnswerResponsePrompt AnswerResponsePrompt `json:"prompt"`
	Version              string               `json:"version"`
}

type AnswerResponsePrompt struct {
	SystemPrompt string                         `json:"system"`
	Examples     []AnswerResponsePromptExamples `json:"examples"`
}

type AnswerResponsePromptExamples struct {
	ExampleAnswer   string `json:"answer"`
	ExampleResponse string `json:"response"`
}

func getQuestions(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	eventName := getEventName(request)

	questions := eventQuestions[eventName]

	questionResponse := QuestionsResponse{
		QuestionSet: eventName,
		Questions:   questions,
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
