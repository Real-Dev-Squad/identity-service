cd health-check
go mod tidy
go test -v
npx kill-port 8090
cd ../verify
go mod tidy
npm install -g firebase-tools
if (firebase --project="test" emulators:exec "go test"); then
    echo "Exited Success"
else
    exit 1
fi
npx kill-port 8090