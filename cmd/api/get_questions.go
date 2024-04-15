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
	Version              string               `json:"version"`
	AnswerResponsePrompt AnswerResponsePrompt `json:"prompt"`  // V1 only
	PromptsV2            PromptsV2            `json:"prompts"` // V2 only
	Scoring              ScoringThings        `json:"scoring"` // V2 only
}

type PromptsV2 struct {
	ResponsePrompt string `json:"response_prompt"`
	CategoryPrompt string `json:"category_prompt"`
}

type ScoringThings struct {
	ScoringPrompts []ScoringPrompt `json:"scoring_prompts"`
	PointyWords    []string        `json:"pointy_words"`
}

type ScoringPrompt struct {
	Prompt       string `json:"prompt"`
	MaximumScore int    `json:"maximum_score"`
	Description  string `json:"description"`
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
