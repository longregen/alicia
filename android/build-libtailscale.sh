#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DIR="${SCRIPT_DIR}/.tailscale-build"
OUTPUT_DIR="${SCRIPT_DIR}/app/libs"

echo "=== Building libtailscale.aar ==="

for cmd in go git curl; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: $cmd is required but not found in PATH"
        exit 1
    fi
done

if [ -z "${ANDROID_HOME:-}" ]; then
    echo "Error: ANDROID_HOME is not set. Run from the nix android devShell:"
    echo "  nix develop .#android"
    exit 1
fi

if [ -z "${ANDROID_NDK_HOME:-}" ]; then
    NDK_DIR=$(ls -d "$ANDROID_HOME/ndk/"* 2>/dev/null | head -1)
    if [ -n "$NDK_DIR" ]; then
        export ANDROID_NDK_HOME="$NDK_DIR"
    else
        echo "Error: Android NDK not found. Ensure ndk is in the SDK composition."
        exit 1
    fi
fi

echo "Using NDK: $ANDROID_NDK_HOME"

# Clone or update tailscale-android
if [ -d "$BUILD_DIR/tailscale-android" ]; then
    echo "Updating tailscale-android..."
    cd "$BUILD_DIR/tailscale-android"
    git pull --ff-only 2>/dev/null || true
else
    echo "Cloning tailscale-android..."
    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"
    git clone --depth 1 https://github.com/tailscale/tailscale-android.git
    cd tailscale-android
fi

# Use Tailscale's custom Go toolchain (handles version requirements)
echo "Setting up Tailscale Go toolchain..."
bash ./tool/go version

TAILSCALE_GO="$HOME/.cache/tailscale-go/bin/go"
export GOROOT="$HOME/.cache/tailscale-go"
export GOTOOLCHAIN=local
export GOWORK=off

GOBIN="$PWD/android/build/go/bin"
export GOBIN
mkdir -p "$GOBIN"

echo "Using Go: $($TAILSCALE_GO version)"

# Install gomobile/gobind
if [ ! -f "$GOBIN/gomobile" ]; then
    echo "Installing gomobile..."
    $TAILSCALE_GO install golang.org/x/mobile/cmd/gomobile@latest
    $TAILSCALE_GO install golang.org/x/mobile/cmd/gobind@latest
fi

# Update x/mobile to match gomobile version
echo "Ensuring x/mobile compatibility..."
$TAILSCALE_GO get golang.org/x/mobile@latest
$TAILSCALE_GO mod tidy

# Download Go dependencies
echo "Downloading Go dependencies..."
$TAILSCALE_GO mod download

# Build AAR with gomobile bind
# GOWORK=off is critical — prevents parent go.work from interfering
echo "Building libtailscale.aar (this takes several minutes)..."
export PATH="$GOROOT/bin:$GOBIN:$PATH"
"$GOBIN/gomobile" bind \
    -target android \
    -androidapi 26 \
    -o libtailscale.aar \
    ./libtailscale

mkdir -p "$OUTPUT_DIR"
cp libtailscale.aar "$OUTPUT_DIR/libtailscale.aar"

echo "=== Done! Output: $OUTPUT_DIR/libtailscale.aar ==="
ls -lh "$OUTPUT_DIR/libtailscale.aar"
