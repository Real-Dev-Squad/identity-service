cd health
go mod tidy
cd ../health-check
go mod tidy
cd ../verify
go mod tidy
cd ../call-profile
go mod tidy
cd ../call-profiles
go mod tidy
cd ..
sam build
sam local start-api --env-vars env.json