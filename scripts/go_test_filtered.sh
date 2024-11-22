#!/bin/bash

# This script runs go test in all directories excluding node_modules, because
# AWS CDK is a node_module that includes Go test files with invalid names that fail

# Check if -v option is passed
VERBOSE=""
if [[ "$1" == "-v" ]]; then
  VERBOSE="-v"
fi

# Find all directories excluding node_modules
DIRS=$(find . -type d -not -path "./node_modules*")

# Create a temporary coverage file
COVERAGE_FILE=$(mktemp)

# Initialize coverage file with mode header
echo "mode: set" > "$COVERAGE_FILE"

# Run go test in each of those directories
for dir in $DIRS; do
  if ls $dir/*.go &> /dev/null; then
    go test $VERBOSE -coverprofile=coverage.out $dir
    if [ -f coverage.out ]; then
      # Append coverage data without the mode line
      tail -n +2 coverage.out >> "$COVERAGE_FILE"
      rm coverage.out
    fi
  fi
done

# Display total coverage if we have any test results
if [ -f "$COVERAGE_FILE" ]; then
  echo -e "\nTotal test coverage:"
  go tool cover -func="$COVERAGE_FILE" | grep total: | awk '{print $3}'
  rm "$COVERAGE_FILE"
fi
