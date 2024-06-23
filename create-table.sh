#!/bin/sh

aws --endpoint-url=http://localhost:4566 \
    dynamodb create-table \
    --table-name results-table \
    --attribute-definitions \
        AttributeName=event_name,AttributeType=S AttributeName=quiz_run_id,AttributeType=S \
    --key-schema \
        AttributeName=event_name,KeyType=HASH AttributeName=quiz_run_id,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5