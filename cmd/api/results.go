package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// add the results to the dynamodb table

type ResultTable struct {
	TableName      string
	DynamoDbClient *dynamodb.Client
}

type Result struct {
	EventName  string `dynamodbav:"event_name"`
	QuizRunId  string `dynamodbav:"quiz_run_id"`
	QuestionId string `dynamodbav:"question_id"`
	Answer     string `dynamodbav:"answer"`
	TraceId    string `dynamodbav:"trace_id"`
	Score      int    `dynamodbav:"score"`
}

func (table ResultTable) AddResult(result Result) error {
	item, err := attributevalue.MarshalMap(result)
	if err != nil {
		panic(err)
	}
	_, err = table.DynamoDbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(table.TableName), Item: item,
	})
	if err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}
	return err
}
