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

# Run modules that need Firebase emulator in parallel
CONCURRENCY=2  # Limit to 2 to avoid port conflicts
pids=()
module_names=()

echo "Starting Firebase emulator tests with concurrency limit of $CONCURRENCY..."

for module in "${firebase_modules[@]}"; do
    # Wait if we've reached concurrency limit
    while (( ${#pids[@]} >= CONCURRENCY )); do
        # Check for completed processes
        for i in "${!pids[@]}"; do
            if ! kill -0 "${pids[$i]}" 2>/dev/null; then
                # Process completed, remove from array
                unset pids[$i]
                unset module_names[$i]
            fi
        done
        # Rebuild array without gaps
        pids=("${pids[@]}")
        module_names=("${module_names[@]}")
        sleep 0.1
    done
    
    # Start new test in background
    (
        echo "Starting $module tests with Firebase emulator..."
        cd "$module" || exit 1
        
        # Use different ports for parallel execution to avoid conflicts
        PORT_OFFSET=$((RANDOM % 1000 + 1000))
        FIRESTORE_PORT=$((8090 + PORT_OFFSET))
        UI_PORT=$((4000 + PORT_OFFSET))
        
        # Create temporary firebase.json with unique ports
        cat > firebase.json << EOF
{
  "emulators": {
    "firestore": {
      "port": $FIRESTORE_PORT
    },
    "ui": {
      "enabled": true,
      "port": $UI_PORT
    }
  }
}
EOF
        
        # Set emulator host environment variable
        export FIRESTORE_EMULATOR_HOST="127.0.0.1:$FIRESTORE_PORT"
        
        npx firebase-tools --project="test" emulators:exec "go test -v -cover -coverprofile=\"$TEMP_DIR/$module.out\""
        MODULE_EXIT=$?
        
        # Cleanup
        npx kill-port $FIRESTORE_PORT 2>/dev/null || true
        npx kill-port $UI_PORT 2>/dev/null || true
        rm -f firebase.json
        
        if [ $MODULE_EXIT -ne 0 ]; then
            echo "Firebase test exited with error ($module)"
        else
            echo "Completed $module tests successfully"
        fi
        
        exit $MODULE_EXIT
    ) &
    
    pid=$!
    pids+=($pid)
    module_names+=($module)
    echo "Started $module tests (PID: $pid)"
done

# Wait for all background processes to complete
echo "Waiting for all Firebase emulator tests to complete..."
for i in "${!pids[@]}"; do
    wait "${pids[$i]}"
    exit_code=$?
    module_name="${module_names[$i]}"
    exit_codes+=($exit_code)
    echo "Module $module_name completed with exit code $exit_code"
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