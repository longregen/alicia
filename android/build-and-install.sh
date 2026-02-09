#!/usr/bin/env bash
set -e

# Re-exec inside nix android dev shell if not already there
if [ -z "$ANDROID_HOME" ]; then
    exec nix develop "$(git rev-parse --show-toplevel)#android" --command "$0" "$@"
fi

# Regenerate local.properties with current SDK path
echo "sdk.dir=$ANDROID_HOME" > local.properties

./gradlew assembleDebug \
    -Pandroid.aapt2FromMavenOverride="$ANDROID_HOME/build-tools/35.0.0/aapt2"
adb install -r app/build/outputs/apk/debug/app-arm64-v8a-debug.apk
adb shell pm grant com.alicia.assistant android.permission.RECORD_AUDIO
adb shell settings put secure assistant com.alicia.assistant/com.alicia.assistant.MainActivity
