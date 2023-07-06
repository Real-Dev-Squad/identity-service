cd health-check
go mod tidy
cd ../verify
go mod tidy
cd ../profile
go mod tidy
cd ..
sam.cmd build
sam.cmd local start-api --env-vars env.json