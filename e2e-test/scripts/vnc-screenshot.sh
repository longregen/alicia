#!/usr/bin/env bash
# vnc-screenshot.sh - Capture desktop screenshot in NixOS test environment
set -euo pipefail

NAME="${1:-screenshot}"
ARTIFACT_DIR="${ARTIFACT_DIR:-./test-results}"
OUTPUT_DIR="${ARTIFACT_DIR}/screenshots/desktop"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
FILENAME="${NAME}-${TIMESTAMP}.png"

mkdir -p "$OUTPUT_DIR"

# Use scrot for X11 screenshot
DISPLAY="${DISPLAY:-:0}" scrot "$OUTPUT_DIR/$FILENAME"

echo "Desktop screenshot saved: $OUTPUT_DIR/$FILENAME"
