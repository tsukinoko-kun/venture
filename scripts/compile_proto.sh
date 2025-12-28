#!/bin/bash

# Script to compile protobuf definitions to Go code
# Requires protoc and protoc-gen-go to be installed
# Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed. Please install Protocol Buffers compiler."
    echo "On macOS: brew install protobuf"
    echo "On Linux: apt-get install protobuf-compiler"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go is not installed."
    echo "Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/proto/level"

# Compile the protobuf
echo "Compiling proto/level.proto..."
protoc \
    --proto_path="$PROJECT_ROOT/proto" \
    --go_out="$PROJECT_ROOT/proto/level" \
    --go_opt=paths=source_relative \
    "$PROJECT_ROOT/proto/level.proto"

echo "✓ Protobuf compilation complete!"
echo "Generated: proto/level/level.pb.go"

