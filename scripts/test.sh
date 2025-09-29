#!/bin/bash
# Create temporary directory for coverage files
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

go mod tidy

declare -a exit_codes=()
declare -a modules=("health")
declare -a firebase_modules=("call-profile" "verify" "health-check" "call-profiles")

# Run modules that don't need Firebase emulator
for module in "${modules[@]}"; do
    echo "Running $module tests..."
    cd "$module" || exit 1
    go test -v -cover -coverprofile="$TEMP_DIR/$module.out"
    exit_codes+=($?)
    npx kill-port 8090 2>/dev/null || true
    cd .. || exit 1
done

# Run modules that need Firebase emulator
for module in "${firebase_modules[@]}"; do
    echo "Running $module tests with Firebase emulator..."
    cd "$module" || exit 1
    npx firebase-tools --project="test" emulators:exec "go test -v -cover -coverprofile=\"$TEMP_DIR/$module.out\""
    MODULE_EXIT=$?
    [ $MODULE_EXIT -ne 0 ] && echo "Firebase test exited with error ($module)"
    exit_codes+=($MODULE_EXIT)
    npx kill-port 8090 2>/dev/null || true
    cd .. || exit 1
done

echo "================================"
echo "COVERAGE REPORTS"
echo "================================"

all_modules=("${modules[@]}" "${firebase_modules[@]}")
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