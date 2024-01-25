#!/bin/sh

if [ ! -d .git ] ; then
    echo "Please run this script from the root of the repository"
    echo "I got tired of cding to infra"
    exit 1
fi

rm ./api
rm ./deep_checks_callback
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./api ./cmd/api/*.go
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./deep_checks_callback ./cmd/deep_checks_callback/*.go
zip -j ./api.zip ./api
zip -j ./deep_checks_callback.zip ./deep_checks_callback

cd infra
pulumi up --yes
