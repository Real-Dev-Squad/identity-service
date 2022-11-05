cd health-check
go mod tidy
go test -v
npx kill-port 8090
cd ../verify
go mod tidy
npx firebase --project="test" emulators:exec "go test"
npx kill-port 8090
cd ../profile
go mod tidy
if (npx firebase --project="test" emulators:exec "go test"); then
    echo "Exited Success"
else
    exit 1
fi
npx kill-port 8090