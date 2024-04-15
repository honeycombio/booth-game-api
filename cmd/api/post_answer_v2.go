package main

import (
	"context"
	"fmt"
	"net/http"
	"observaquiz_lambda/cmd/api/deepchecks"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)


func respondToAnswerV2(currentContext context.Context, questionDefinition Question, answer AnswerBody) (response *responseToAnswer, errorResponse *errorResponseType) {
	postQuestionSpan := trace.SpanFromContext(currentContext)

	var question string = questionDefinition.Question
	var promptSpec AnswerResponsePrompt = questionDefinition.AnswerResponsePrompt
	postQuestionSpan.SetAttributes(attribute.String("app.post_answer.question", question))

	/* now use that definition to construct a prompt */
	openaiMessages, fullPrompt := constructPrompt(promptSpec, question, answer.Answer)
	postQuestionSpan.SetAttributes(attribute.String("app.llm.input", answer.Answer),
		attribute.String("app.llm.full_prompt", fullPrompt))

	/* now call OpenAI */
	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	openAIConfig := openai.DefaultConfig(settings.OpenAIKey)
	openAIConfig.HTTPClient = &httpClient
	client := openai.NewClientWithConfig(openAIConfig)

	startTime := time.Now()
	model := openai.GPT3Dot5Turbo1106
	responseType := openai.ChatCompletionResponseFormatTypeJSONObject // openai.ChatCompletionResponseFormatTypeText
	postQuestionSpan.SetAttributes(attribute.String("app.llm.responseType", fmt.Sprintf("%v", responseType)))
	openaiChatCompletionResponse, err := client.CreateChatCompletion(
		currentContext,
		openai.ChatCompletionRequest{
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: responseType,
			},
			MaxTokens: 2000,
			Model:     model,
			Messages:  openaiMessages,
		},
	)
	if err != nil {
		postQuestionSpan.RecordError(err,
			trace.WithAttributes(
				attribute.String("error.message", "Failure talking to OpenAI")))
		postQuestionSpan.SetAttributes(attribute.String("error.message", "Failure talking to OpenAI"))
		postQuestionSpan.SetStatus(codes.Error, err.Error())

		return nil, &errorResponseType{message: "Could not reach LLM. No fallback in place", statusCode: 500}
	}

	addLlmResponseAttributesToSpan(postQuestionSpan, openaiChatCompletionResponse)
	llmResponse := openaiChatCompletionResponse.Choices[0].Message.Content

	/* report for analysis */

	interactionReported := deepchecks.DeepChecksAPI{ApiKey: settings.DeepchecksApiKey}.ReportInteraction(currentContext, deepchecks.LLMInteractionDescription{
		FullPrompt: fullPrompt,
		Input:      answer.Answer,
		Output:     llmResponse,
		StartedAt:  startTime,
		FinishedAt: time.Now(),
		Model:      model,
	})
	// try to unmarshal the response as JSON and get a score and a response. Otherwise, fall back to treating it as a string and defaulting the score

	parsedLlmResponse, err := parseLLMResponse(currentContext, llmResponse)
	if err != nil {
		return nil, &errorResponseType{message: "Could not parse LLM response", statusCode: 500}
	}

	return &responseToAnswer{response: parsedLlmResponse.Response, score: parsedLlmResponse.Score, evaluationId: interactionReported.EvaluationId}, nil
}
