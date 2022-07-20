cd health-check
go mod tidy
go test -v
npx kill-port 8090
cd ../verify
go mod tidy
npx firebase emulators:exec "go test -v"
npx kill-port 8090
cd ../profile
go mod tidy
npx firebase emulators:exec "go test -v"
npx kill-port 8090