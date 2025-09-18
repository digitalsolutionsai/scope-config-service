#!/bin/bash
# This script handles the protobuf generation.
# It first checks for required dependencies and then runs the generation command.

set -e

# --- Dependency Check ---
echo "Checking for required tools..."

# Check for buf
if ! command -v buf &> /dev/null; then
    echo "Error: 'buf' is not installed. Please install it to continue."
    echo "See installation instructions: https://buf.build/docs/installation"
    exit 1
fi

# Check for go
if ! command -v go &> /dev/null; then
    echo "Error: 'go' is not installed. Please install it to continue."
    exit 1
fi

# We need to check for protoc-gen-go and protoc-gen-go-grpc.
# These are typically installed in GOPATH/bin.
GOPATH_VAL=$(go env GOPATH)
GO_BIN_PATH="$GOPATH_VAL/bin"

# Check for protoc-gen-go by checking the user's PATH first, then GOPATH/bin
if ! command -v protoc-gen-go &> /dev/null && [ ! -f "$GO_BIN_PATH/protoc-gen-go" ]; then
    echo "Error: 'protoc-gen-go' is not installed or not in your PATH."
    echo "Install it by running: go install google.golang.org/protobuf/cmd/protoc-gen-go"
    exit 1
fi

# Check for protoc-gen-go-grpc by checking the user's PATH first, then GOPATH/bin
if ! command -v protoc-gen-go-grpc &> /dev/null && [ ! -f "$GO_BIN_PATH/protoc-gen-go-grpc" ]; then
    echo "Error: 'protoc-gen-go-grpc' is not installed or not in your PATH."
    echo "Install it by running: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc"
    exit 1
fi

echo "All required tools are available."
# --- End Dependency Check ---


# Add the Go bin directory to the PATH to ensure buf can find the plugins.
# This is crucial for environments where GOPATH/bin is not in the default PATH.
export PATH="$PATH:$GO_BIN_PATH"

# Generate the protobuf code.
echo "Generating protobuf code..."
buf generate
echo "Protobuf code generated successfully."
