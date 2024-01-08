package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

type QuestionsResponse struct {
	QuestionSet string     `json:"question_set"`
	Questions   []Question `json:"questions"`
}

type Question struct {
	Id          uuid.UUID `json:"id"`
	Question    string    `json:"question"`
	PromptCheck string    `json:"prompt_check"`
}

func getQuestions(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

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
