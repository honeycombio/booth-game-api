package queryData

import (
	"context"
	"crypto/sha256"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type HoneycombQuery struct {
	TimeRange    int           `json:"time_range"`
	Granularity  int           `json:"granularity"`
	Breakdowns   []string      `json:"breakdowns"`
	Calculations []Calculation `json:"calculations"`
	Filters      []Filter      `json:"filters"`
	Orders       []Order       `json:"orders"`
	Havings      []interface{} `json:"havings"`
	Limit        int           `json:"limit"`
}

type Filter struct {
	Column string `json:"column"`
	Op     string `json:"op"`
	Value  string `json:"value"`
}

type Calculation struct {
	Op     string  `json:"op"`
	Column *string `json:"column,omitempty"`
}

type Order struct {
	Op     string `json:"op"`
	Order  string `json:"order"`
	Column string `json:"column"`
}

type QueryDataRequest struct {
	QueryDefinition HoneycombQuery `json:"query"`
	DatasetSlug     string         `json:"dataset_slug"`
	AttendeeApiKey  string         `json:"attendee_api_key"`
}

type QueryDataResponse struct {
	QueryId   string                   `json:"query_id"`
	ResultId  string                   `json:"result_id"`
	Error     string                   `json:"error"`
	QueryData []map[string]interface{} `json:"query_data"`
}

func errorQueryDataResponse(err error) (QueryDataResponse, error) {
	return QueryDataResponse{Error: err.Error()}, err
}

func CreateAndRunHoneycombQuery(currentContext context.Context, queryDataApiKey string, request QueryDataRequest) (response QueryDataResponse, err error) {
	// 0. Construct the query
	queryDefinition := request.QueryDefinition

	hasher := sha256.New()
	hasher.Write([]byte(request.AttendeeApiKey))
	expectedValueOfApiKey := fmt.Sprintf("%x", hasher.Sum(nil))
	oteltrace.SpanFromContext(currentContext).SetAttributes(attribute.String("observaquiz.hashed_honeycomb_api_key", expectedValueOfApiKey))

	newFilter := Filter{
		Column: "app.honeycomb_api_key",
		Op:     "=",
		Value:  expectedValueOfApiKey,
	}
	// Make sure they only ever see their own data.
	queryDefinition.Filters = append(queryDefinition.Filters, newFilter)

	hnyApi := NewHoneycombAPI(queryDataApiKey)
	// 1. Create query
	createQueryResponse, err := hnyApi.CreateQuery(currentContext, queryDefinition, request.DatasetSlug)
	if err != nil {
		return errorQueryDataResponse(err)
	}

	// 2. Run query
	startQueryResponse, err := hnyApi.StartQuery(currentContext, createQueryResponse.QueryId, request.DatasetSlug)
	if err != nil {
		return errorQueryDataResponse(err)
	}

	// 3. Get results
	queryData, err := hnyApi.GiveMeTheData(currentContext, startQueryResponse.ResultId, request.DatasetSlug)
	if err != nil {
		return errorQueryDataResponse(err)
	}

	return QueryDataResponse{
		QueryId:   createQueryResponse.QueryId,
		ResultId:  startQueryResponse.ResultId,
		QueryData: queryData.Data,
	}, nil

}
