#!/bin/sh

if [ ! -d .git ] ; then
    echo "Please run this script from the root of the repository"
    exit 1
fi

if [ ! -d deploy ] ; then
    mkdir deploy
fi

rm ./deploy/*
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./deploy/api ./cmd/api/*.go
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./deploy/deepchecks_callback ./cmd/deepchecks_callback/*.go
zip -j ./deploy/api.zip ./deploy/api
zip -j ./deploy/deepchecks_callback.zip ./deploy/deepchecks_callback

cd infra
pulumi up --yes
