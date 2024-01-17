# Api for the booth game

This is the backend API for the booth game, it's primarily a go app handling those requests.

## Local debugging

`export HONEYCOMB_API_KEY=your api key`

This uses AWS SAM (Serverless Application Model).

We define the API in `template.yaml`, with it's name etc. this looks for it on the filesystem as a relative path.

You'll need an OpenAI key, put this in a file named `environment.json` in the root directory, with the format:

```json
{
    "API": {
        "openai_key": "<key>",
        "OTEL_EXPORTER_OTLP_ENDPOINT": "https://collector:4318",
        "OTEL_EXPORTER_OTLP_INSECURE": true
    }
}
```

## Run locally

`./run.sh`

and then to test, open `http_tests/questions.http` and click the tiny "Send Request" above the examples.

## Build and Deploy.

Build and package

```sh
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api src/*.go && zip api.zip api
```

Make sure the packages are restored

```sh
cd infra
npm i
```

Login to pulumi via the CLI

```sh
pulumi login
```

Follow the prompts to login

Deploy

```sh
pulumi up
```