package main

import (
	"context"
	"fmt"
	"observaquiz_lambda/pkg/instrumentation"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/jessevdk/go-flags"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/otel/trace"
)

type sdkConfigKey string

var SDK_CONFIG_KEY sdkConfigKey = "sdkConfig"

var settings struct {
	ResultsTableName string `env:"results_table_name" short:"r"`
	UseLocalStack    bool   `env:"use_local_stack" short:"l"`
}

var tracer trace.Tracer

func main() {
	fmt.Print("Starting result calculator\n")
	flags.Parse(&settings)
	// print all the environment variables to the console
	currentContext := context.Background()

	tracerProvider := instrumentation.CreateTracerProvider(currentContext, "observaquiz-result-calculator")
	tracer = tracerProvider.Tracer("observaquiz-ddb/stream")

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

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(handler,
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
