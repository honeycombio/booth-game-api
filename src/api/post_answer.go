package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var postAnswerEndpoint = apiEndpoint{
	"POST",
	"/api/questions/{questionId}/answer",
	regexp.MustCompile("^/api/questions/([^/]+)/answer$"),
	postAnswer,
	true,
}

const (
	start_system_prompt = "You are a quizmaster, who is also an Observability evangelist, validating people's answers who gives a score between 0 and 100. You provide the output as a json object in the format { \"score\": \"{score}\", \"better_answer\": \"{an answer that would improve the score}\"}"
)

type AnswerBody struct {
	Answer string `json:"answer"`
}

func postAnswer(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {

	currentContext, postQuestionSpan := tracer.Start(currentContext, "Answer Question")
	defer postQuestionSpan.End()
	eventName := getEventName(request)

	path := request.RequestContext.HTTP.Path
	pathSplit := strings.Split(path, "/")
	questionId := pathSplit[3]

	var prompt string
	var question string
	var bestanswer string
	eventQuestions := eventQuestions[eventName]

	for _, v := range eventQuestions {
		if v.Id.String() == questionId {
			prompt = v.PromptCheck
			question = v.Question
			bestanswer = v.BestAnswer
			break
		}
	}

	answer := AnswerBody{}
	err := json.Unmarshal([]byte(request.Body), &answer)
	if err != nil {
		newErr := fmt.Errorf("error unmarshalling answer: %w\n request body: %s", err, request.Body)
		postQuestionSpan.RecordError(newErr)

		return events.APIGatewayV2HTTPResponse{Body: "Internal Server Error :-P", StatusCode: 500}, nil
	}

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	openAIConfig := openai.DefaultConfig(settings.OpenAIKey)
	openAIConfig.HTTPClient = &httpClient

	client := openai.NewClientWithConfig(openAIConfig)

	full_prompt := fmt.Sprintf("%v %v", prompt, answer.Answer)
	postQuestionSpan.SetAttributes(attribute.String("app.llm.input", answer.Answer),
		attribute.String("app.llm.full_prompt", full_prompt))

	startTime := time.Now()
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
					Content: fmt.Sprintf("%v %v", "You're looking for ", prompt),
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fmt.Sprintf("This is the question: %s", question),
				},
				{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fmt.Sprintf("This is ideal answer: %s", bestanswer),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("This is the contestant's answer: %s", answer.Answer),
				},
			},
		},
	)
	if err != nil {
		postQuestionSpan.RecordError(err,
			trace.WithAttributes(
				attribute.String("error.message", "Failure talking to OpenAI")))
		postQuestionSpan.SetAttributes(attribute.String("error.message", "Failure talking to OpenAI"))
		postQuestionSpan.SetStatus(codes.Error, err.Error())

		return events.APIGatewayV2HTTPResponse{Body: `{ "message": "Could not reach LLM. No fallback in place", 
		"trace.trace_id": "` + postQuestionSpan.SpanContext().TraceID().String() +
			`", "trace.span_id":"` + postQuestionSpan.SpanContext().SpanID().String() +
			`", "dataset": "` + HoneycombDatasetName + `" }`, StatusCode: 500}, nil
	}
	llmResponse := resp.Choices[0].Message.Content

	tellDeepChecksAboutIt(currentContext, LLMInteractionDescription{
		FullPrompt: full_prompt,
		Input:      answer.Answer,
		Output:     llmResponse,
		StartedAt:  startTime,
		FinishedAt: time.Now(),
	})

	postQuestionSpan.SetAttributes(attribute.String("app.llm.output", llmResponse))
	return events.APIGatewayV2HTTPResponse{Body: llmResponse, StatusCode: 200}, nil
}
