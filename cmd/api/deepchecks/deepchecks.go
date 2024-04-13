package deepchecks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"observaquiz_lambda/pkg/instrumentation"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const FeatureFlag_SendToDeepchecks = true

const appName = "Booth Game Quiz"
const appVersion = "alpha"

// tracerProvider is initialized in main 
var tracer = instrumentation.TracerProvider.Tracer("observaquiz-bff/deepchecks")

type DeepChecksAPI struct {
	ApiKey string
}

// Define your data structure
type DeepChecksCreateInteractions struct {
	AppName      string                  `json:"app_name"`
	VersionName  string                  `json:"version_name"`
	EnvType      string                  `json:"env_type"`
	Interactions []DeepChecksInteraction `json:"interactions"`
}

type DeepChecksInteraction struct {
	UserInteractionID string            `json:"user_interaction_id"`
	FullPrompt        string            `json:"full_prompt"`
	Input             string            `json:"input"`
	Output            string            `json:"output"`
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
	Model      string
}

func describeInteractionOnSpan(span trace.Span, interactionDescription LLMInteractionDescription) {
	span.SetAttributes(attribute.String("app.llm.full_prompt", interactionDescription.FullPrompt),
		attribute.String("app.llm.input", interactionDescription.Input),
		attribute.String("app.llm.output", interactionDescription.Output),
		attribute.String("app.llm.started_at", interactionDescription.StartedAt.String()),
		attribute.String("app.llm.finished_at", interactionDescription.FinishedAt.String()))
}

type InteractionReported struct {
	EvaluationId string
}

func (settings DeepChecksAPI) ReportInteraction(currentContext context.Context, interactionDescription LLMInteractionDescription) (result InteractionReported) {
	trace.SpanFromContext(currentContext).SetAttributes(attribute.Bool("app.feature_flag.send_to_deepchecks", FeatureFlag_SendToDeepchecks))
	if !FeatureFlag_SendToDeepchecks {
		return InteractionReported{}
	}

	currentContext, span := tracer.Start(currentContext, "Report LLM interaction for evaluation")
	defer span.End()

	// JESS: rename this environment variable.
	environment := os.Getenv("DEEPCHECKS_ENV_TYPE")
	span.SetAttributes(attribute.String("env.deepchecks_env_type", environment))

	// JESS: I think we'd rather use the LLM span? but this one will do.
	interactionId := fmt.Sprintf("%s-%s", span.SpanContext().TraceID(), span.SpanContext().SpanID())
	result = InteractionReported{EvaluationId: interactionId}
	span.SetAttributes(attribute.String("deepchecks.user_interaction_id", interactionId),
		attribute.String("deepchecks.app_name", appName),
		attribute.String("deepchecks.version_name", appVersion),
		attribute.String("deepchecks.env_type", "PROD"),
		attribute.String("deepchecks.custom_props.environment", environment))
	describeInteractionOnSpan(span, interactionDescription)

	interaction := DeepChecksInteraction{
		UserInteractionID: interactionId,
		Input:             interactionDescription.Input,
		FullPrompt:        interactionDescription.FullPrompt,
		Output:            interactionDescription.Output,
		StartedAt:         interactionDescription.StartedAt,
		RawJSONData:       []byte("{}"),
		FinishedAt:        interactionDescription.FinishedAt,
		CustomProps: map[string]string{"Environment": environment,
			"Model": interactionDescription.Model},
	}

	data := DeepChecksCreateInteractions{
		AppName:      appName,
		VersionName:  appVersion,
		EnvType:      "PROD",
		Interactions: []DeepChecksInteraction{interaction},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		span.RecordError(err, trace.WithAttributes(attribute.String("error.message", "Failure marshalling JSON"),
			attribute.String("error.json.input", fmt.Sprintf("%v", data))))
		span.SetStatus(codes.Error, err.Error())
		return InteractionReported{}
	}

	body, _ := settings.send_to_deepchecks(currentContext, "POST", "interactions", jsonData)

	fmt.Println(string(body))
	return
}

func (settings DeepChecksAPI) send_to_deepchecks(currentContext context.Context, method string, relativeUrl string, jsonData []byte) (body []byte, err error) {
	span := trace.SpanFromContext(context.Background())

	url := "https://app.llm.deepchecks.com/api/v1/" + relativeUrl
	span.SetAttributes(attribute.String("request.body", string(jsonData)))
	req, _ := http.NewRequestWithContext(currentContext, method, url, bytes.NewBuffer(jsonData))

	//req = req.WithContext(currentContext)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", "Basic "+settings.ApiKey)
	span.SetAttributes(attribute.String("app.deepchecks.apikey_masked", maskString(settings.ApiKey)))

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	res, err := httpClient.Do(req)

	if err != nil {
		span.RecordError(err, trace.WithAttributes(attribute.String("error.message", "Failure talking to DeepChecks")))
		// the world does not end
		return
	}

	body, err = io.ReadAll(res.Body)
	defer res.Body.Close()

	span.SetAttributes(attribute.String("response.body", string(body)))
	return
}

func maskString(input string) string {
	// Define the masking character
	maskChar := '*'

	// Calculate the number of characters to mask (80% of the string length)
	maskLength := int(float64(len(input)) * 0.8)

	// Split the string into runes (Unicode characters)
	chars := []rune(input)

	// Mask the first 80% of characters
	for i := 0; i < maskLength; i++ {
		chars[i] = maskChar
	}

	// Convert the masked slice of runes back to a string
	return string(chars)
}
