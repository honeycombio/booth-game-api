docker_compose("./local-collector/docker-compose.yml")
dc_resource("collector")

local_resource("API",
    deps= ["cmd/api"],
	cmd = "GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/api cmd/api/*.go",
)

local_resource("Callback",
    deps= ["cmd/deepchecks_callback"],
	cmd = "GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o ./dist/deepchecks_callback cmd/deepchecks_callback/*.go",
)

local_resource("Lambda Simulator",
    deps=["environment.json", "template.yaml"],
	serve_cmd = "sam local start-api --env-vars environment.json --docker-network local-collector_collector_net",
	resource_deps = ["collector", "API", "Callback"],
	links = ["http://127.0.0.1:3000/"]
)
