package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/sdk/trace"
)

type HoneycombApiKeyProcessor struct{}

const (
	APIKEY_BAGGAGE_NAME = "boothgame.attendee_apikey"
)

var _ trace.SpanProcessor = (*HoneycombApiKeyProcessor)(nil)

func NewHoneycombApiKeyProcessor() trace.SpanProcessor {
	return &HoneycombApiKeyProcessor{}
}

func (processor HoneycombApiKeyProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	apikey := baggage.FromContext(ctx).Member(APIKEY_BAGGAGE_NAME)
	span.SetAttributes(attribute.String("app.honeycomb_api_key", apikey.String()))
}

func (processor HoneycombApiKeyProcessor) OnEnd(span trace.ReadOnlySpan)    {}
func (processor HoneycombApiKeyProcessor) Shutdown(context.Context) error   { return nil }
func (processor HoneycombApiKeyProcessor) ForceFlush(context.Context) error { return nil }

func SetApiKeyInBaggage(ctx context.Context, apikey string) context.Context {
	baggageApikeyMember, err := baggage.NewMember(APIKEY_BAGGAGE_NAME, apikey)
	if err != nil {
		panic(err)
	}
	currentBaggage := baggage.FromContext(ctx)
	currentBaggage.SetMember(baggageApikeyMember)
	return baggage.ContextWithBaggage(ctx, currentBaggage)
}
