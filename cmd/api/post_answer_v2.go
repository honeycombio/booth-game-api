package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"observaquiz_lambda/cmd/api/deepchecks"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type openaiApi struct {
   model string
   client *openai.Client
}

func newOpenaiApi(model string, key string) *openaiApi {
	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	openAIConfig := openai.DefaultConfig(key)
	openAIConfig.HTTPClient = &httpClient
	client := openai.NewClientWithConfig(openAIConfig)

   return &openaiApi{model: model, client: client}
}

type chatResult struct {
	   responseContent string
	   evaluationId string
}

func (api openaiApi) chat(context context.Context, theirAnswer string, prompt string, output *chatResult) (err error) {
	context, span := tracer.Start(context, "chat with AI")
	defer span.End()
	startTime := time.Now()
	model := openai.GPT3Dot5Turbo1106
	responseType := openai.ChatCompletionResponseFormatTypeJSONObject // openai.ChatCompletionResponseFormatTypeText
	span.SetAttributes(attribute.String("app.llm.responseType", fmt.Sprintf("%v", responseType)))
	openaiMessage := openai.ChatCompletionMessage{Role: "system", Content: prompt}

	openaiChatCompletionResponse, err := api.client.CreateChatCompletion(
		context,
		openai.ChatCompletionRequest{
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: responseType,
			},
			MaxTokens: 2000,
			Model:     model,
			Messages:  []openai.ChatCompletionMessage{openaiMessage},
		},
	)
	if err != nil {
		span.RecordError(err,
			trace.WithAttributes(
				attribute.String("error.message", "Failure talking to OpenAI")))
		span.SetAttributes(attribute.String("error.message", "Failure talking to OpenAI"))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	addLlmResponseAttributesToSpan(span, openaiChatCompletionResponse)
	llmResponse := openaiChatCompletionResponse.Choices[0].Message.Content

	/* report for analysis */

	interactionReported := deepchecks.DeepChecksAPI{ApiKey: settings.DeepchecksApiKey}.ReportInteraction(context, deepchecks.LLMInteractionDescription{
		FullPrompt: prompt,
		Input:      theirAnswer,
		Output:     llmResponse,
		StartedAt:  startTime,
		FinishedAt: time.Now(),
		Model:      model,
	})

	output.responseContent = llmResponse
	output.evaluationId = interactionReported.EvaluationId

	return
}

type CategoryResult struct {
	Category string `json:"category"`
	Confidence string `json:"confidence"`
	Reasoning string `json:"reasoning"`
}

func respondToAnswerV2(currentContext context.Context, questionDefinition Question, answer AnswerBody) (response *responseToAnswer, errorResponse *errorResponseType) {
	span := trace.SpanFromContext(currentContext)

	var question string = questionDefinition.Question
	span.SetAttributes(attribute.String("app.post_answer.question", question))

	/* now use that definition to construct a prompt */
	categoryPrompt := strings.Replace(questionDefinition.PromptsV2.CategoryPrompt, "THEIR_ANSWER", answer.Answer, -1)
	span.SetAttributes(attribute.String("app.llm.input", answer.Answer),
		attribute.String("app.llm.category_prompt", categoryPrompt))

	llmApi := newOpenaiApi("GPT3Dot5Turbo1106", settings.OpenAIKey)

	categoryResponse := chatResult{}
	err := llmApi.chat(currentContext, answer.Answer, categoryPrompt, &categoryResponse)
	if err != nil {
		return nil, &errorResponseType{message: "Could not reach LLM. No fallback in place", statusCode: 500}
		
	}
	span.SetAttributes(attribute.String("app.llm.category_response", categoryResponse.responseContent));

	categoryResult := CategoryResult{}
	err = json.Unmarshal([]byte(categoryResponse.responseContent), &categoryResult)
	if err != nil {
		return nil, &errorResponseType{message: "Could not parse category response", statusCode: 500}
	}
	span.SetAttributes(attribute.String("app.llm.assigned_category", categoryResult.Category))


	return &responseToAnswer{response: categoryResult.Category, score: 10, evaluationId: categoryResponse.evaluationId}, nil
}
