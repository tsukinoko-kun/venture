# Venture

<img src="venture.webp" width="256" height="256">

This is the build tool for the Venture game engine.

## Features

- **Import Linting**: Scans Odin source files for forbidden imports that prevent console portability
- **Protobuf Generation**: Automatically generates Odin code from `.proto` files
- **Clay C Library Compilation**: Compiles the Clay UI library with clang
- **Steam Library Management**: Downloads and manages Steamworks SDK libraries
- **Odin Compilation**: Builds the Odin game with platform-specific settings
- **Distribution Packaging**: 
  - **macOS**: Creates a zip archive with the binary, assets, and all shared libraries bundled using `dylibbundler`
  - **Linux**: Creates AppImage packages with all dependencies using `linuxdeploy`
  - **Windows**: Creates a zip archive with the binary, assets, and all required DLLs
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
venture build [--platform PLATFORM] [--debug] [--release]
```

**Note**: Venture only builds for the current operating system, as cross-compilation is not supported when bundling shared libraries.

Options:
- `--platform, -p`: Storefront platform (steam/fallback, default: fallback)
- `--debug, -d`: Build with debug symbols
- `--release, -r`: Build with optimizations

Example:
```bash
# Build for current platform with fallback platform
venture build

# Build with Steam platform and optimizations
venture build --platform steam --release
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

- **platform**: Platform detection
- **linter**: Import linting for console portability
- **formatter**: Odin code formatting
- **protobuf**: Protobuf code generation
- **clay**: Clay C library compilation
- **steamworks**: Steam library management and downloading
- **odin**: Odin compilation
- **packager**: Distribution packaging (creates zip archives on macOS/Windows, AppImages on Linux)

Each command in `cmd/` orchestrates these packages in a high-level, declarative way.

## Requirements

- Go 1.25.5 or later
- Odin compiler
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-odin` (Odin protobuf plugin)
- `odinfmt` (Odin formatter)
- `clang` (for Clay C compilation)
- **macOS only**: `dylibbundler` (install with: `brew install dylibbundler`)
- **Linux only**: `linuxdeploy` (download from: https://github.com/linuxdeploy/linuxdeploy/releases)

## Output

- **Build command on macOS**: Creates zip archive in `./build/` directory
  - Example: `./build/adventurer-darwin_arm64.zip`
  - Contains: binary, `assets/` directory, and `libs/` directory with all shared libraries bundled
  - Extract and run the binary - shared libraries will be found via `@executable_path/libs/`
  
- **Build command on Linux**: Creates AppImage in `./build/` directory
  - Example: `./build/adventurer-linux_amd64.AppImage`
  - Contains: executable, assets/, and all dependencies bundled
  - Self-contained, portable, and executable on most Linux distributions

- **Build command on Windows**: Creates zip archive in `./build/` directory
  - Example: `./build/adventurer-windows_amd64.zip`
  - Contains: executable (`.exe`), `assets/` directory, and all required DLLs
  - Extract and run the executable - DLLs will be found in the same directory
  
- **Run command**: Builds to project root and executes immediately

## Error Handling

All errors are wrapped with context using `fmt.Errorf`. Commands exit with non-zero status codes on any error, making them suitable for CI/CD pipelines.

