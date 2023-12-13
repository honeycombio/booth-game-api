# Api for the booth game

This is the backend API for the booth game, it's primarily a go app handling those requests.

(right now, there is no local debugging)

## Build and Deploy.

Build and package

```sh
cd src
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ../HandleRequest main.go && zip ../HandleRequest.zip ../HandleRequest
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