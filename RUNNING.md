# Running Venture with Level Editor

## Quick Start

The simplest way to run the level editor during development:

```bash
./run.sh level levels/test.yaml
```

This script automatically sets the required library path for the CGAL collision detection library.

## Alternative Methods

### Method 1: Using the wrapper script (Recommended)

```bash
./run.sh level levels/test.yaml
```

### Method 2: Set environment variable manually

**macOS:**
```bash
export DYLD_LIBRARY_PATH="${DYLD_LIBRARY_PATH}:${PWD}/bsp/cgal"
go run . level levels/test.yaml
```

**Linux:**
```bash
export LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:${PWD}/bsp/cgal"
go run . level levels/test.yaml
```

### Method 3: Using the compiled binary

```bash
# Build
go build

# Run (macOS)
DYLD_LIBRARY_PATH="${PWD}/bsp/cgal" ./venture level levels/test.yaml

# Run (Linux)
LD_LIBRARY_PATH="${PWD}/bsp/cgal" ./venture level levels/test.yaml
```

## Why is this needed?

The level editor's collision test tool uses CGAL (Computational Geometry Algorithms Library) via CGO. The C++ wrapper library (`libpartition.dylib` on macOS, `libpartition.so` on Linux) must be findable at runtime. 

The environment variable tells the dynamic linker where to find the library:
- `DYLD_LIBRARY_PATH` on macOS
- `LD_LIBRARY_PATH` on Linux

## Troubleshooting

If you see an error like:
```
dyld[...]: Library not loaded: libpartition.dylib
```

This means the library path is not set. Use one of the methods above.

If the library doesn't exist, build it first:
```bash
cd bsp/cgal
make
```

This requires CGAL to be installed:
- **macOS**: `brew install cgal`
- **Linux**: `sudo apt-get install libcgal-dev`

