#!/bin/sh

if [ ! -d .git ] ; then
    echo "Please run this script from the root of the repository"
    echo "I got tired of cding to infra"
    exit 1
fi

rm ./api
rm ./deepchecks_callback
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./api ./cmd/api/*.go
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./deepchecks_callback ./cmd/deepchecks_callback/*.go
zip -j ./api.zip ./api
zip -j ./deepchecks_callback.zip ./deepchecks_callback

cd infra
pulumi up --yes
