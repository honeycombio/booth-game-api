package main

import (
	"context"
	"encoding/json"
	"fmt"
	"observaquiz_lambda/pkg/instrumentation"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"observaquiz_lambda/cmd/api/deepchecks"
)

var postOpinionEndpoint = apiEndpoint{
	"POST",
	"/api/opinion",
	regexp.MustCompile("^/api/opinion$"),
	postOpinion,
	true,
}

type PostOpinionBody struct {
	EvaluationId string `json:"evaluation_id"`
	Opinion      string `json:"opinion"`
}

type PostOpinionResponse struct {
	Something string `json:"something"`
}

func postOpinion(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {

	currentContext, postOpinionSpan := tracer.Start(currentContext, "post opinion")
	defer postOpinionSpan.End()
	defer func() {
		if r := recover(); r != nil {
			response = instrumentation.RespondToPanic(postOpinionSpan, r)
		}
	}()

	/* Parse what they sent */
	postOpinionSpan.SetAttributes(attribute.String("request.body", request.Body))
	answer := PostOpinionBody{}
	err = json.Unmarshal([]byte(request.Body), &answer)
	if err != nil {
		newErr := fmt.Errorf("error unmarshalling answer: %w\n request body: %s", err, request.Body)
		postOpinionSpan.RecordError(newErr)
		return events.APIGatewayV2HTTPResponse{Body: "Bad request. Expected format: { 'evaluation_id': 'trace-span', opinion: 'whoa' }", StatusCode: 400}, nil
	}

	// /* what question are they referring to? */
	// eventName := getEventName(request)
	// postOpinionSpan.SetAttributes(attribute.String("app.post_answer.event_name", eventName))
	// path := request.RequestContext.HTTP.Path
	// pathSplit := strings.Split(path, "/")
	// questionId := pathSplit[3]
	// postOpinionSpan.SetAttributes(attribute.String("app.post_answer.question_id", questionId))

	// /* find that question in our question definitions */
	// var question string
	// var openaiMessages []openai.ChatCompletionMessage
	// var promptSpec AnswerResponsePrompt
	// var fullPrompt string
	// eventQuestions := eventQuestions[eventName]

	// for _, v := range eventQuestions {
	// 	if v.Id.String() == questionId {
	// 		promptSpec = v.AnswerResponsePrompt
	// 		question = v.Question
	// 		break
	// 	}
	// }
	// if question == "" {
	// 	postOpinionSpan.SetAttributes(attribute.String("error.message", "Couldn't find question"))
	// 	postOpinionSpan.SetStatus(codes.Error, "Couldn't find question")
	// 	return instrumentation.ErrorResponse("Couldn't find question with that ID", 404), nil
	// }
	// postOpinionSpan.SetAttributes(attribute.String("app.post_answer.question", question))

	deepchecksAPI := deepchecks.DeepChecksAPI{ApiKey: settings.DeepchecksApiKey}
	interactionReported := deepchecksAPI.ReportOpinion(currentContext, deepchecks.LLMInteractionOpinionReport{
		EvaluationId: answer.EvaluationId,
		Opinion:      deepchecks.Unknown,
		AppVersionId: "1", // TODO: get from question definition. Except we have to use the number here :( :(
	})

	postOpinionSpan.SetAttributes(attribute.Bool("app.reported", interactionReported.Reported), attribute.Bool("app.success", interactionReported.Success))

	/* tell the UI what we got */
	result := PostOpinionResponse{Something: "does this work"}
	jsonData, err := json.Marshal(result)
	if err != nil {
		postOpinionSpan.RecordError(err, trace.WithAttributes(attribute.String("error.message", "Failure marshalling JSON")))
		return instrumentation.ErrorResponse("wtaf", 500), nil
	}

	return events.APIGatewayV2HTTPResponse{Body: string(jsonData), StatusCode: 200}, nil
}
