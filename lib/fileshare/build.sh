#!/bin/bash
set -e

# build.sh - Build script for TorBox FileShare (tfs)
# Usage:
#   ./build.sh          # Builds tfs for current host OS/ARCH
#   ./build.sh arm64    # Builds tfs-arm64 (Raspberry Pi 3/4/5)
#   ./build.sh armv6    # Builds tfs-armv6 (Raspberry Pi Zero/1)
#   ./build.sh all      # Builds all targets

TARGET=${1:-local}

case "$TARGET" in
  local)
    echo "==> Building tfs for local platform..."
    go build -ldflags="-s -w" -o tfs main.go
    echo "==> Successfully built tfs"
    ;;
  arm64)
    echo "==> Cross-compiling tfs for Linux ARM64 (Raspberry Pi 3/4/5)..."
    GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o tfs-arm64 main.go
    echo "==> Successfully built tfs-arm64"
    ;;
  armv6)
    echo "==> Cross-compiling tfs for Linux ARMv6 (Raspberry Pi Zero/1)..."
    GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o tfs-armv6 main.go
    echo "==> Successfully built tfs-armv6"
    ;;
  all)
    echo "==> Building local tfs..."
    go build -ldflags="-s -w" -o tfs main.go
    echo "==> Building tfs-arm64..."
    GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o tfs-arm64 main.go
    echo "==> Building tfs-armv6..."
    GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o tfs-armv6 main.go
    echo "==> All binaries built successfully!"
    ;;
  *)
    echo "Unknown target: $TARGET"
    echo "Usage: ./build.sh [local|arm64|armv6|all]"
    exit 1
    ;;
esac
