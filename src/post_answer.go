package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var postAnswerEndpoint = apiEndpoint{
	"POST",
	"/api/questions/{questionId}/answer",
	regexp.MustCompile("^/api/questions/([^/]+)/answer$"),
	postAnswer,
	true,
}

const (
	start_system_prompt = "You are a quizmaster validating people's answers who gives a score between 0 and 100. You provide the output as a json object in the format { \"score\": \"{score}\", \"better_answer\": \"{an answer that would improve the score}\"}"
)

type AnswerBody struct {
	Answer string `json:"answer"`
}

func postAnswer(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	currentContext, postQuestionSpan := tracer.Start(currentContext, "postAnswer")
	defer postQuestionSpan.End()
	eventName := getEventName(request)

	path := request.RequestContext.HTTP.Path
	pathSplit := strings.Split(path, "/")
	questionId := pathSplit[2]

	var prompt string
	var question string
	eventQuestions := eventQuestions[eventName]

	for _, v := range eventQuestions {
		if v.Id.String() == questionId {
			prompt = v.PromptCheck
			question = v.Question
			break
		}
	}

	answer := AnswerBody{}
	err := json.Unmarshal([]byte(request.Body), &answer)
	if err != nil {
		fmt.Printf("Error unmarshalling answer: %v\n", err)
		return events.APIGatewayV2HTTPResponse{Body: "Internal Server Error", StatusCode: 500}, nil
	}

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	openAIConfig := openai.DefaultConfig(settings.OpenAIKey)
	openAIConfig.HTTPClient = &httpClient

	client := openai.NewClientWithConfig(openAIConfig)

	resp, err := client.CreateChatCompletion(
		currentContext,
		openai.ChatCompletionRequest{
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
			MaxTokens: 2000,
			Model:     openai.GPT3Dot5Turbo1106,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: start_system_prompt,
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fmt.Sprintf("%v %v", "I'm looking for ", prompt),
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: question,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: answer.Answer,
				},
			},
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return events.APIGatewayV2HTTPResponse{Body: "Internal Server Error", StatusCode: 500}, nil
	}
	return events.APIGatewayV2HTTPResponse{Body: resp.Choices[0].Message.Content, StatusCode: 200}, nil
}
