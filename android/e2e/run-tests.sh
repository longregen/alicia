#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Resolve adb and maestro paths
ADB="${ADB:-adb}"
if ! command -v "$ADB" &> /dev/null; then
    # Try Android SDK default location
    if [ -x "$HOME/.android/sdk/platform-tools/adb" ]; then
        ADB="$HOME/.android/sdk/platform-tools/adb"
    elif [ -n "$ANDROID_HOME" ] && [ -x "$ANDROID_HOME/platform-tools/adb" ]; then
        ADB="$ANDROID_HOME/platform-tools/adb"
    else
        echo -e "${RED}Error: adb not found. Set ADB= or install Android SDK platform-tools.${NC}"
        exit 1
    fi
fi

MAESTRO="${MAESTRO:-maestro}"
if ! command -v "$MAESTRO" &> /dev/null; then
    if [ -x "$HOME/.maestro/bin/maestro" ]; then
        MAESTRO="$HOME/.maestro/bin/maestro"
    else
        echo -e "${YELLOW}Maestro not found. Installing...${NC}"
        curl -Ls https://get.maestro.mobile.dev | bash
        MAESTRO="$HOME/.maestro/bin/maestro"
    fi
fi

# Restart ADB to ensure clean connection (Maestro loses ADB between runs)
restart_adb() {
    "$ADB" kill-server 2>/dev/null || true
    sleep 1
    "$ADB" start-server 2>/dev/null || true
    sleep 1
}

restart_adb

# Check device connected
if ! "$ADB" devices | grep -q "device$"; then
    echo -e "${RED}Error: No Android device/emulator connected.${NC}"
    exit 1
fi

DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
echo -e "${GREEN}Using device: $DEVICE${NC}"

# Create screenshots directory
mkdir -p screenshots

# Install APK if provided
if [ -n "$1" ]; then
    echo -e "${YELLOW}Installing APK: $1${NC}"
    "$ADB" -s "$DEVICE" install -r "$1"
fi

# Run tests in order: onboarding first, then feature tests
echo -e "${GREEN}=== Running E2E Tests ===${NC}"

run_test() {
    local test_file="$1"
    local test_name=$(basename "$test_file" .yaml)

    # Restart ADB before each test to avoid Maestro connection issues
    restart_adb

    echo -e "\n${YELLOW}Running: $test_name${NC}"
    if "$MAESTRO" test "$test_file"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        return 1
    fi
}

# Track results
PASSED=0
FAILED=0

# Core tests (onboarding must run first — it grants permissions for later tests)
TESTS=(onboarding.yaml voice_interaction.yaml conversations.yaml)

# assistant_setup.yaml tests system Settings UI which is device-specific
# (stock Google AOSP API 34). Include it when present; skip gracefully on
# non-Google images where the Settings text differs.
if [ -f assistant_setup.yaml ]; then
    # Revoke the assistant role so the ASSISTANT onboarding page appears.
    # clearState only revokes runtime permissions, not system roles.
    echo -e "${YELLOW}Revoking assistant role for clean assistant_setup test...${NC}"
    if ! "$ADB" -s "$DEVICE" shell cmd role remove-role-holder android.app.role.ASSISTANT com.alicia.assistant 2>&1; then
        echo -e "${YELLOW}Warning: could not revoke assistant role (may not be held)${NC}"
    fi

    TESTS+=(assistant_setup.yaml)
fi

for test_file in "${TESTS[@]}"; do
    if [ -f "$test_file" ]; then
        if run_test "$test_file"; then
            PASSED=$((PASSED + 1))
        else
            FAILED=$((FAILED + 1))
        fi
    fi
done

# Summary
echo -e "\n${GREEN}=== Test Summary ===${NC}"
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Screenshots saved to: $(pwd)/screenshots/"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
