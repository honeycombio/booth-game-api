#!/bin/sh

export AWS_PROFILE=localstack

aws dynamodb create-table \
    --table-name QuizRuns \
    --attribute-definitions \
        AttributeName=quiz-name,AttributeType=S \
        AttributeName=quiz-run-id,AttributeType=S \
    --key-schema \
        AttributeName=quiz-run-id,KeyType=HASH \
        AttributeName=quiz-name,KeyType=RANGE \
    --provisioned-throughput \
        ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --table-class STANDARD