#!/bin/sh
if [ -z "${HONEYCOMB_API_KEY}" ]; then
    echo "HONEYCOMB_API_KEY Environment variable isn't set"
    exit 1
fi

if [ ! -e "environment.json" ]; then
    echo "environment.json file doesn't exist"
    exit 1
fi

docker-compose -f local-collector/docker-compose.yml up -d

rm api
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api src/*.go

sam local start-api --env-vars environment.json --docker-network local-collector_collector_net

