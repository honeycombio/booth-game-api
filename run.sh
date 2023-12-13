#!/bin/sh

rm HandleRequest
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o HandleRequest src/main.go

sam local start-api