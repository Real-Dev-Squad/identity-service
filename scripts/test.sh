#!/bin/bash
# Create temporary directory for coverage files
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

go mod tidy

declare -a exit_codes=()
declare -a modules=("health" "call-profile" "call-profiles" "health-check")

for module in "${modules[@]}"; do
    echo "Running $module tests..."
    cd "$module" || exit 1
    go test -v -cover -coverprofile="$TEMP_DIR/$module.out"
    exit_codes+=($?)
    npx kill-port 8090 2>/dev/null || true
    cd .. || exit 1
done

echo "Running verify tests..."
cd verify || exit 1
npx firebase-tools --project="test" emulators:exec "go test -v -cover -coverprofile=\"$TEMP_DIR/verify.out\""
VERIFY_EXIT=$?
[ $VERIFY_EXIT -ne 0 ] && echo "Firebase test exited with error (verify)"
exit_codes+=($VERIFY_EXIT)
npx kill-port 8090 2>/dev/null || true
cd .. || exit 1

echo "================================"
echo "COVERAGE REPORTS"
echo "================================"

all_modules=("${modules[@]}" "verify")
for module in "${all_modules[@]}"; do
    if [ -f "$TEMP_DIR/$module.out" ]; then
        echo "================================"
        echo "$(echo ${module:0:1} | tr '[:lower:]' '[:upper:]')${module:1} module coverage:"
        go tool cover -func="$TEMP_DIR/$module.out"
    fi
done

echo "================================"
echo "Combined coverage:"
echo "mode: set" > "$TEMP_DIR/all.out"
for module in "${all_modules[@]}"; do
    if [ -f "$TEMP_DIR/$module.out" ]; then
        tail -n +2 "$TEMP_DIR/$module.out" >> "$TEMP_DIR/all.out" 2>/dev/null
    fi
done
go tool cover -func="$TEMP_DIR/all.out"
echo "================================"

for exit_code in "${exit_codes[@]}"; do
    [ "$exit_code" -ne 0 ] && exit "$exit_code"
done

exit 0