package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const appName = "Booth Game Quiz"
const appVersion = "alpha"

// Define your data structure
type DeepChecksInteraction struct {
	UserInteractionID string            `json:"user_interaction_id"`
	FullPrompt        string            `json:"full_prompt"`
	Input             string            `json:"input"`
	Output            string            `json:"output"`
	AppName           string            `json:"app_name"`
	VersionName       string            `json:"version_name"`
	EnvType           string            `json:"env_type"`
	RawJSONData       json.RawMessage   `json:"raw_json_data"`
	StartedAt         time.Time         `json:"started_at"`
	FinishedAt        time.Time         `json:"finished_at"`
	CustomProps       map[string]string `json:"custom_props"`
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

	// JESS: rename this environment variable.
	environment := os.Getenv("DEEPCHECKS_ENV_TYPE")
	span.SetAttributes(attribute.String("env.deepchecks_env_type", environment))

	// JESS: I think we'd rather use the LLM span? but this one will do.
	interactionId := fmt.Sprintf("%s-%s", span.SpanContext().TraceID(), span.SpanContext().SpanID())
	span.SetAttributes(attribute.String("deepchecks.user_interaction_id", interactionId),
		attribute.String("deepchecks.app_name", appName),
		attribute.String("deepchecks.version_name", appVersion),
		attribute.String("deepchecks.env_type", "PROD"),
		attribute.String("deepchecks.custom_props.environment", environment))
	describeInteractionOnSpan(span, interactionDescription)

	data := DeepChecksInteraction{
		UserInteractionID: interactionId,
		AppName:           appName,
		VersionName:       appVersion,
		EnvType:           "PROD",
		FullPrompt:        interactionDescription.FullPrompt,
		Input:             interactionDescription.Input,
		Output:            interactionDescription.Output,
		RawJSONData:       []byte("{}"),
		StartedAt:         interactionDescription.StartedAt,
		FinishedAt:        interactionDescription.FinishedAt,
		CustomProps:       map[string]string{"Environment": environment},
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
