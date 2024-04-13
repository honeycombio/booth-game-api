package deepchecks

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Annotation string

const (
	Good    Annotation = "good"
	Bad     Annotation = "bad"
	Unknown Annotation = "unknown"
)

// https://llmdocs.deepchecks.com/reference/updateinteractions
type LLMInteractionOpinionReport struct {
	EvaluationId string     `json:"evaluation_id"`
	AppVersionId string     `json:"app_version_id"`
	Annotation   Annotation `json:"annotation"`
}

type annotationForDeepChecks struct {
	Annotation Annotation `json:"annotation"`
}

type OpinionReported struct {
	Reported bool `json:"reported"`
	Success  bool `json:"success"`
	Message string `json:"message"`
}


func (settings DeepChecksAPI) ReportOpinion(currentContext context.Context, interactionOpinion LLMInteractionOpinionReport) (result OpinionReported) {
	trace.SpanFromContext(currentContext).SetAttributes(attribute.Bool("app.feature_flag.send_to_deepchecks", FeatureFlag_SendToDeepchecks))
	if !FeatureFlag_SendToDeepchecks {
		return OpinionReported{Reported: false, Success: true}
	}

	currentContext, span := tracer().Start(currentContext, "Report opinion")
	defer span.End()

	span.SetAttributes(attribute.String("app.llm.evaluation_id", interactionOpinion.EvaluationId),
		attribute.String("app.llm.app_version_id", interactionOpinion.AppVersionId),
		attribute.String("app.llm.annotation", string(interactionOpinion.Annotation)))

	data := annotationForDeepChecks{Annotation: interactionOpinion.Annotation}
	jsonData, _ := json.Marshal(data)

	url := fmt.Sprintf("application_versions/%s/interactions/%s", interactionOpinion.AppVersionId, interactionOpinion.EvaluationId)

	body, err := settings.send_to_deepchecks(currentContext, "PUT", url, jsonData)

	// It is not the most secure thing to return the body. It should go only into the traces.
	return OpinionReported{Reported: true, Success: err == nil, Message: string(body)}
}