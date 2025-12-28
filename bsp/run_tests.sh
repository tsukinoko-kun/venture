#!/bin/bash
# Helper script to run BSP tests

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Run go test with all arguments passed through
cd "$PROJECT_ROOT"
go test ./bsp "$@"
