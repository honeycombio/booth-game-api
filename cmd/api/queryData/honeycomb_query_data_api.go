package queryData

import (
	"booth_game_lambda/pkg/instrumentation"
	"context"
	"encoding/json"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
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

type DefineQueryResponse struct {
	QueryId string `json:"query"`
}

func (api HoneycombQueryDataAPI) CreateQuery(currentContext context.Context, queryDefinition HoneycombQuery, datasetSlug string) (response DefineQueryResponse, err error) {
	currentContext, span := api.Tracer.Start(currentContext, "Create Honeycomb Query")
	defer span.End()

	queryDefinitionString, err := json.Marshal(queryDefinition)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error marshalling query definition")
		return response, err
	}
	span.SetAttributes(attribute.String("app.request.payload", string(queryDefinitionString)),
		attribute.String("app.request.datasetSlug", datasetSlug))

	queryId := "hardcoded to 1234"

	span.SetAttributes(attribute.String("app.response.query_id", queryId))
	response = DefineQueryResponse{
		QueryId: queryId,
	}
	return response, nil
}

type StartQueryResponse struct {
	ResultId string `json:"result_id"`
}

type StartHoneycombQuery struct {
	QueryId       string `json:"query_id"`
	DisableSeries bool   `json:"disable_series"`
	Limit         int    `json:"limit"`
}

/**
 * https://docs.honeycomb.io/api/tag/Query-Data#operation/createQueryResult
 */
func (api HoneycombQueryDataAPI) StartQuery(currentContext context.Context, queryId string, datasetSlug string) (response StartQueryResponse, err error) {
	currentContext, span := api.Tracer.Start(currentContext, "Start Honeycomb Query Run")
	defer span.End()
	span.SetAttributes(attribute.String("app.request.query_id", queryId))

	startQueryInput := StartHoneycombQuery{
		QueryId:       queryId,
		DisableSeries: true,
		Limit:         100,
	}

	startQueryInputString, err := json.Marshal(startQueryInput)
	span.SetAttributes(attribute.String("app.request.payload", string(startQueryInputString)),
		attribute.String("app.request.datasetSlug", datasetSlug))

	//TODO: do the thing

	resultId := "hard coded to 5678"
	span.SetAttributes(attribute.String("app.response.result_id", resultId))

	response = StartQueryResponse{
		ResultId: resultId,
	}
	return response, nil
}
