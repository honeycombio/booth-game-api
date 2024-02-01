package main

import (
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Function to set attributes on a span from a JSON object
func setSpanAttributesFromJSON(span trace.Span, keyPrefix string, jsonObj interface{}) {
	switch jsonObj := jsonObj.(type) {
	case map[string]interface{}:
		for key, val := range jsonObj {
			if val != nil {
				fullKey := key
				if keyPrefix != "" {
					fullKey = keyPrefix + "." + key
				}
				setSpanAttributesFromJSON(span, fullKey, val)
			}
		}
	case string:
		// Attempt to parse the string as a date
		if t, err := time.Parse(time.RFC3339Nano, jsonObj); err == nil {
			// It's a date, set it as a string and as a UNIX timestamp
			span.SetAttributes(attribute.String(keyPrefix, jsonObj))
			span.SetAttributes(attribute.Int64(keyPrefix+"_unix", t.Unix()))
		} else {
			// It's not a date, just set it as a string
			span.SetAttributes(attribute.String(keyPrefix, jsonObj))
		}
	case float64:
		// JSON numbers are float64 by default
		span.SetAttributes(attribute.Float64(keyPrefix, jsonObj))
	case int:
		// Handle int specifically if needed
		span.SetAttributes(attribute.Int(keyPrefix, jsonObj))
	case bool:
		// Handle bool specifically if needed
		span.SetAttributes(attribute.Bool(keyPrefix, jsonObj))
	default:
		fmt.Println("Unhandled type:", keyPrefix, jsonObj)
	}
}

func main() {
	// Example JSON
	jsonData := `{
		"stuff": "things",
		"more": {
			"dingleberries": 42,
			"when": "2024-01-30T14:29:16.261245+00:00"
		},
		"topic": null
	}`

	// Unmarshal JSON into an interface{}
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonData), &jsonObj); err != nil {
		panic(err)
	}

	// Assume `span` is an existing OpenTelemetry span
	var span trace.Span // Placeholder for an actual span

	// Set attributes on the span from the JSON object
	setSpanAttributesFromJSON(span, "", jsonObj)

	// Your span processing logic here...
}
