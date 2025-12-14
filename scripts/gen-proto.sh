#!/bin/bash
# This script handles the protobuf generation for both Go and Python.
# It first checks for required dependencies and then runs the generation commands.

set -e

# --- Dependency Check ---
echo "Checking for required tools..."

# Check for go
if ! command -v go &> /dev/null; then
    echo "Error: 'go' is not installed. Please install it to continue."
    exit 1
fi

# Add the Go bin directory to the PATH early so we can find installed tools like buf.
GOPATH_VAL=$(go env GOPATH)
GO_BIN_PATH="$GOPATH_VAL/bin"
export PATH="$PATH:$GO_BIN_PATH"

# Check for buf
if ! command -v buf &> /dev/null; then
    echo "Error: 'buf' is not installed. Please install it to continue."
    echo "See installation instructions: https://buf.build/docs/installation"
    exit 1
fi

if ! command -v protoc-gen-go &> /dev/null && [ ! -f "$GO_BIN_PATH/protoc-gen-go" ]; then
    echo "Error: 'protoc-gen-go' is not installed or not in your PATH."
    echo "Install it by running: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

if ! command -v protoc-gen-go-grpc &> /dev/null && [ ! -f "$GO_BIN_PATH/protoc-gen-go-grpc" ]; then
    echo "Error: 'protoc-gen-go-grpc' is not installed or not in your PATH."
    echo "Install it by running: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

# Check for the grpcio-tools Python package.
# if ! python3 -c "import grpc_tools.protoc" &> /dev/null; then
#     echo "Error: 'grpcio-tools' Python package is not installed."
#     echo "Install it by running: pip install grpcio-tools"
#     exit 1
# fi

echo "All required tools are available."
# --- End Dependency Check ---

# Add the Go bin directory to the PATH to ensure buf can find the plugins.
export PATH="$PATH:$GO_BIN_PATH"

# --- Go Protobuf Generation ---
echo "Generating Go protobuf code..."
buf generate
echo "Go protobuf code generated successfully."

# --- Python Protobuf Generation ---
# echo "Generating Python protobuf code..."

# Create the Python output directory
# PYTHON_OUTPUT_DIR="./sdks/python/gen"
# mkdir -p "$PYTHON_OUTPUT_DIR"

# Define the proto file path
# PROTO_FILE="./proto/config/v1/config.proto"
# INCLUDE_DIR="./proto"

# python3 -m grpc_tools.protoc \
#     -I="$INCLUDE_DIR" \
#     --python_out="$PYTHON_OUTPUT_DIR" \
#     --grpc_python_out="$PYTHON_OUTPUT_DIR" \
#     "$PROTO_FILE"

# echo "Python protobuf code generated successfully in $PYTHON_OUTPUT_DIR."