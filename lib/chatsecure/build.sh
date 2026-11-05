#!/bin/bash
set -e

# build.sh - Build script for TorBox ChatSecure (tcs)
# Usage:
#   ./build.sh          # Builds tcs for current host OS/ARCH
#   ./build.sh arm64    # Builds tcs-arm64 (Raspberry Pi 3/4/5)
#   ./build.sh armv6    # Builds tcs-armv6 (Raspberry Pi Zero/1)
#   ./build.sh all      # Builds all targets

TARGET=${1:-local}

case "$TARGET" in
  local)
    echo "==> Building tcs for local platform..."
    go build -ldflags="-s -w" -o tcs main.go
    echo "==> Successfully built tcs"
    ;;
  arm64)
    echo "==> Cross-compiling tcs for Linux ARM64 (Raspberry Pi 3/4/5)..."
    GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o tcs-arm64 main.go
    echo "==> Successfully built tcs-arm64"
    ;;
  armv6)
    echo "==> Cross-compiling tcs for Linux ARMv6 (Raspberry Pi Zero/1)..."
    GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o tcs-armv6 main.go
    echo "==> Successfully built tcs-armv6"
    ;;
  all)
    echo "==> Building local tcs..."
    go build -ldflags="-s -w" -o tcs main.go
    echo "==> Building tcs-arm64..."
    GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o tcs-arm64 main.go
    echo "==> Building tcs-armv6..."
    GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o tcs-armv6 main.go
    echo "==> All binaries built successfully!"
    ;;
  *)
    echo "Unknown target: $TARGET"
    echo "Usage: ./build.sh [local|arm64|armv6|all]"
    exit 1
    ;;
esac
