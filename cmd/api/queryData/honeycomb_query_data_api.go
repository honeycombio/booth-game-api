package queryData

import (
	"booth_game_lambda/pkg/instrumentation"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

/**
 * Adapter for the Honeycomb QDAPI, so you could fake it out for testing
 */

type honeycombQueryDataAPI struct {
	OurHoneycombAPIKey string
	HoneycombApiUrl    string
	Tracer             oteltrace.Tracer
}

func productionQueryDataAPI(apikey string) honeycombQueryDataAPI {
	return honeycombQueryDataAPI{
		OurHoneycombAPIKey: apikey,
		HoneycombApiUrl:    "https://api.honeycomb.io/1", // our DevRel team is in US region.
		Tracer:             instrumentation.TracerProvider.Tracer("app.honeycomb_query_data_api"),
	}
}

type DefineQueryResponse struct {
	QueryId string `json:"query"`
}

func (api honeycombQueryDataAPI) sendToHoneycomb(currentContext context.Context, method string, relativeUrl string, payload []byte) (response []byte, err error) {
	span := oteltrace.SpanFromContext(currentContext)
	span.SetAttributes(attribute.String("app.request.url", api.HoneycombApiUrl+relativeUrl))

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	url := api.HoneycombApiUrl + relativeUrl
	req, _ := http.NewRequestWithContext(currentContext, method, url, bytes.NewBuffer(payload))

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-honeycomb-team", api.OurHoneycombAPIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(attribute.Int("app.response.status", resp.StatusCode))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = errors.New("Honeycomb API returned " + resp.Status)
		return nil, err
	}

	// Read the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.String("app.response.body", string(bodyBytes)))

	return bodyBytes, nil
}

type honeycombCreateQueryResponse struct {
	QueryId string `json:"id"`
}

func (api honeycombQueryDataAPI) CreateQuery(currentContext context.Context, queryDefinition HoneycombQuery, datasetSlug string) (response DefineQueryResponse, err error) {
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

	bodyBytes, err := api.sendToHoneycomb(currentContext, "POST", "/queries/"+datasetSlug, queryDefinitionString)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error creating query")
		return response, err
	}
	output := honeycombCreateQueryResponse{}
	err = json.Unmarshal(bodyBytes, &output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error unmarshalling response")
		return response, err
	}
	queryId := output.QueryId

	span.SetAttributes(attribute.String("app.response.query_id", queryId))
	response = DefineQueryResponse{
		QueryId: queryId,
	}
	return response, nil
}

type startQueryResponseBody struct {
	ResultId string `json:"id"`
	Links    links  `json:"links"`
}

type startHoneycombQueryRequestBody struct {
	QueryId       string `json:"query_id"`
	DisableSeries bool   `json:"disable_series"`
	Limit         int    `json:"limit"`
}

/**
 * https://docs.honeycomb.io/api/tag/Query-Data#operation/createQueryResult
 */
func (api honeycombQueryDataAPI) StartQuery(currentContext context.Context, queryId string, datasetSlug string) (response startQueryResponseBody, err error) {
	currentContext, span := api.Tracer.Start(currentContext, "Start Honeycomb Query Run")
	defer span.End()
	span.SetAttributes(attribute.String("app.request.query_id", queryId))

	startQueryInput := startHoneycombQueryRequestBody{
		QueryId:       queryId,
		DisableSeries: true,
		Limit:         100,
	}

	startQueryInputString, err := json.Marshal(startQueryInput)
	span.SetAttributes(attribute.String("app.request.payload", string(startQueryInputString)),
		attribute.String("app.request.datasetSlug", datasetSlug))

	startQueryJson, err := api.sendToHoneycomb(currentContext, "POST", "/query_results/"+datasetSlug, startQueryInputString)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return response, err
	}
	startQueryResponse := startQueryResponseBody{}
	err = json.Unmarshal(startQueryJson, &startQueryResponse)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error unmarshalling response")
		return response, err
	}

	span.SetAttributes(attribute.String("app.response.queryURL", startQueryResponse.Links.QueryURL),
		attribute.String("app.response.graphImageURL", startQueryResponse.Links.GraphImageURL))

	resultId := startQueryResponse.ResultId
	span.SetAttributes(attribute.String("app.response.result_id", resultId))

	response = startQueryResponseBody{
		ResultId: resultId,
	}
	return response, nil
}

type honeycombQueryData struct {
	Data []map[string]interface{} // values can be numbers or strings
}

type Data struct {
	Results []struct {
		Data map[string]interface{} `json:"data"`
	} `json:"results"`
}

type links struct {
	QueryURL      string `json:"query_url"`
	GraphImageURL string `json:"graph_image_url"`
}

type getQueryResultResponse struct {
	Query    interface{} `json:"query"`
	ID       string      `json:"id"`
	Complete bool        `json:"complete"`
	Data     Data        `json:"data"`
	Links    links       `json:"links"`
}

func (api honeycombQueryDataAPI) GiveMeTheData(currentContext context.Context, resultId string, datasetSlug string) (response honeycombQueryData, err error) {
	currentContext, span := api.Tracer.Start(currentContext, "Get Honeycomb Query Result")
	defer span.End()
	span.SetAttributes(attribute.String("app.request.result_id", resultId))

	// TODO: the thing
	// 1. Poll the result URL until it's done
	queryResult := getQueryResultResponse{}
	pollCount := 0
	for {
		pollCount++
		bodyBytes, err := api.sendToHoneycomb(currentContext, "GET", fmt.Sprintf("/query_results/%s/%s", datasetSlug, resultId), []byte{})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Error fetching query results")
			return response, err
		}

		// 2. Get the data
		err = json.Unmarshal(bodyBytes, &queryResult)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Error unmarshalling response")
			return response, err
		}

		if queryResult.Complete {
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	// 3. Return it

	span.SetAttributes(attribute.Int("app.queryData.pollCount", pollCount),
		attribute.String("app.response.queryURL", queryResult.Links.QueryURL),
		attribute.String("app.response.graphImageURL", queryResult.Links.GraphImageURL))

	// Go doesn't map over slices, WTAF???!!???!
	original := queryResult.Data.Results
	mapped := make([]map[string]interface{}, len(original))

	for i, v := range original {
		mapped[i] = v.Data
	}

	response = honeycombQueryData{
		Data: mapped,
	}

	return response, nil
}
