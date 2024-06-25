package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

func handler(ctx context.Context, e events.DynamoDBEvent) {
	fmt.Printf("Processing record data")
	for _, record := range e.Records {
		fmt.Printf("Processing request data for event ID %s, type %s.\n", record.EventID, record.EventName)

		// Print new values for attributes of type String
		for name, value := range record.Change.NewImage {
			if value.DataType() == events.DataTypeString {
				fmt.Printf("Attribute name: %s, value: %s\n", name, value.String())
			}
		}
	}
}
