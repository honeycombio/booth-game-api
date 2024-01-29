package queryData

import (
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"encoding/json"
	"os"

	oteltrace "go.opentelemetry.io/api/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

/**
 * Adapter for the Honeycomb QDAPI, so you could fake it out for testing
 */

type HoneycombQueryDataAPI struct {
	OurHoneycombAPIKey string
	HoneycombApiUrl    string
	Tracer             oteltrace.Tracer
}

func ProductionQueryDataAPI() HoneycombQueryDataAPI {
	return HoneycombQueryDataAPI{
		OurHoneycombAPIKey: os.Getenv("HONEYCOMB_API_KEY"),
		HoneycombApiUrl:    "https://api.honeycomb.io/",
		Tracer:             instrumentation.TracerProvider.Tracer("app.honeycomb_query_data_api"),
	}
}

type CreateQueryResponse struct {
	QueryId string `json:"query"`
}

func (api HoneycombQueryDataAPI) CreateQuery(currentContext context.Context, queryDefinition HoneycombQuery, datasetSlug string) (response CreateQueryResponse, err error) {
	span := api.Tracer.start(currentContext, "Create Honeycomb Query")
	defer span.End()

	queryDefinitionString, err := json.Marshal(queryDefinition)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error marshalling query definition")
		return response, err
	}
	span.SetAttributes(attribute.String("app.request.queryDefinition", string(queryDefinitionString)),
		attribute.String("app.request.datasetSlug", datasetSlug))

	queryId := "hardcoded to 1234"

	span.SetAttributes(attribute.String("app.response.query_id", queryId))
	response = CreateQueryResponse{
		QueryId: queryId,
	}
	return response, nil
}
