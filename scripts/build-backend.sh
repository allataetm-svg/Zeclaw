#!/bin/bash
set -e

BINARY_DIR="frontend/assets/backend"
mkdir -p "$BINARY_DIR"

echo "Building backend for Android ARM64..."
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build \
  -o "$BINARY_DIR/zeclaw-backend-arm64" \
  ./backend/cmd/zeclaw/

echo "Building backend for Android ARM32..."
GOOS=android GOARCH=arm CGO_ENABLED=0 go build \
  -o "$BINARY_DIR/zeclaw-backend-arm" \
  ./backend/cmd/zeclaw/

echo "Done! Binaries:"
ls -lh "$BINARY_DIR/"
