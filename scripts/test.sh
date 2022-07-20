cd health-check
go mod tidy
go test -v
npx kill-port 8090
cd ../verify
go mod tidy
firebase emulators:exec "go test -v"
npx kill-port 8090
cd ../profile
go mod tidy
firebase emulators:exec "go test -v"
npx kill-port 8090