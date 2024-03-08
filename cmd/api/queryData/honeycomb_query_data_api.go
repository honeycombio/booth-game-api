package queryData

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"observaquiz_lambda/pkg/instrumentation"
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
	queryDataApiKey string
	apiBaseUrl      string
	Tracer          oteltrace.Tracer
	client          http.Client
}

func NewHoneycombAPI(apikey string) honeycombQueryDataAPI {
	return honeycombQueryDataAPI{
		queryDataApiKey: apikey,
		apiBaseUrl:      "https://api.honeycomb.io/1", // our DevRel team is in US region.
		Tracer:          instrumentation.TracerProvider.Tracer("querydata.honeycomb"),
		client:          http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)},
	}
}

type DefineQueryResponse struct {
	QueryId string `json:"query"`
}

func (api honeycombQueryDataAPI) send(currentContext context.Context, method string, relativeUrl string, payload []byte) (response []byte, err error) {
	span := oteltrace.SpanFromContext(currentContext)
	span.SetAttributes(attribute.String("app.request.url", api.apiBaseUrl+relativeUrl))

	url := api.apiBaseUrl + relativeUrl
	req, _ := http.NewRequestWithContext(currentContext, method, url, bytes.NewBuffer(payload))

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-honeycomb-team", api.queryDataApiKey)

	resp, err := api.client.Do(req)
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

func (honeycombapi honeycombQueryDataAPI) CreateQuery(currentContext context.Context, queryDefinition HoneycombQuery, datasetSlug string) (response DefineQueryResponse, err error) {
	currentContext, span := honeycombapi.Tracer.Start(currentContext, "Create Honeycomb Query")
	defer span.End()

	queryDefinitionString, err := json.Marshal(queryDefinition)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error marshalling query definition")
		return response, err
	}
	span.SetAttributes(attribute.String("observaquiz.qd.query.body", string(queryDefinitionString)),
		attribute.String("observaquiz.qd.query.dataset", datasetSlug))

	bodyBytes, err := honeycombapi.send(currentContext, "POST", "/queries/"+datasetSlug, queryDefinitionString)
	if err != nil {
		span.SetAttributes(attribute.String("observaquiz.qd.error_response_body", string(bodyBytes)))
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
	span.SetAttributes(attribute.String("observaquiz.qd.query.id", string(startQueryInputString)),
		attribute.String("observaquiz.qd.query.dataset", datasetSlug))

	bodyBytes, err := api.send(currentContext, "POST", "/query_results/"+datasetSlug, startQueryInputString)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("observaquiz.qd.error_response_body", string(bodyBytes)))
		span.SetStatus(codes.Error, err.Error())
		return response, err
	}
	startQueryResponse := startQueryResponseBody{}
	err = json.Unmarshal(bodyBytes, &startQueryResponse)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error unmarshalling response")
		return response, err
	}

	span.SetAttributes(
		attribute.String("observaquiz.qd.result.link", startQueryResponse.Links.QueryURL),
		attribute.String("observaquiz.qd.result.graph_link", startQueryResponse.Links.GraphImageURL))

	resultId := startQueryResponse.ResultId
	span.SetAttributes(attribute.String("observaquiz.qd.result.id", resultId))

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
	currentContext, span := api.Tracer.Start(currentContext, "Poll for Honeycomb Query Result")
	defer span.End()
	span.SetAttributes(attribute.String("app.request.result_id", resultId))

	// TODO: the thing
	// 1. Poll the result URL until it's done
	queryResult := getQueryResultResponse{}
	pollCount := 0
	for {
		pollCount++
		bodyBytes, err := api.send(currentContext, "GET", fmt.Sprintf("/query_results/%s/%s", datasetSlug, resultId), []byte{})
		if err != nil {
			span.SetAttributes(attribute.String("observaquiz.qd.error_response_body", string(bodyBytes)))
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

	span.SetAttributes(attribute.Int("observaquiz.qd.request.poll_count", pollCount),
		attribute.String("observaquiz.qd.result.link", queryResult.Links.QueryURL),
		attribute.String("observaquiz.qd.result.graph_link", queryResult.Links.GraphImageURL))

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
