package main

import "github.com/aws/aws-lambda-go/events"

func postAnswer(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{Body: "Not Implemented", StatusCode: 501}, nil
}
