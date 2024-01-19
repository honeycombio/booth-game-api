# build.sh

GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api src/api/*.go
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o deep_checks_callback src/deep_checks_callback/*.go

