#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

HEADSCALE_DIR="/tmp/headscale-test"

# Resolve tool paths from nix store or PATH
find_tool() {
    local name="$1" pattern="$2"
    command -v "$name" 2>/dev/null && return
    local p
    p=$(echo /nix/store/*-${pattern}/bin/${name} 2>/dev/null | tr ' ' '\n' | tail -1)
    [ -x "$p" ] && echo "$p" && return
    p=$(echo /nix/store/*-${pattern}/${name} 2>/dev/null | tr ' ' '\n' | tail -1)
    [ -x "$p" ] && echo "$p" && return
    return 1
}

HEADSCALE=$(find_tool headscale "headscale-*") || {
    echo -e "${RED}headscale not found. Add pkgs.headscale to devShell or install it.${NC}"
    exit 1
}

echo -e "${YELLOW}Starting Headscale for VPN E2E tests...${NC}"

rm -rf "$HEADSCALE_DIR"
mkdir -p "$HEADSCALE_DIR"
cp headscale-config.yaml "$HEADSCALE_DIR/config.yaml"

"$HEADSCALE" serve --config "$HEADSCALE_DIR/config.yaml" &
HEADSCALE_PID=$!
echo "$HEADSCALE_PID" > "$HEADSCALE_DIR/headscale.pid"

echo -e "${YELLOW}Waiting for Headscale to be ready...${NC}"
for i in $(seq 1 30); do
    if curl -sf http://127.0.0.1:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}Headscale is ready (PID $HEADSCALE_PID)${NC}"
        break
    fi
    if ! kill -0 "$HEADSCALE_PID" 2>/dev/null; then
        echo -e "${RED}Headscale process died unexpectedly${NC}"
        exit 1
    fi
    sleep 0.5
done

if ! curl -sf http://127.0.0.1:8080/health > /dev/null 2>&1; then
    echo -e "${RED}Headscale failed to start within 15s${NC}"
    kill "$HEADSCALE_PID" 2>/dev/null || true
    exit 1
fi

echo -e "${YELLOW}Creating test user...${NC}"
"$HEADSCALE" users create --config "$HEADSCALE_DIR/config.yaml" test

USER_ID=$("$HEADSCALE" users list --config "$HEADSCALE_DIR/config.yaml" -o json | \
    python3 -c "import sys,json; users=json.load(sys.stdin); print(next(u['id'] for u in users if u['name']=='test'))")
echo -e "${GREEN}Created user 'test' (ID: $USER_ID)${NC}"

echo -e "${YELLOW}Creating pre-auth key...${NC}"
AUTHKEY=$("$HEADSCALE" preauthkeys create --config "$HEADSCALE_DIR/config.yaml" --user "$USER_ID" --reusable -o json | \
    python3 -c "import sys,json; print(json.load(sys.stdin)['key'])")

echo "$AUTHKEY" > "$HEADSCALE_DIR/authkey.txt"

echo -e "${GREEN}Headscale ready${NC}"
echo -e "  URL:      http://10.0.2.2:8080"
echo -e "  Auth key: ${AUTHKEY:0:12}..."
echo -e "  PID:      $HEADSCALE_PID"
echo -e "  Dir:      $HEADSCALE_DIR"
