package deepchecks

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Opinion string

const (
    Good    Opinion = "good"
    Bad     Opinion = "bad"
    Unknown Opinion = "unknown"
)

// https://llmdocs.deepchecks.com/reference/updateinteractions
type LLMInteractionOpinionReport struct {
	EvaluationId string `json:"evaluation_id"`
	AppVersionId string `json:"app_version_id"`
	Opinion   Opinion `json:"annotation"`
}

type annotationForDeepChecks struct {
	Annotation Opinion `json:"annotation"`
}

type OpinionReported struct {
	Reported bool `json:"reported"`
	Success bool `json:"success"`
}

func (settings DeepChecksAPI) ReportOpinion(currentContext context.Context, interactionOpinion LLMInteractionOpinionReport) (result OpinionReported) {
	trace.SpanFromContext(currentContext).SetAttributes(attribute.Bool("app.feature_flag.send_to_deepchecks", FeatureFlag_SendToDeepchecks))
	if !FeatureFlag_SendToDeepchecks {
		return OpinionReported{ Reported: false, Success: true}
	}

	currentContext, span := tracer.Start(currentContext, "Report LLM interaction for evaluation")
	defer span.End()

	span.SetAttributes(attribute.String("app.llm.evaluation_id", interactionOpinion.EvaluationId),
		attribute.String("app.llm.app_version_id", interactionOpinion.AppVersionId),
		attribute.String("app.llm.annotation", string(interactionOpinion.Opinion)))

	data := annotationForDeepChecks{Annotation: interactionOpinion.Opinion}
    jsonData, _ := json.Marshal(data)

	url := fmt.Sprintf("application_versions/%s/interactions/%s", interactionOpinion.AppVersionId, interactionOpinion.EvaluationId)

	_, err := settings.send_to_deepchecks(currentContext, "PUT", url, jsonData);

	return OpinionReported{ Reported: true, Success: err == nil}
}