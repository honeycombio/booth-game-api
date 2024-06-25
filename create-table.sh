#!/bin/sh

export AWS_REGION=us-east-1

aws --endpoint-url=http://localhost:4566 \
    dynamodb create-table \
    --table-name results-table \
    --attribute-definitions \
        AttributeName=event_name,AttributeType=S AttributeName=quiz_run_id,AttributeType=S, AttributeName=type,AttributeType=S, AttributeName=total_score,AttributeType=N \
    --key-schema \
        AttributeName=quiz_run_id,KeyType=HASH AttributeName=type,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5