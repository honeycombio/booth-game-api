package queryData

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	Op string `json:"op"`
}

type Order struct {
	Op    string `json:"op"`
	Order string `json:"order"`
}

type QueryDataRequest struct {
	QueryDefinition HoneycombQuery `json:"query"`
	DatasetSlug     string         `json:"dataset_slug"`
	AttendeeApiKey  string         `json:"attendee_api_key"`
}

type QueryDataResponse struct {
	QueryId    string `json:"query"`
	ResponseId string `json:"response"`
	Error      string `json:"error"`
	QueryData  string `json:"query_data"`
}

func RunHoneycombQuery(currentContext context.Context, request QueryDataRequest) (response QueryDataResponse, err error) {

	// 0. Construct the query
	queryDefinition := request.QueryDefinition
	newFilter := Filter{
		Column: "app.honeycomb_api_key",
		Op:     "=",
		Value:  request.AttendeeApiKey,
	}
	// Append the new filter to the existing filters
	queryDefinition.Filters = append(queryDefinition.Filters, newFilter)

	hnyApi := ProductionQueryDataAPI()
	// 1. Create query
	createQueryResponse, err := hnyApi.CreateQuery(currentContext, queryDefinition, request.DatasetSlug)
	
	return QueryDataResponse{
		QueryId:    createQueryResponse.QueryId,
	}, nil

}
