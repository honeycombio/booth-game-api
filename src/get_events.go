package main

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

//go:embed questions/*
var eventDirectories embed.FS

var eventsWithQuestions, _ = eventDirectories.ReadDir("questions")

var eventQuestions = parseEvents()

func getEvents(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	eventNames := []string{}
	for _, v := range eventsWithQuestions {
		if v.IsDir() {
			eventNames = append(eventNames, v.Name())
		}
	}

	eventsJson, _ := json.Marshal(eventNames)

	return events.APIGatewayV2HTTPResponse{
		Body:       string(eventsJson),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200}, nil
}

func parseEvents() map[string][]Question {
	parsedEventsWithQuestions := map[string][]Question{}
	for _, v := range eventsWithQuestions {
		if v.IsDir() {
			var questionList []Question

			questionsFile, err := eventDirectories.ReadFile(fmt.Sprintf("questions/%v/questions.json", v.Name()))
			if err != nil {
				fmt.Printf("Error unmarshalling questions: %v\n", err)
				continue
			}

			err = json.Unmarshal(questionsFile, &questionList)
			if err != nil {
				fmt.Printf("Error unmarshalling questions: %v\n", err)
				continue
			}
			parsedEventsWithQuestions[v.Name()] = questionList
		}
	}

	return parsedEventsWithQuestions
}
