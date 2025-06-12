#!/bin/bash
# Create temporary directory for coverage files
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

go mod tidy
cd health
# Run health tests with coverage
go test -v -cover -coverprofile="$TEMP_DIR/health.out"
HEALTH_EXIT=$?
npx kill-port 8090
cd ../verify
# Use npx instead of global installation
npx firebase-tools --project="test" emulators:exec "go test -v -cover -coverprofile=\"$TEMP_DIR/verify.out\"" --timeout=60s || {
    echo "Firebase test exited with error or timed out"
    exit 1
}
echo "Exited Success"
npx kill-port 8090

# Display coverage reports in console
cd ..
echo "================================"
echo "COVERAGE REPORTS"
echo "================================"
echo "Health module coverage:"
go tool cover -func="$TEMP_DIR/health.out"
echo "================================"
echo "Verify module coverage:"
go tool cover -func="$TEMP_DIR/verify.out"
echo "================================"

# Display total coverage
echo "Combined coverage:"
echo "mode: set" > "$TEMP_DIR/all.out"
tail -n +2 -q "$TEMP_DIR/health.out" "$TEMP_DIR/verify.out" >> "$TEMP_DIR/all.out" 2>/dev/null
go tool cover -func="$TEMP_DIR/all.out"
echo "================================"

# Exit with proper exit code
[ $HEALTH_EXIT -ne 0 ] && exit $HEALTH_EXIT
exit 0