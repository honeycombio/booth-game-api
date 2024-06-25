package main

import (
	"context"
	"fmt"
	"observaquiz_lambda/cmd/api/results"
	"observaquiz_lambda/pkg/instrumentation"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	default_event           = "devopsdays_whenever"
	ATTENDEE_API_KEY_HEADER = "x-honeycomb-api-key"
	EXECUTION_ID_HEADER     = "x-observaquiz-execution-id"
	ServiceName             = "observaquiz-bff"
)

// key for sdkconfig in context
type sdkConfigKey string

var SDK_CONFIG_KEY sdkConfigKey = "sdkConfig"

var tracer oteltrace.Tracer

const LocalTraceLink = true // feature flag, enable locally and turn off in prod ideally

var resultsTable results.ResultTable

func RouterWithSpan(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	currentContext, cleanup := context.WithTimeout(currentContext, 30*time.Second)
	defer cleanup()
	lambdaSpan := oteltrace.SpanFromContext(currentContext)
	defer func() {
		if r := recover(); r != nil {
			response = instrumentation.RespondToPanic(lambdaSpan, r)
		}
	}()

	currentContext, _ = setAttributesOnSpanAndBaggageFromHeaders(currentContext, request)
	instrumentation.AddHttpRequestAttributesToSpan(lambdaSpan, request)

	response, err = getResponseFromAPIRouter(currentContext, request)

	instrumentation.AddHttpResponseAttributesToSpan(lambdaSpan, response)
	addSpanAttributesToResponse(currentContext, &response)

	return response, err

}

func getResponseFromAPIRouter(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	lambdaSpan := oteltrace.SpanFromContext(currentContext)

	endpoint, endpointFound := api.findEndpoint(request.RequestContext.HTTP.Method, request.RequestContext.HTTP.Path)

	if !endpointFound {
		methodPath := request.RequestContext.HTTP.Method + " " + request.RequestContext.HTTP.Path
		response = events.APIGatewayV2HTTPResponse{Body: fmt.Sprintf("Unhandled Route %v", methodPath), StatusCode: 404}
	} else {
		lambdaSpan.SetName(fmt.Sprintf("%s %s", endpoint.method, endpoint.pathTemplate))
		response, err = getResponseFromHandler(currentContext, endpoint, request)
		if err != nil {
			lambdaSpan.RecordError(err)
		}
	}

	return response, err

}

func addSpanAttributesToResponse(currentContext context.Context, response *events.APIGatewayV2HTTPResponse) {
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}

	carrier := propagation.MapCarrier{}

	otel.GetTextMapPropagator().Inject(currentContext, carrier)
	response.Headers["x-tracechild"] = carrier["traceparent"]
}

func setAttributesOnSpanAndBaggageFromHeaders(currentContext context.Context, request events.APIGatewayV2HTTPRequest) (context.Context, error) {
	var currentSpan = oteltrace.SpanFromContext(currentContext)
	headerInfo := getHeaderInfo(request)

	currentSpan.SetAttributes(attribute.String(instrumentation.ATTENDEE_API_KEY_ATTRIBUTE_KEY, headerInfo.AttendeeApiKey))
	currentSpan.SetAttributes(attribute.String(instrumentation.EXECUTION_ID_ATTRIBUTE_KEY, headerInfo.ExecutionId))
	return instrumentation.SetApiKeyInBaggage(currentContext, headerInfo.AttendeeApiKey, headerInfo.ExecutionId)
}

func getEventName(request events.APIGatewayV2HTTPRequest) string {
	eventName := request.Headers["event-name"]
	if eventName == "" {
		eventName = default_event
	}

	return eventName
}

type HeaderInfo struct {
	AttendeeApiKey string
	ExecutionId    string
}

func getHeaderInfo(request events.APIGatewayV2HTTPRequest) HeaderInfo {
	var executionId string
	var attendeeApiKey string
	for k, v := range request.Headers {
		lowerKey := strings.ToLower(k)
		if lowerKey == ATTENDEE_API_KEY_HEADER {
			attendeeApiKey = v
		} else if lowerKey == EXECUTION_ID_HEADER {
			executionId = v
		}
	}
	return HeaderInfo{AttendeeApiKey: attendeeApiKey, ExecutionId: executionId}
}

var settings struct {
	OpenAIKey        string `env:"openai_key" short:"o"`
	QueryDataApiKey  string `env:"query_data_api_key" short:"q"`
	DeepchecksApiKey string `env:"deepchecks_api_key" short:"d"`
	ResultsTableName string `env:"results_table_name" short:"r"`
	UseLocalStack    bool   `env:"use_local_stack" short:"l" default:"false"`
}

func main() {
	flags.Parse(&settings)
	currentContext := context.Background()

	tracerProvider := instrumentation.CreateTracerProvider(currentContext, ServiceName)
	tracer = tracerProvider.Tracer("observaquiz-bff/main")

	var err error
	var sdkConfig aws.Config
	if settings.UseLocalStack {
		sdkConfig = getLocalStackConfig()
	} else {
		sdkConfig, err = config.LoadDefaultConfig(currentContext)
	}
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	currentContext = context.WithValue(currentContext, SDK_CONFIG_KEY, sdkConfig)

	resultsTable = results.NewResultTable(settings.ResultsTableName, getDynamoDbClient(currentContext))

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(RouterWithSpan,
			otellambda.WithFlusher(tracerProvider),
			otellambda.WithTracerProvider(tracerProvider)),
		lambda.WithContext(currentContext),
	)
}

func getLocalStackConfig() aws.Config {
	awsCfg, _ := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)

	return awsCfg
}

func getDynamoDbClient(currentContext context.Context) *dynamodb.Client {
	sdkConfig := currentContext.Value(SDK_CONFIG_KEY).(aws.Config)
	return dynamodb.NewFromConfig(sdkConfig, func(o *dynamodb.Options) {
		o.Region = "us-east-1"
		o.BaseEndpoint = aws.String("http://localstack:4566")
	})
}
