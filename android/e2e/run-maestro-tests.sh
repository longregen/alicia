#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"
ANDROID_DIR="$(cd .. && pwd)"
PROJECT_ROOT="$(cd ../.. && pwd)"

# Usage
usage() {
    echo "Usage: $0 [options] [test_name ...]"
    echo ""
    echo "Run Maestro E2E tests for the Alicia Android app."
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help"
    echo "  -l, --list     List available tests"
    echo "  -s, --skip-build  Skip APK build (use already-installed app)"
    echo ""
    echo "Examples:"
    echo "  $0                          # Run all tests"
    echo "  $0 conversations            # Run only the conversations test"
    echo "  $0 onboarding conversations # Run onboarding and conversations"
    echo "  $0 vpn                      # Run only VPN test"
    echo ""
    echo "Test names (partial match supported):"
    echo "  onboarding, voice, conversations, vpn"
    exit 0
}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Parse arguments
SKIP_BUILD=0
TEST_FILTERS=()
for arg in "$@"; do
    case "$arg" in
        -h|--help) usage ;;
        -l|--list)
            echo "Available tests:"
            for f in *.yaml; do
                echo "  $(basename "$f" .yaml)"
            done
            exit 0
            ;;
        -s|--skip-build) SKIP_BUILD=1 ;;
        -*) echo "Unknown option: $arg"; usage ;;
        *) TEST_FILTERS+=("$arg") ;;
    esac
done

# Re-exec inside nix develop .#android if not already there
if [ -z "$IN_NIX_ANDROID_SHELL" ]; then
    if ! command -v nix &>/dev/null; then
        echo -e "${RED}nix is not installed.${NC}"
        echo -e "This script requires the Nix android dev shell. Install Nix first, then run:"
        echo -e "  ${GREEN}cd $PROJECT_ROOT && nix develop .#android --impure${NC}"
        echo -e "  ${GREEN}cd android/e2e && ./run-maestro-tests.sh${NC}"
        exit 1
    fi
    echo -e "${YELLOW}Entering nix android dev shell...${NC}"
    export IN_NIX_ANDROID_SHELL=1
    exec env NIXPKGS_ALLOW_UNFREE=1 NIXPKGS_ACCEPT_ANDROID_SDK_LICENSE=1 \
        nix develop "$PROJECT_ROOT#android" --impure -c bash "$(realpath "$0")" "$@"
fi

# Verify required tools are available
for tool in adb emulator maestro; do
    if ! command -v "$tool" &>/dev/null; then
        echo -e "${RED}$tool not found in PATH.${NC}"
        echo -e "Run this script from the Nix android dev shell:"
        echo -e "  ${GREEN}cd $(cd "$PROJECT_ROOT" && pwd)${NC}"
        echo -e "  ${GREEN}NIXPKGS_ALLOW_UNFREE=1 NIXPKGS_ACCEPT_ANDROID_SDK_LICENSE=1 nix develop .#android --impure${NC}"
        echo -e "  ${GREEN}cd android/e2e && ./run-maestro-tests.sh${NC}"
        exit 1
    fi
done

ADB="adb"
MAESTRO="maestro"
MAGICK=$(command -v magick 2>/dev/null) || MAGICK=""

export MAESTRO_CLI_NO_ANALYTICS=1
export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED=true

# Build APK with Nix (includes Vosk model)
if [ "$SKIP_BUILD" -eq 0 ]; then
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
else
    echo -e "${YELLOW}Skipping build (--skip-build)${NC}"
fi

# Pick device
restart_adb() {
    "$ADB" kill-server 2>/dev/null || true
    sleep 1
    "$ADB" start-server 2>/dev/null || true
    sleep 2
    # Wait for a device to reconnect after server restart
    local i=0
    while [ $i -lt 15 ]; do
        if "$ADB" devices 2>/dev/null | grep -q 'device$'; then
            return 0
        fi
        sleep 1
        i=$((i + 1))
    done
    echo -e "${YELLOW}Warning: no device found after ADB restart${NC}"
}

restart_adb

# Start emulator if no device connected
DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
if [ -z "$DEVICE" ]; then
    AVD=$(emulator -list-avds 2>/dev/null | head -1)
    if [ -z "$AVD" ]; then
        echo -e "${RED}No Android device/emulator connected and no AVDs available.${NC}"
        exit 1
    fi
    echo -e "${YELLOW}Starting emulator ($AVD)...${NC}"
    EMULATOR_STARTED=1
    emulator -avd "$AVD" -no-window -no-audio -gpu swiftshader_indirect &
    EMULATOR_PID=$!

    # Wait for emulator to boot
    echo -e "${YELLOW}Waiting for emulator to boot...${NC}"
    "$ADB" wait-for-device
    # Wait for boot_completed property
    i=0
    while [ $i -lt 120 ]; do
        if "$ADB" shell getprop sys.boot_completed 2>/dev/null | grep -q '1'; then
            break
        fi
        sleep 2
        i=$((i + 2))
    done
    if ! "$ADB" shell getprop sys.boot_completed 2>/dev/null | grep -q '1'; then
        echo -e "${RED}Emulator failed to boot.${NC}"
        kill "$EMULATOR_PID" 2>/dev/null || true
        exit 1
    fi
    echo -e "${GREEN}Emulator booted${NC}"
    DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
fi

if [ -z "$DEVICE" ]; then
    echo -e "${RED}No Android device/emulator connected.${NC}"
    exit 1
fi
export ANDROID_SERIAL="$DEVICE"
echo -e "${GREEN}Using device: $DEVICE${NC}"

if [ "$SKIP_BUILD" -eq 0 ]; then
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
fi

# Start Headscale + mock API for VPN tests if available (skip if VPN test not selected)
HEADSCALE_RUNNING=0
MOCK_API_PID=""
vpn_needed=1
if [ ${#TEST_FILTERS[@]} -gt 0 ]; then
    vpn_needed=0
    for filter in "${TEST_FILTERS[@]}"; do
        [[ "vpn_registration" == *"$filter"* ]] && vpn_needed=1
    done
fi
if [ "$vpn_needed" -eq 1 ] && [ -f "vpn_registration.yaml" ] && [ -f "start-headscale.sh" ]; then
    echo -e "\n${YELLOW}Starting Headscale for VPN tests...${NC}"
    if bash start-headscale.sh; then
        HEADSCALE_RUNNING=1
        export HEADSCALE_URL="http://10.0.2.2:8080"
        export HEADSCALE_AUTHKEY="$(cat /tmp/headscale-test/authkey.txt)"
        echo -e "${GREEN}Headscale ready${NC}"

        # Start mock Alicia API server for auto-provisioning
        if [ -f "mock-alicia-api.py" ]; then
            echo -e "\n${YELLOW}Starting mock Alicia API...${NC}"
            HEADSCALE_PREAUTH_KEY="$HEADSCALE_AUTHKEY" \
            HEADSCALE_URL="http://10.0.2.2:8080" \
            MOCK_API_PORT=8181 \
                python3 mock-alicia-api.py &
            MOCK_API_PID=$!
            # Wait for mock API to be ready
            for i in $(seq 1 10); do
                if curl -sf http://127.0.0.1:8181/health > /dev/null 2>&1; then
                    echo -e "${GREEN}Mock API ready (PID $MOCK_API_PID)${NC}"
                    break
                fi
                sleep 0.3
            done

            # Push API URL override so the app calls our mock server.
            # Launch the app briefly to create the data dir, then write the override.
            echo -e "${YELLOW}Configuring app API URL override...${NC}"
            "$ADB" -s "$DEVICE" shell am start -n com.alicia.assistant/.MainActivity > /dev/null 2>&1
            sleep 2
            "$ADB" -s "$DEVICE" shell am force-stop com.alicia.assistant
            "$ADB" -s "$DEVICE" shell "run-as com.alicia.assistant sh -c 'echo -n http://10.0.2.2:8181 > /data/data/com.alicia.assistant/files/api_url_override.txt'" 2>/dev/null || \
                echo -e "${YELLOW}Could not write API override via run-as${NC}"
            # Verify it was written
            "$ADB" -s "$DEVICE" shell "run-as com.alicia.assistant cat files/api_url_override.txt" 2>/dev/null && \
                echo -e "\n${GREEN}API URL override set${NC}" || \
                echo -e "${YELLOW}Warning: API URL override may not have been written${NC}"
        fi
    else
        echo -e "${YELLOW}Headscale failed to start, skipping VPN tests${NC}"
    fi
fi

# Clear old screenshots and logs
mkdir -p screenshots
rm -f screenshots/*.png
rm -f logcat.txt

# Clear logcat buffer before tests
"$ADB" -s "$DEVICE" logcat -c 2>/dev/null || true

# Run tests in dependency order
ALL_TESTS=(onboarding.yaml voice_interaction.yaml conversations.yaml)
if [ "$HEADSCALE_RUNNING" -eq 1 ]; then
    ALL_TESTS+=(vpn_registration.yaml)
fi

# Filter tests if specific names were given
TESTS=()
if [ ${#TEST_FILTERS[@]} -gt 0 ]; then
    for test_file in "${ALL_TESTS[@]}"; do
        test_name=$(basename "$test_file" .yaml)
        for filter in "${TEST_FILTERS[@]}"; do
            if [[ "$test_name" == *"$filter"* ]]; then
                TESTS+=("$test_file")
                break
            fi
        done
    done
    if [ ${#TESTS[@]} -eq 0 ]; then
        echo -e "${RED}No tests matched filters: ${TEST_FILTERS[*]}${NC}"
        echo "Available: ${ALL_TESTS[*]}"
        exit 1
    fi
else
    TESTS=("${ALL_TESTS[@]}")
fi
PASSED=0
FAILED=0

for test_file in "${TESTS[@]}"; do
    [ -f "$test_file" ] || continue
    test_name=$(basename "$test_file" .yaml)
    restart_adb
    DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
    export ANDROID_SERIAL="$DEVICE"
    echo -e "\n${YELLOW}Running: $test_name (device: $DEVICE)${NC}"
    MAESTRO_ENV=""
    if [ "$test_name" = "vpn_registration" ] && [ -n "$MOCK_API_PID" ]; then
        # Re-write the API URL override (clearState in earlier tests may have wiped it).
        # Launch the app briefly so the filesDir exists, then write the override.
        echo -e "${YELLOW}  Re-writing API URL override for VPN test...${NC}"
        "$ADB" shell am start -n com.alicia.assistant/.MainActivity > /dev/null 2>&1
        sleep 3
        "$ADB" shell am force-stop com.alicia.assistant
        "$ADB" shell "run-as com.alicia.assistant sh -c 'echo -n http://10.0.2.2:8181 > /data/data/com.alicia.assistant/files/api_url_override.txt'" 2>/dev/null
        echo -e "${GREEN}  API URL override re-written${NC}"
        # Restart ADB to ensure clean connection for Maestro
        restart_adb
        DEVICE=$("$ADB" devices | grep 'device$' | head -1 | awk '{print $1}')
        export ANDROID_SERIAL="$DEVICE"
    fi
    if "$MAESTRO" test $MAESTRO_ENV "$test_file"; then
        echo -e "${GREEN}  PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}  FAILED${NC}"
        FAILED=$((FAILED + 1))
    fi
done

# Stop mock API and Headscale if they were started
if [ -n "$MOCK_API_PID" ] && kill -0 "$MOCK_API_PID" 2>/dev/null; then
    echo -e "\n${YELLOW}Stopping mock API (PID $MOCK_API_PID)...${NC}"
    kill "$MOCK_API_PID" 2>/dev/null || true
    # Remove API URL override
    "$ADB" -s "$DEVICE" shell "run-as com.alicia.assistant rm -f files/api_url_override.txt" 2>/dev/null || true
fi
if [ "$HEADSCALE_RUNNING" -eq 1 ]; then
    echo -e "\n${YELLOW}Stopping Headscale...${NC}"
    bash stop-headscale.sh 2>/dev/null || true
fi

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

# Stop emulator if we started it
if [ "${EMULATOR_STARTED:-0}" -eq 1 ] && [ -n "${EMULATOR_PID:-}" ]; then
    echo -e "\n${YELLOW}Stopping emulator...${NC}"
    "$ADB" -s "$DEVICE" emu kill 2>/dev/null || kill "$EMULATOR_PID" 2>/dev/null || true
    wait "$EMULATOR_PID" 2>/dev/null || true
    echo -e "${GREEN}Emulator stopped${NC}"
fi

[ "$FAILED" -gt 0 ] && exit 1
exit 0
