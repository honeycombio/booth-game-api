#!/bin/sh

rm ../api
rm ../deep_checks_callback
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ../api ../cmd/api/*.go
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ../deep_checks_callback ../cmd/deep_checks_callback/*.go
zip -j ../api.zip ../api
zip -j ../deep_checks_callback.zip ../deep_checks_callback

pulumi up
