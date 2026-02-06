#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"
ANDROID_DIR="$(cd .. && pwd)"
PROJECT_ROOT="$(cd ../.. && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

ADB=$(find_tool adb "platform-tools-*") || { echo -e "${RED}adb not found${NC}"; exit 1; }
MAESTRO=$(find_tool maestro "maestro-*") || { echo -e "${RED}maestro not found${NC}"; exit 1; }
MAGICK=$(find_tool magick "imagemagick-*") || MAGICK=""

export MAESTRO_CLI_NO_ANALYTICS=1
export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED=true

# Build APK with Nix (includes Vosk model)
echo -e "${YELLOW}Building APK with Nix (includes bundled models)...${NC}"
cd "$PROJECT_ROOT"
nix build .#android --no-link --print-out-paths > /tmp/nix-android-out.txt 2>&1 || {
    echo -e "${RED}Nix build failed:${NC}"
    cat /tmp/nix-android-out.txt
    exit 1
}
NIX_OUT=$(cat /tmp/nix-android-out.txt | tail -1)
echo -e "${GREEN}Built: $NIX_OUT${NC}"
cd "$ANDROID_DIR/e2e"

# Pick device
restart_adb() {
    "$ADB" kill-server 2>/dev/null || true
    sleep 1
    "$ADB" start-server 2>/dev/null || true
    sleep 2
}

restart_adb

DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
if [ -z "$DEVICE" ]; then
    echo -e "${RED}No Android device/emulator connected.${NC}"
    exit 1
fi
export ANDROID_SERIAL="$DEVICE"
echo -e "${GREEN}Using device: $DEVICE${NC}"

# Determine APK to install based on device architecture
ARCH=$("$ADB" -s "$DEVICE" shell getprop ro.product.cpu.abi | tr -d '\r')
case "$ARCH" in
    arm64-v8a) APK="$NIX_OUT/app-arm64-v8a-debug.apk" ;;
    armeabi-v7a) APK="$NIX_OUT/app-armeabi-v7a-debug.apk" ;;
    x86_64) APK="$NIX_OUT/app-x86_64-debug.apk" ;;
    x86) APK="$NIX_OUT/app-x86-debug.apk" ;;
    *) APK="$NIX_OUT/app-universal-debug.apk" ;;
esac

if [ ! -f "$APK" ]; then
    echo -e "${RED}APK not found: $APK${NC}"
    ls -la "$NIX_OUT/"
    exit 1
fi

echo -e "${YELLOW}Installing $APK...${NC}"
"$ADB" -s "$DEVICE" install -r "$APK" || {
    echo -e "${YELLOW}Install failed, trying uninstall first...${NC}"
    "$ADB" -s "$DEVICE" uninstall com.alicia.assistant 2>/dev/null || true
    "$ADB" -s "$DEVICE" install "$APK"
}
echo -e "${GREEN}Installed${NC}"

# Clear old screenshots and logs
mkdir -p screenshots
rm -f screenshots/*.png
rm -f logcat.txt

# Clear logcat buffer before tests
"$ADB" -s "$DEVICE" logcat -c 2>/dev/null || true

# Run tests in dependency order
TESTS=(onboarding.yaml voice_interaction.yaml conversations.yaml)
PASSED=0
FAILED=0

for test_file in "${TESTS[@]}"; do
    [ -f "$test_file" ] || continue
    test_name=$(basename "$test_file" .yaml)
    restart_adb
    echo -e "\n${YELLOW}Running: $test_name${NC}"
    if "$MAESTRO" test "$test_file"; then
        echo -e "${GREEN}  PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}  FAILED${NC}"
        FAILED=$((FAILED + 1))
    fi
done

# Resize screenshots to 50% (540x1200) so they fit in Claude's 2000px image input
count=$(ls screenshots/*.png 2>/dev/null | wc -l)
if [ "$count" -gt 0 ]; then
    if [ -n "$MAGICK" ]; then
        echo -e "\n${YELLOW}Resizing $count screenshots to 50%...${NC}"
        for f in screenshots/*.png; do
            "$MAGICK" "$f" -resize 50% "$f"
        done
        echo -e "${GREEN}Done${NC}"
    else
        echo -e "\n${YELLOW}magick not found, skipping resize. Install imagemagick or run: nix-shell -p imagemagick${NC}"
    fi
fi

# Dump logcat to file (restart adb first to ensure connection)
restart_adb
echo -e "\n${YELLOW}Capturing logcat...${NC}"
"$ADB" -s "$DEVICE" logcat -d -v threadtime > logcat.txt 2>&1 || true
LOGCAT_LINES=$(wc -l < logcat.txt)
echo -e "${GREEN}Captured $LOGCAT_LINES lines${NC}"

# Summary
echo -e "\n${GREEN}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Screenshots: $(pwd)/screenshots/"
echo -e "Logcat: $(pwd)/logcat.txt ($LOGCAT_LINES lines)"

[ "$FAILED" -gt 0 ] && exit 1
exit 0
