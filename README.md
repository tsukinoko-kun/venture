# Venture

Venture is a comprehensive build tool for the Adventurer game engine. It replaces the Python-based `build.py` with a clean, modular Go implementation using Cobra for CLI management.

## Features

- **Import Linting**: Scans Odin source files for forbidden imports that prevent console portability
- **Protobuf Generation**: Automatically generates Odin code from `.proto` files
- **Clay C Library Compilation**: Compiles the Clay UI library with cross-compilation support via Zig
- **Steam Library Management**: Downloads and manages Steamworks SDK libraries
- **Odin Compilation**: Builds the Odin game with platform-specific settings
- **Distribution Packaging**: Creates zip archives ready for distribution to Steam and other platforms
- **Code Formatting**: Runs `odinfmt` on the source code

## Installation

```bash
cd venture
go build -o venture
```

Then move the `venture` binary to a location in your PATH, or use it directly from the venture directory.

## Usage

### Commands

#### Build
Build and package for distribution:
```bash
venture build [--target TARGET] [--platform PLATFORM] [--debug] [--release]
```

Options:
- `--target, -t`: Target platform (windows, linux, macos, macos-intel, or Odin target format)
- `--platform, -p`: Storefront platform (steam/fallback, default: fallback)
- `--debug, -d`: Build with debug symbols
- `--release, -r`: Build with optimizations

Example:
```bash
# Build for current platform with fallback platform
venture build

# Build for Windows with Steam platform
venture build --target windows --platform steam --release

# Build for macOS with optimizations
venture build --target macos --release
```

#### Run
Build and run for the current platform (development):
```bash
venture run [--platform PLATFORM] [--debug] [--release]
```

Options:
- `--platform, -p`: Storefront platform (steam/fallback, default: fallback)
- `--debug, -d`: Build with debug symbols
- `--release, -r`: Build with optimizations

Example:
```bash
# Run with fallback platform
venture run

# Run with Steam platform and debug symbols
venture run --platform steam --debug
```

#### Lint
Check source code for console portability issues:
```bash
venture lint
```

#### Format
Format Odin source code:
```bash
venture fmt [--check]
```

Options:
- `--check`: Check formatting without modifying files (dry run)

## Architecture

Venture is organized into focused packages:

- **platform**: Platform detection and target mapping
- **linter**: Import linting for console portability
- **formatter**: Odin code formatting
- **protobuf**: Protobuf code generation
- **clay**: Clay C library compilation with cross-compilation support
- **steamworks**: Steam library management and downloading
- **odin**: Odin compilation
- **packager**: Distribution packaging (creates zip archives)

Each command in `cmd/` orchestrates these packages in a high-level, declarative way.

## Requirements

- Go 1.25.5 or later
- Odin compiler
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-odin` (Odin protobuf plugin)
- `odinfmt` (Odin formatter)
- `clang` (for Clay C compilation)
- `zig` (optional, for cross-compilation)

## Cross-Compilation

When building for a different platform than the current one:
- **Clay C library**: Venture attempts to use Zig for cross-compilation. If Zig is not available, it will fail with a helpful error message.
- **Odin**: Native cross-compilation support via the `-target` flag
- **Steam libraries**: Pre-built binaries are downloaded from GitHub

## Output

- **Build command**: Creates zip files in `./build/` directory
  - Example: `./build/adventurer-darwin_arm64.zip`
  - Contains: executable, assets/, and platform libraries (if applicable)
  
- **Run command**: Builds to project root and executes immediately

## Error Handling

All errors are wrapped with context using `fmt.Errorf`. Commands exit with non-zero status codes on any error, making them suitable for CI/CD pipelines.

