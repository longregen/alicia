#!/usr/bin/env bash
set -e

HEADSCALE_DIR="/tmp/headscale-test"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

if [ -f "$HEADSCALE_DIR/headscale.pid" ]; then
    PID=$(cat "$HEADSCALE_DIR/headscale.pid")
    if kill -0 "$PID" 2>/dev/null; then
        kill "$PID"
        echo -e "${GREEN}Stopped Headscale (PID $PID)${NC}"
    else
        echo -e "${RED}Headscale process $PID already stopped${NC}"
    fi
else
    echo -e "${RED}No PID file found at $HEADSCALE_DIR/headscale.pid${NC}"
fi

rm -rf "$HEADSCALE_DIR"
echo -e "${GREEN}Cleaned up $HEADSCALE_DIR${NC}"
