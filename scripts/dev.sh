go mod tidy
sam build
sam local start-api --env-vars env.json