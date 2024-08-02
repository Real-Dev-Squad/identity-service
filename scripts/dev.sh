go mod tidy
sam.cmd build
sam.cmd local start-api --env-vars env.json