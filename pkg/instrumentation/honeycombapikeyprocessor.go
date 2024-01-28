package instrumentation

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

type HoneycombApiKeyProcessor struct{}

const (
	APIKEY_BAGGAGE_NAME        = "boothgame.attendee_apikey"
	EXECUTION_ID_BAGGAGE_NAME  = "app.observaquiz.execution_id"
	EXECUTION_ID_ATTRIBUTE_KEY = "app.observaquiz.execution_id"
)

var _ trace.SpanProcessor = (*HoneycombApiKeyProcessor)(nil)

func NewHoneycombApiKeyProcessor() trace.SpanProcessor {
	return &HoneycombApiKeyProcessor{}
}

func (processor HoneycombApiKeyProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	apikey := baggage.FromContext(ctx).Member(APIKEY_BAGGAGE_NAME)
	span.SetAttributes(attribute.String("app.honeycomb_api_key", apikey.Value()))
	executionId := baggage.FromContext(ctx).Member(EXECUTION_ID_BAGGAGE_NAME)
	span.SetAttributes(attribute.String(EXECUTION_ID_ATTRIBUTE_KEY, executionId.Value()))
}

func (processor HoneycombApiKeyProcessor) OnEnd(span trace.ReadOnlySpan)    {}
func (processor HoneycombApiKeyProcessor) Shutdown(context.Context) error   { return nil }
func (processor HoneycombApiKeyProcessor) ForceFlush(context.Context) error { return nil }

func SetApiKeyInBaggage(ctx context.Context, apikey string, executionId string) (context.Context, error) {
	baggageApikeyMember, err := baggage.NewMember(APIKEY_BAGGAGE_NAME, apikey)
	if err != nil {
		return ctx, err
	}
	baggageExecutionIdMember, err := baggage.NewMember(EXECUTION_ID_BAGGAGE_NAME, executionId)
	if err != nil {
		return ctx, err
	}
	currentBaggage := baggage.FromContext(ctx)
	currentBaggage, err = currentBaggage.SetMember(baggageApikeyMember)
	if err != nil {
		return ctx, err
	}
	currentBaggage, err = currentBaggage.SetMember(baggageExecutionIdMember)
	if err != nil {
		return ctx, err
	}
	return baggage.ContextWithBaggage(ctx, currentBaggage), nil
}
