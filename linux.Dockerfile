# Multi-stage Dockerfile for building Venture on Linux x86_64
# This enforces x86_64 architecture and validates dynamic linking
# Note: --platform=linux/amd64 is intentionally hardcoded to ensure x86_64 build on ARM Macs

# Build stage
FROM --platform=linux/amd64 golang:1.23.4-bookworm AS builder

# Allow Go toolchain auto-download to match go.mod requirements
ENV GOTOOLCHAIN=auto

# Install build dependencies for CGAL and C++ compilation
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libcgal-dev \
    libgmp-dev \
    libmpfr-dev \
    libboost-dev \
    g++ \
    make \
    gcc libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev libgtk-3-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the CGAL static library
WORKDIR /build/bsp/cgal
RUN make clean && make static

# Build the Go application
WORKDIR /build
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o venture .

# Runtime stage - minimal image to test linking
FROM --platform=linux/amd64 debian:bookworm-slim

# Install runtime dependencies
# GUI libraries (without -dev packages) needed for Gio UI framework
# GMP/MPFR are statically linked so not needed here
RUN apt-get update && apt-get install -y \
    libstdc++6 \
    libwayland-egl1 \
    libwayland-client0 \
    libwayland-cursor0 \
    libxkbcommon-x11-0 \
    libxkbcommon0 \
    libx11-xcb1 \
    libx11-6 \
    libxcursor1 \
    libxfixes3 \
    libegl1 \
    libgles2 \
    ca-certificates \
    file \
    && rm -rf /var/lib/apt/lists/*

# Copy the built binary from builder
COPY --from=builder /build/venture /venture

# Set the entrypoint to show dynamic library dependencies
ENTRYPOINT ["sh", "-c", "echo '=== Dynamic Library Dependencies ===' && ldd /venture && echo '' && echo '=== Binary Info ===' && file /venture"]
