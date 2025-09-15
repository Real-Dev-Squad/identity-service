#!/bin/bash
# Create temporary directory for coverage files
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

go mod tidy

# Run health tests with coverage
cd health
go test -v -cover -coverprofile="$TEMP_DIR/health.out"
HEALTH_EXIT=$?
npx kill-port 8090
cd ..

cd call-profile
go test -v -cover -coverprofile="$TEMP_DIR/call-profile.out"
CALL_PROFILE_EXIT=$?
cd ..

cd call-profiles
go test -v -cover -coverprofile="$TEMP_DIR/call-profiles.out"
CALL_PROFILES_EXIT=$?
cd ..

cd health-check
go test -v -cover -coverprofile="$TEMP_DIR/health-check.out"
HEALTH_CHECK_EXIT=$?
cd ..

cd verify
# Use npx instead of global installation
npx firebase-tools --project="test" emulators:exec "go test -v -cover -coverprofile=\"$TEMP_DIR/verify.out\"" --timeout=60s || {
    echo "Firebase test exited with error or timed out"
    exit 1
}
echo "Exited Success"
npx kill-port 8090
cd ..

# Display coverage reports in console
echo "================================"
echo "COVERAGE REPORTS"
echo "================================"
echo "Health module coverage:"
go tool cover -func="$TEMP_DIR/health.out"
echo "================================"
echo "Call-profile module coverage:"
go tool cover -func="$TEMP_DIR/call-profile.out"
echo "================================"
echo "Call-profiles module coverage:"
go tool cover -func="$TEMP_DIR/call-profiles.out"
echo "================================"
echo "Health-check module coverage:"
go tool cover -func="$TEMP_DIR/health-check.out"
echo "================================"
echo "Verify module coverage:"
go tool cover -func="$TEMP_DIR/verify.out"
echo "================================"

# Display total coverage
echo "Combined coverage:"
echo "mode: set" > "$TEMP_DIR/all.out"
tail -n +2 -q "$TEMP_DIR/health.out" "$TEMP_DIR/call-profile.out" "$TEMP_DIR/call-profiles.out" "$TEMP_DIR/health-check.out" "$TEMP_DIR/verify.out" >> "$TEMP_DIR/all.out" 2>/dev/null
go tool cover -func="$TEMP_DIR/all.out"
echo "================================"

# Exit with proper exit code
[ $HEALTH_EXIT -ne 0 ] && exit $HEALTH_EXIT
[ $CALL_PROFILE_EXIT -ne 0 ] && exit $CALL_PROFILE_EXIT
[ $CALL_PROFILES_EXIT -ne 0 ] && exit $CALL_PROFILES_EXIT
[ $HEALTH_CHECK_EXIT -ne 0 ] && exit $HEALTH_CHECK_EXIT
exit 0