package main

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

//go:embed questions.json
var questionList embed.FS

type question struct {
	Question       string    `json:"question"`
	PromptCheck    string    `json:"prompt_check"`
	QuestionNumber int       `json:"question_number"`
	Id             uuid.UUID `json:"id"`
}

type questionResponse struct {
	QuestionSet string     `json:"question_set"`
	Questions   []question `json:"questions"`
}

func getQuestions(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// return a question struct as json

	jsonQuestions, err := questionList.ReadFile("questions.json")
	if err != nil {
		return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Error %v", err), StatusCode: 500}, nil
	}
	var questions []question
	json.Unmarshal(jsonQuestions, &questions)

	questionResponse := questionResponse{
		QuestionSet: "DevRel-Testing",
		Questions:   questions}

	questionResponseJson, err := json.Marshal(questionResponse)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Error %v", err), StatusCode: 500}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		Body:       string(questionResponseJson),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}
