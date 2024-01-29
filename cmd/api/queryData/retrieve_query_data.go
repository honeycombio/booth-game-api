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
}

type QueryDataResponse struct {
	QueryId    string `json:"query"`
	ResponseId string `json:"response"`
	Error      string `json:"error"`
	QueryData  string `json:"query_data"`
}

func RunHoneycombQuery(currentContext context.Context, request QueryDataRequest) (response QueryDataResponse, err error) {
	datasetSlug := "observaquiz-bff"
	honeycombAPIKey := os.Getenv("HONEYCOMB_API_KEY") // Assuming the API key is set in environment variable

	// 0. Construct the query
	queryDefinition := request.QueryDefinition
	newFilter := Filter{
		Column: "app.honeycomb_api_key",
		Op:     "=",
		Value:  honeycombAPIKey, // No. The one they passed in, on the header. Need to get that from Main
	}
	// Append the new filter to the existing filters
	queryDefinition.Filters = append(queryDefinition.Filters, newFilter)

	// 1. Create query
	queryCreateURL := fmt.Sprintf("https://api.honeycomb.io/1/queries/%s", datasetSlug)
	queryID, err := postRequest(queryCreateURL, honeycombAPIKey, queryDefinition)
	if err != nil {
		return QueryDataResponse{
			QueryId: queryID,
			Error:   err.Error(),
		}, nil
	}

	// 2. (Optional) Fetch query definition - omitted as it's noted as boring

	// 3. Execute the query
	queryResultsURL := fmt.Sprintf("https://api.honeycomb.io/1/query_results/%s", datasetSlug)
	executionPayload := map[string]interface{}{
		"query_id":       queryID,
		"disable_series": true,
		"limit":          100,
	}
	executionPayloadBytes, err := json.Marshal(executionPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal execution payload: %w", err)
	}
	resultID, err := postRequest(queryResultsURL, honeycombAPIKey, executionPayloadBytes)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	// 4. Fetch query results
	resultsURL := fmt.Sprintf("https://api.honeycomb.io/1/query_results/%s/%s", datasetSlug, resultID)
	result, err := getRequest(resultsURL, honeycombAPIKey)
	if err != nil {
		return fmt.Errorf("failed to get query results: %w", err)
	}

	// Assuming you want to save the result to a file
	if err := ioutil.WriteFile("actual_result.json", result, 0644); err != nil {
		return fmt.Errorf("failed to write actual_result.json: %w", err)
	}

	return nil
}

func postRequest(url, apiKey string, payload []byte) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Honeycomb-Team", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	} else {
		return "", fmt.Errorf("no id found in response")
	}
}

func getRequest(url, apiKey string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Honeycomb-Team", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return body, nil
}
