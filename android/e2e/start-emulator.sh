#!/usr/bin/env bash
set -e

AVD_NAME="alicia-test"
API_LEVEL="34"

# Check if running in nix shell with Android SDK
if [ -z "$ANDROID_HOME" ]; then
    echo "Error: ANDROID_HOME not set. Run: nix develop .#android"
    exit 1
fi

# Check KVM availability - required for x86_64 emulator
if [ ! -e /dev/kvm ]; then
    echo "Error: /dev/kvm not found. Hardware acceleration is required for x86_64 emulator."
    echo ""
    echo "To enable KVM on NixOS, add to your configuration.nix:"
    echo "  boot.kernelModules = [ \"kvm-intel\" ];  # or kvm-amd"
    echo "  virtualisation.libvirtd.enable = true;"
    echo ""
    echo "Or load the module temporarily:"
    echo "  sudo modprobe kvm-intel  # or kvm-amd"
    echo ""
    echo "Alternatives:"
    echo "  - Use a physical Android device: adb devices"
    echo "  - Run E2E tests on GitHub Actions (has KVM enabled)"
    echo "  - Skip this check: SKIP_KVM_CHECK=1 $0 $*"
    echo ""
    if [ -z "$SKIP_KVM_CHECK" ]; then
        exit 1
    fi
    echo "SKIP_KVM_CHECK set, continuing anyway (will likely fail)..."
fi

# Set up writable SDK directory for system images and AVDs
export ANDROID_USER_HOME="${ANDROID_USER_HOME:-$HOME/.android}"
export ANDROID_AVD_HOME="$ANDROID_USER_HOME/avd"

mkdir -p "$ANDROID_USER_HOME"
mkdir -p "$ANDROID_AVD_HOME"

SYSTEM_IMAGE="system-images;android-${API_LEVEL};google_apis;x86_64"
PLATFORM="platforms;android-${API_LEVEL}"

# Use system Android SDK if available (non-Nix), otherwise use Nix workaround
if command -v /opt/android-sdk/cmdline-tools/latest/bin/sdkmanager &>/dev/null; then
    # Standard Android SDK installation
    ANDROID_SDK_ROOT="/opt/android-sdk"
    export ANDROID_SDK_ROOT
    SDK_MANAGER="sdkmanager"
    AVD_MANAGER="avdmanager"
else
    # Nix environment: use FHS wrapper or download SDK components manually
    # The Nix SDK is read-only, so we create a writable overlay
    LOCAL_SDK="$ANDROID_USER_HOME/sdk"
    mkdir -p "$LOCAL_SDK/platforms"
    mkdir -p "$LOCAL_SDK/system-images"
    mkdir -p "$LOCAL_SDK/licenses"

    # Copy licenses from Nix SDK
    cp -f "$ANDROID_HOME/licenses/"* "$LOCAL_SDK/licenses/" 2>/dev/null || true

    # Link tools from Nix SDK
    for dir in platform-tools emulator build-tools cmdline-tools; do
        if [ -d "$ANDROID_HOME/$dir" ] && [ ! -e "$LOCAL_SDK/$dir" ]; then
            ln -sf "$ANDROID_HOME/$dir" "$LOCAL_SDK/$dir"
        fi
    done

    export ANDROID_SDK_ROOT="$LOCAL_SDK"

    # Download platform if needed using commandlinetools directly
    if [ ! -d "$LOCAL_SDK/platforms/android-${API_LEVEL}" ]; then
        echo "Downloading Android $API_LEVEL platform SDK..."
        if ! yes | "$LOCAL_SDK/cmdline-tools/latest/bin/sdkmanager" --sdk_root="$LOCAL_SDK" "$PLATFORM" 2>&1; then
            echo "Warning: Platform download may have issues, continuing..."
        fi
    fi

    # Download system image if needed
    if [ ! -d "$LOCAL_SDK/system-images/android-${API_LEVEL}/google_apis/x86_64" ]; then
        echo "Downloading Android $API_LEVEL system image..."
        if ! yes | "$LOCAL_SDK/cmdline-tools/latest/bin/sdkmanager" --sdk_root="$LOCAL_SDK" "$SYSTEM_IMAGE" 2>&1; then
            echo "Warning: System image download may have issues, continuing..."
        fi
    fi

    SDK_MANAGER="$LOCAL_SDK/cmdline-tools/latest/bin/sdkmanager --sdk_root=$LOCAL_SDK"
    AVD_MANAGER="$LOCAL_SDK/cmdline-tools/latest/bin/avdmanager"
fi

# Accept licenses
yes | $SDK_MANAGER --licenses > /dev/null 2>&1 || true

# Verify system image exists
if [ ! -d "$ANDROID_SDK_ROOT/system-images/android-${API_LEVEL}/google_apis/x86_64" ]; then
    echo "Error: System image not found at $ANDROID_SDK_ROOT/system-images/android-${API_LEVEL}/google_apis/x86_64"
    echo ""
    echo "On NixOS, you may need to download manually:"
    echo "  $SDK_MANAGER '$SYSTEM_IMAGE'"
    echo ""
    echo "Or use a pre-built system image from another source."
    exit 1
fi

# Create AVD if it doesn't exist
# Note: avdmanager has issues with Nix SDK, so we create the AVD config manually
if [ ! -d "$ANDROID_AVD_HOME/$AVD_NAME.avd" ]; then
    echo "Creating AVD: $AVD_NAME"
    mkdir -p "$ANDROID_AVD_HOME/$AVD_NAME.avd"

    # Create the .ini file that points to the AVD folder
    cat > "$ANDROID_AVD_HOME/$AVD_NAME.ini" << EOF
avd.ini.encoding=UTF-8
path=$ANDROID_AVD_HOME/$AVD_NAME.avd
path.rel=avd/$AVD_NAME.avd
target=android-$API_LEVEL
EOF

    # Create the config.ini inside the AVD folder
    cat > "$ANDROID_AVD_HOME/$AVD_NAME.avd/config.ini" << EOF
PlayStore.enabled=false
abi.type=x86_64
avd.ini.encoding=UTF-8
disk.dataPartition.size=6G
fastboot.chosenSnapshotFile=
fastboot.forceChosenSnapshotBoot=no
fastboot.forceColdBoot=no
fastboot.forceFastBoot=yes
hw.accelerometer=yes
hw.arc=false
hw.audioInput=yes
hw.battery=yes
hw.camera.back=virtualscene
hw.camera.front=emulated
hw.cpu.arch=x86_64
hw.cpu.ncore=4
hw.dPad=no
hw.device.hash2=MD5:3db3250dab5d0d93b29353040571e578
hw.device.manufacturer=Google
hw.device.name=pixel_6
hw.gps=yes
hw.gpu.enabled=yes
hw.gpu.mode=auto
hw.initialOrientation=Portrait
hw.keyboard=yes
hw.lcd.density=411
hw.lcd.height=2400
hw.lcd.width=1080
hw.mainKeys=no
hw.ramSize=2048
hw.sdCard=yes
hw.sensors.orientation=yes
hw.sensors.proximity=yes
hw.trackBall=no
image.sysdir.1=system-images/android-${API_LEVEL}/google_apis/x86_64/
runtime.network.latency=none
runtime.network.speed=full
sdcard.size=512M
showDeviceFrame=no
skin.dynamic=yes
skin.name=1080x2400
skin.path=_no_skin
tag.display=Google APIs
tag.id=google_apis
vm.heapSize=256
EOF
    echo "AVD created"
fi

# Start emulator - use our local SDK for system images
echo "Starting emulator..."

# Use offscreen Qt platform to avoid display dependency
# Set DNS servers manually since emulator can't detect them on NixOS
ANDROID_SDK_ROOT="$ANDROID_SDK_ROOT" \
ANDROID_HOME="$ANDROID_SDK_ROOT" \
QT_QPA_PLATFORM=offscreen \
"$ANDROID_SDK_ROOT/emulator/emulator" -avd "$AVD_NAME" \
    -no-snapshot \
    -no-window \
    -gpu swiftshader_indirect \
    -grpc 8556 \
    -dns-server 8.8.8.8 &

EMULATOR_PID=$!

# Wait for the emulator to appear in adb and detect its serial
echo "Waiting for emulator to boot..."
SERIAL=""
timeout=120
while [ -z "$SERIAL" ]; do
    sleep 2
    timeout=$((timeout - 2))
    if [ $timeout -le 0 ]; then
        echo "Error: Emulator did not appear in adb"
        kill $EMULATOR_PID 2>/dev/null || true
        exit 1
    fi
    SERIAL=$(adb devices | grep -E '^emulator-' | head -1 | awk '{print $1}')
done
echo "Emulator serial: $SERIAL"

# Wait for boot
adb -s "$SERIAL" wait-for-device
while [ "$(adb -s "$SERIAL" shell getprop sys.boot_completed 2>/dev/null)" != "1" ]; do
    sleep 2
    timeout=$((timeout - 2))
    if [ $timeout -le 0 ]; then
        echo "Error: Emulator failed to boot in time"
        kill $EMULATOR_PID 2>/dev/null || true
        exit 1
    fi
done
echo "Emulator ready!"

# Disable animations for testing
adb -s "$SERIAL" shell settings put global window_animation_scale 0
adb -s "$SERIAL" shell settings put global transition_animation_scale 0
adb -s "$SERIAL" shell settings put global animator_duration_scale 0

# Keep running or return PID
if [ "$1" = "--background" ]; then
    echo "Emulator PID: $EMULATOR_PID"
    echo "To stop: kill $EMULATOR_PID"
else
    echo "Press Ctrl+C to stop emulator"
    trap "kill $EMULATOR_PID 2>/dev/null; exit" INT TERM
    wait $EMULATOR_PID
fi
