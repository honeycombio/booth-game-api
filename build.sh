# build.sh
echo "Building regular api..."
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o api cmd/api/*.go
echo "building deepchecks callback..."
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o deep_checks_callback cmd/deep_checks_callback/*.go

