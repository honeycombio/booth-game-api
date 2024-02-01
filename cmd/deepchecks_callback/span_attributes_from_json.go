package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Function to set attributes on a span from a JSON object
func setSpanAttributesFromJSON(span trace.Span, keyPrefix string, jsonObj interface{}) error {
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
		return errors.New("Unhandled type: " + keyPrefix + " " + fmt.Sprintf("%v", jsonObj))
	}
	return nil
}

func SetSpanAttributesFromJSONString(span trace.Span, keyPrefix string, jsonData string) error {

	// Unmarshal JSON into an interface{}
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonData), &jsonObj); err != nil {
		panic(err)
	}

	// Set attributes on the span from the JSON object
	return setSpanAttributesFromJSON(span, keyPrefix, jsonObj)
}
