#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="${SCRIPT_DIR}/.tailscale-build"
OUTPUT_DIR="${SCRIPT_DIR}/app/libs"

echo "=== Building libtailscale.aar ==="

# Check prerequisites
for cmd in go git; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: $cmd is required but not found in PATH"
        exit 1
    fi
done

# Clone or update tailscale-android
if [ -d "$BUILD_DIR/tailscale-android" ]; then
    echo "Updating tailscale-android..."
    cd "$BUILD_DIR/tailscale-android"
    git pull --ff-only
else
    echo "Cloning tailscale-android..."
    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"
    git clone https://github.com/tailscale/tailscale-android.git
    cd tailscale-android
fi

# Install gomobile if needed
if ! command -v gomobile &>/dev/null; then
    echo "Installing gomobile..."
    go install golang.org/x/mobile/cmd/gomobile@latest
    go install golang.org/x/mobile/cmd/gobind@latest
    gomobile init
fi

# Build libtailscale.aar
echo "Building libtailscale.aar (this may take a while)..."
make libtailscale.aar

# Copy output
mkdir -p "$OUTPUT_DIR"
cp libtailscale.aar "$OUTPUT_DIR/libtailscale.aar"

echo "=== Done! Output: $OUTPUT_DIR/libtailscale.aar ==="
ls -lh "$OUTPUT_DIR/libtailscale.aar"
