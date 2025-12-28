#!/bin/bash
# Wrapper script to run venture with correct library path for CGO

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Set DYLD_LIBRARY_PATH for macOS (or LD_LIBRARY_PATH for Linux)
if [[ "$OSTYPE" == "darwin"* ]]; then
    export DYLD_LIBRARY_PATH="${DYLD_LIBRARY_PATH}:${SCRIPT_DIR}/bsp/cgal"
else
    export LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:${SCRIPT_DIR}/bsp/cgal"
fi

# Run venture with all arguments passed through
go run "${SCRIPT_DIR}" "$@"

