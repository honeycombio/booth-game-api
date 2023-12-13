package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/valyala/fastjson"
)

func HandleRequest(context context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	ApiResponse := events.APIGatewayV2HTTPResponse{}
	// Switch for identifying the HTTP request
	switch request.RequestContext.HTTP.Method {
	case "GET":
		// Obtain the QueryStringParameter
		name := request.QueryStringParameters["name"]
		if name != "" {
			ApiResponse = events.APIGatewayV2HTTPResponse{Body: "Hey " + name + " welcome! ", StatusCode: 200}
		} else {
			ApiResponse = events.APIGatewayV2HTTPResponse{Body: "Error: Query Parameter name missing", StatusCode: 500}
		}

	case "POST":
		//validates json and returns error if not working
		err := fastjson.Validate(request.Body)

		if err != nil {
			body := "Error: Invalid JSON payload ||| " + fmt.Sprint(err) + " Body Obtained" + "||||" + request.Body
			ApiResponse = events.APIGatewayV2HTTPResponse{Body: body, StatusCode: 500}
		} else {
			ApiResponse = events.APIGatewayV2HTTPResponse{Body: request.Body, StatusCode: 200}
		}
	default:
		ApiResponse = events.APIGatewayV2HTTPResponse{Body: "Error: Invalid HTTP Method", StatusCode: 500}
	}
	// Response
	return ApiResponse, nil
}

func main() {
	lambda.Start(HandleRequest)
}
