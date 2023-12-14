#!/bin/sh

rm api
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api src/*.go

sam local start-api