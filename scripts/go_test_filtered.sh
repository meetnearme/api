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

# Run go test in each of those directories
for dir in $DIRS; do
  if ls $dir/*.go &> /dev/null; then
    go test $VERBOSE $dir
  fi
done
