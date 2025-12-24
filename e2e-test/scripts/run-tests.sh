#!/usr/bin/env bash
set -euo pipefail

SERVER_URL="${ALICIA_SERVER_URL:-http://server}"
ARTIFACT_DIR="${ARTIFACT_DIR:-./output}"

mkdir -p "$ARTIFACT_DIR/screenshots/browser"
mkdir -p "$ARTIFACT_DIR/screenshots/desktop"
mkdir -p "$ARTIFACT_DIR/logs"
mkdir -p "$ARTIFACT_DIR/traces"
mkdir -p "$ARTIFACT_DIR/dom"

echo "[E2E] Waiting for server at $SERVER_URL..."
for i in $(seq 1 60); do
    if curl -sf "${SERVER_URL}/health" > /dev/null 2>&1; then
        echo "[E2E] Server is ready!"
        break
    fi
    sleep 2
done

echo "[E2E] Running tests..."
ARTIFACT_DIR="$ARTIFACT_DIR" npx playwright test --headed 2>&1 | tee "$ARTIFACT_DIR/logs/playwright-output.log"
