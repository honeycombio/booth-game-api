package results

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.opentelemetry.io/otel/trace"
)

// add the results to the dynamodb table

type ResultTable struct {
	TableName      string
	DynamoDbClient *dynamodb.Client
}

type ResultBase struct {
	ExecutionId string `dynamodbav:"quiz_run_id" json:"quiz_run_id"`
	Type        string `dynamodbav:"type"`
}

type Result struct {
	ResultBase
	EventName  string `dynamodbav:"event_name" json:"event_name"`
	QuestionId string `dynamodbav:"question_id" json:"question_id"`
	Answer     string `dynamodbav:"answer" json:"answer"`
	TraceId    string `dynamodbav:"trace_id" json:"trace_id"`
	Score      int    `dynamodbav:"score" json:"score"`
}

type ResultSummary struct {
	ResultBase
	EventName  string `dynamodbav:"event_name" json:"event_name"`
	TotalScore int    `dynamodbav:"total_score" json:"total_score"`
}

func NewResultTable(tableName string, dynamoDbClient *dynamodb.Client) ResultTable {
	return ResultTable{TableName: tableName, DynamoDbClient: dynamoDbClient}
}

func (table ResultTable) AddResult(currentContext context.Context, executionId, eventName, questionId, answer string, score int) error {
	var currentSpan = trace.SpanFromContext(currentContext)
	questionResult := Result{
		ResultBase: ResultBase{
			ExecutionId: executionId,
			Type:        "result",
		},
		EventName:  eventName,
		QuestionId: questionId,
		Answer:     answer,
		TraceId:    currentSpan.SpanContext().TraceID().String(),
		Score:      score,
	}

	item, err := attributevalue.MarshalMap(questionResult)
	if err != nil {
		panic(err)
	}
	_, err = table.DynamoDbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(table.TableName), Item: item,
	})
	if err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}

	resultSummary, err := table.GetResultSummaryForExecution(currentContext, executionId)

	if err != nil {
		log.Printf("Couldn't get result summary for %v. Here's why: %v\n", executionId, err)
	} else {
		resultSummary.TotalScore += score
		err = table.AddResultSummaryForExecution(currentContext, eventName, executionId, resultSummary.TotalScore)
		if err != nil {
			log.Printf("Couldn't add result summary for %v. Here's why: %v\n", executionId, err)
		}
	}
	return err
}

func (table ResultTable) GetResultSummaryForExecution(currentContext context.Context, executionId string) (ResultSummary, error) {
	//awslocal dynamodb execute-statement --statement "SELECT * FROM \"results-table\" WHERE total_score > 0"
	var ResultSummary ResultSummary
	lookupKey := map[string]types.AttributeValue{
		"quiz_run_id": &types.AttributeValueMemberS{Value: executionId},
		"type":        &types.AttributeValueMemberS{Value: "summary"},
	}
	response, err := table.DynamoDbClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		Key: lookupKey, TableName: aws.String(table.TableName),
	})
	if err != nil {
		log.Printf("Couldn't get info about %v. Here's why: %v\n", executionId, err)
	} else {
		err = attributevalue.UnmarshalMap(response.Item, &ResultSummary)
		if err != nil {
			log.Printf("Couldn't unmarshal response. Here's why: %v\n", err)
		}
	}
	return ResultSummary, err
}

func (table ResultTable) AddResultSummaryForExecution(currentContext context.Context, eventName string, executionId string, totalScore int) error {

	resultSummary := ResultSummary{
		ResultBase: ResultBase{
			ExecutionId: executionId,
			Type:        "summary",
		},
		EventName:  eventName,
		TotalScore: totalScore,
	}
	item, err := attributevalue.MarshalMap(resultSummary)
	if err != nil {
		panic(err)
	}
	_, err = table.DynamoDbClient.PutItem(currentContext, &dynamodb.PutItemInput{
		TableName: aws.String(table.TableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}
	return err
}

func (table ResultTable) GetExecutionResultFromDynamo(currentContext context.Context, executionId string) ([]Result, error) {

	var results []Result
	var err error
	var response *dynamodb.QueryOutput
	expr, err := expression.NewBuilder().WithKeyCondition(
		expression.Key("quiz_run_id").Equal(expression.Value(executionId)).
			And(expression.Key("type").Equal(expression.Value("result")))).
		Build()
	if err != nil {
		log.Printf("Couldn't build expression for query. Here's why: %v\n", err)
	} else {
		queryPaginator := dynamodb.NewQueryPaginator(table.DynamoDbClient, &dynamodb.QueryInput{
			TableName:                 aws.String(table.TableName),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		})
		for queryPaginator.HasMorePages() {
			response, err = queryPaginator.NextPage(currentContext)
			if err != nil {
				log.Printf("Couldn't query for movies released in %v. Here's why: %v\n", executionId, err)
				break
			} else {
				var resultPage []Result
				err = attributevalue.UnmarshalListOfMaps(response.Items, &resultPage)
				if err != nil {
					log.Printf("Couldn't unmarshal query response. Here's why: %v\n", err)
					break
				} else {
					results = append(results, resultPage...)
				}
			}
		}
	}

	return results, err
}

func (table ResultTable) GetAllResultsForEvent(currentContext context.Context, eventName string) ([]ResultSummary, error) {

	var results []ResultSummary
	var err error
	var response *dynamodb.QueryOutput
	fmt.Printf("Getting results for %v\n", eventName)
	expr, err := expression.NewBuilder().WithKeyCondition(
		expression.Key("event_name").Equal(expression.Value(eventName))).
		Build()
	if err != nil {
		log.Printf("Couldn't build expression for query. Here's why: %v\n", err)
	} else {
		queryPaginator := dynamodb.NewQueryPaginator(table.DynamoDbClient, &dynamodb.QueryInput{
			TableName:                 aws.String(table.TableName),
			IndexName:                 aws.String("total_score-index"),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		})
		for queryPaginator.HasMorePages() {
			response, err = queryPaginator.NextPage(currentContext)
			if err != nil {
				log.Printf("Couldn't query for movies released in %v. Here's why: %v\n", eventName, err)
				break
			} else {
				var resultPage []ResultSummary
				err = attributevalue.UnmarshalListOfMaps(response.Items, &resultPage)
				if err != nil {
					log.Printf("Couldn't unmarshal query response. Here's why: %v\n", err)
					break
				} else {
					results = append(results, resultPage...)
				}
			}
		}
	}

	return results, err
}
