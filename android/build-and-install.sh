#!/usr/bin/env bash
set -e
./gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-arm64-v8a-debug.apk
adb shell pm grant com.alicia.assistant android.permission.RECORD_AUDIO
