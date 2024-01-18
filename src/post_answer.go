package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// JESS: move this to its own file

const appName = "Booth Game Quiz"
const appVersion = "alpha"
const envType = "PROD"

// Define your data structure
type DeepChecksInteraction struct {
	UserInteractionID string          `json:"user_interaction_id"`
	FullPrompt        string          `json:"full_prompt"`
	Input             string          `json:"input"`
	Output            string          `json:"output"`
	AppName           string          `json:"app_name"`
	VersionName       string          `json:"version_name"`
	EnvType           string          `json:"env_type"`
	RawJSONData       json.RawMessage `json:"raw_json_data"`
	StartedAt         time.Time       `json:"started_at"`
	FinishedAt        time.Time       `json:"finished_at"`
}

type LLMInteractionDescription struct {
	FullPrompt string
	Input      string
	Output     string
	StartedAt  time.Time
	FinishedAt time.Time
}

func describeInteractionOnSpan(span trace.Span, interactionDescription LLMInteractionDescription) {
	span.SetAttributes(attribute.String("app.llm.full_prompt", interactionDescription.FullPrompt),
		attribute.String("app.llm.input", interactionDescription.Input),
		attribute.String("app.llm.output", interactionDescription.Output),
		attribute.String("app.llm.started_at", interactionDescription.StartedAt.String()),
		attribute.String("app.llm.finished_at", interactionDescription.FinishedAt.String()))
}

func tellDeepChecksAboutIt(currentContext context.Context, interactionDescription LLMInteractionDescription) {

	currentContext, span := tracer.Start(currentContext, "Report LLM interaction for evaluation")
	defer span.End()

	// JESS: I think we'd rather use the LLM span? but this one will do.
	interactionId := fmt.Sprintf("%s-%s", span.SpanContext().TraceID(), span.SpanContext().SpanID())
	span.SetAttributes(attribute.String("deepchecks.user_interaction_id", interactionId),
		attribute.String("deepchecks.app_name", appName),
		attribute.String("deepchecks.version_name", appVersion),
		attribute.String("deepchecks.env_type", envType))
	describeInteractionOnSpan(span, interactionDescription)

	data := DeepChecksInteraction{
		UserInteractionID: interactionId,
		AppName:           appName,
		VersionName:       appVersion,
		EnvType:           envType,
		FullPrompt:        interactionDescription.FullPrompt,
		Input:             interactionDescription.Input,
		Output:            interactionDescription.Output,
		RawJSONData:       []byte("{}"),
		StartedAt:         interactionDescription.StartedAt,
		FinishedAt:        interactionDescription.FinishedAt,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		span.RecordError(err, trace.WithAttributes(attribute.String("error.message", "Failure marshalling JSON"),
			attribute.String("error.json.input", fmt.Sprintf("%v", data))))
		span.SetStatus(codes.Error, err.Error())
		return
	}

	url := "https://app.llm.deepchecks.com/api/v1/interactions"

	req, _ := http.NewRequestWithContext(currentContext, "POST", url, bytes.NewBuffer(jsonData))

	//req = req.WithContext(currentContext)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", "Basic amVzc2l0cm9uQGhvbmV5Y29tYi5pbw==.b3JnX2hvbmV5Y29tYl9kZXZyZWxfODMxNTY0NjVlOGI4YjlkNA==.8JiwZHT8Di7sZ4o__0WNxw")

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	res, _ := httpClient.Do(req)
	body, _ := io.ReadAll(res.Body)

	span.SetAttributes(attribute.String("response.body", string(body)))

	defer res.Body.Close()

	fmt.Println(string(body))
}
