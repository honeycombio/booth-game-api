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
        "OTEL_EXPORTER_OTLP_INSECURE": true,
        "query_data_api_key": "<key>",
        "deepchecks_api_key": "goes here"
    }
}
```

## Run locally

`./run.sh`

## Testing

The API requires a Honeycomb API key for the attendee. To test the you'll need to add that header as `x-honeycomb-api-key`.

### Testing with Rest client in VSCode

There are built in tests in the repository using rest client for VSCode. These tests live in the `http_tests` folder.

To use this, you'll need to setup a `.env` file in that directory with the Attendee's API key. There is a `.env.sample` file to copy to get the format right.

To test, open one of the `.http` files and click the tiny "Send Request" above the examples.

## Iterating

While the sam thinger is running the lambda, change the .go files and then do the build step:

```sh
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api src/*.go
```

Now try the test again.

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

Log in to AWS somehow.

Deploy

```sh
pulumi stack select honeycomb-devrel/booth-game-api
pulumi up
```
