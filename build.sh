# build.sh
date
if [ ! -d dist ] ; then
    mkdir dist
fi
rm -rf dist/*
echo "Building regular api..."
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/api cmd/api/*.go
echo "building deepchecks callback..."
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/deepchecks_callback cmd/deepchecks_callback/*.go

