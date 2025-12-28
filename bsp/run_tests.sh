#!/bin/bash
# Helper script to run BSP tests with correct library path

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Set library path for CGAL shared library
export DYLD_LIBRARY_PATH="$SCRIPT_DIR/cgal:$DYLD_LIBRARY_PATH"
export LD_LIBRARY_PATH="$SCRIPT_DIR/cgal:$LD_LIBRARY_PATH"

# Run go test with all arguments passed through
cd "$PROJECT_ROOT"
go test ./bsp "$@"

