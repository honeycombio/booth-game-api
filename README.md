# Observaquiz

Take the observaquiz: [quiz.honeydemo.io]()

This is the backend API for the observaquiz game, it's primarily a go app handling those requests.

## Local debugging

`export HONEYCOMB_API_KEY=your api key`

This uses AWS SAM (Serverless Application Model).

We define the API in `template.yaml`, with its name etc. this looks for it on the filesystem as a relative path.

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

```
./build.sh
```

#### Deploy one-time setup

Go to the infra directory

```sh
cd infra
```

install packages

```
npm i
```

Log in to pulumi via the CLI, and do some one-time things

```sh
pulumi login
pulumi stack select honeycomb-devrel/prod
pulumi config refresh
```

Follow the prompts to login

Log in to AWS somehow.

#### Deploy

In the root of the project,

```sh
./deploy.sh
```

## changing secrets

in infra:

```sh
pulumi config set --secret observaquiz-api:openai-api-key <value>
```

and then deploy again.
