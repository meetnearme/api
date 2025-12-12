#!/bin/bash

# This script runs go test in all directories excluding node_modules, because
# AWS CDK is a node_module that includes Go test files with invalid names that fail

# Set test environment
export GO_ENV=test

# Check if -v option is passed
VERBOSE=""
if [[ "$1" == "-v" ]]; then
  VERBOSE="-v"
fi

echo "🧪 Running Go tests with timeouts..."
echo "🔧 Test environment: $GO_ENV"

# Find all directories excluding node_modules
DIRS=$(find ./functions -type d -not -path "./node_modules*" -not -path "*/node_modules*")

# Create a temporary coverage file
COVERAGE_FILE=$(mktemp)

# Initialize coverage file with mode header
echo "mode: set" > "$COVERAGE_FILE"

# Track failed directories
FAILED_DIRS=()
PASSED_DIRS=()

# Run go test in each of those directories
for dir in $DIRS; do
  if ls $dir/*.go &> /dev/null; then
    echo "🔍 Testing: $dir"

    # Use Go's built-in timeout to prevent hanging
    # -timeout 30s ensures tests don't hang forever
    if go test $VERBOSE -timeout 30s -coverprofile=coverage.out $dir 2>/dev/null; then
      echo "   ✅ PASSED"
      PASSED_DIRS+=("$dir")

      if [ -f coverage.out ]; then
        # Append coverage data without the mode line
        tail -n +2 coverage.out >> "$COVERAGE_FILE"
        rm coverage.out
      fi
    else
      echo "   ❌ FAILED or TIMEOUT"
      FAILED_DIRS+=("$dir")
    fi
  fi
done

# Display results summary
echo ""
echo "📊 TEST SUMMARY:"
echo "   ✅ Passed: ${#PASSED_DIRS[@]} directories"
echo "   ❌ Failed: ${#FAILED_DIRS[@]} directories"

if [ ${#FAILED_DIRS[@]} -gt 0 ]; then
  echo ""
  echo "❌ Failed directories:"
  for dir in "${FAILED_DIRS[@]}"; do
    echo "   - $dir"
  done
fi

# Display total coverage if we have any test results
if [ -f "$COVERAGE_FILE" ] && [ -s "$COVERAGE_FILE" ]; then
  echo ""
  echo "📊 Total test coverage:"
  go tool cover -func="$COVERAGE_FILE" | grep total: | awk '{print $3}'
  rm "$COVERAGE_FILE"
fi

echo ""
echo "🎉 Test run complete!"
