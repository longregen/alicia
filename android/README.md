# Alicia Voice Assistant - Android App

Voice assistant with wake word detection, AI chat, and device tool integration.

## Features

- **Voice Activation**: Tap mic button or say "Alicia" to activate
- **AI Assistant**: Chat powered by LLM via self-hosted backend
- **Voice Notes**: Record, transcribe, and manage voice notes
- **Screen Context**: Read on-screen content to assist with questions
- **Device Tools**: Battery, location, clipboard, time/date via MCP
- **Onboarding**: Step-by-step setup for permissions and assistant role

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Kotlin |
| Min SDK | 34 (Android 14) |
| UI | Material Design 3, ViewBinding |
| Wake Word | Vosk (on-device) |
| VAD | Silero VAD v5 (ONNX Runtime) |
| STT | Whisper API (remote) |
| TTS | Kokoro API (remote) |
| AI | LLM Agent via REST API |
| Device Tools | WebSocket + MCP protocol |
| Storage | DataStore Preferences |
| Telemetry | OpenTelemetry → SigNoz |

## Building

### With Nix (Recommended)

```bash
# Build APKs
nix build .#android

# Output in result/
ls result/
# app-arm64-v8a-debug.apk  (78MB)
# app-armeabi-v7a-debug.apk (76MB)
# app-x86_64-debug.apk     (80MB)
# app-universal-debug.apk  (126MB)
```

### With Gradle

```bash
# Enter dev shell
nix develop .#android

# Or manually set up Android SDK and run:
./gradlew assembleDebug
```

### Install on Device

```bash
adb install result/app-arm64-v8a-debug.apk
```

## E2E Testing

Requires a connected device or emulator.

```bash
# Run all E2E tests
nix run .#android-e2e

# Run with fresh APK install
nix run .#android-e2e -- --install

# Run specific test
nix run .#android-e2e -- onboarding.yaml

# Interactive testing (in android dir)
nix develop .#android
maestro test e2e/onboarding.yaml
```

### Test Suites

| File | Description |
|------|-------------|
| `e2e/onboarding.yaml` | Full onboarding flow with permissions |
| `e2e/assistant_setup.yaml` | Configure as default Android assistant |
| `e2e/voice_interaction.yaml` | Voice input → server → response |
| `e2e/conversations.yaml` | Verify conversation storage |

## Project Structure

```
app/src/main/
├── java/com/alicia/assistant/
│   ├── MainActivity.kt                 # Main screen, voice activation
│   ├── OnboardingActivity.kt           # First-run setup flow
│   ├── OnboardingPagerAdapter.kt       # Onboarding pages
│   ├── SettingsActivity.kt             # App settings
│   ├── VoiceNotesActivity.kt           # Voice notes list
│   ├── ChatActivity.kt                 # Conversation view
│   ├── ConversationListActivity.kt     # All conversations
│   ├── AliciaApplication.kt            # App initialization
│   ├── model/
│   │   ├── Models.kt                   # Data classes
│   │   └── RecognitionResult.kt        # STT result type
│   ├── service/
│   │   ├── AliciaApiClient.kt          # REST API client
│   │   ├── AliciaInteractionSession.kt # Voice session (wake word)
│   │   ├── VoiceAssistantService.kt    # Background wake word detection
│   │   ├── VoiceRecognitionManager.kt  # Whisper STT integration
│   │   ├── SileroVadDetector.kt        # Voice activity detection
│   │   ├── TtsManager.kt               # Text-to-speech playback
│   │   ├── ScreenContextManager.kt     # Read on-screen content
│   │   └── NoteSaver.kt                # Voice note transcription
│   ├── ws/
│   │   ├── AssistantWebSocket.kt       # WebSocket for MCP tools
│   │   ├── ToolRegistry.kt             # Device tool registration
│   │   └── ToolExecutor.kt             # Tool execution interface
│   ├── tools/                          # Device tool implementations
│   │   ├── GetBatteryExecutor.kt
│   │   ├── GetLocationExecutor.kt
│   │   ├── GetClipboardExecutor.kt
│   │   ├── GetTimeExecutor.kt
│   │   ├── GetDateExecutor.kt
│   │   └── ReadScreenExecutor.kt
│   ├── storage/
│   │   ├── PreferencesManager.kt       # DataStore preferences
│   │   └── NoteRepository.kt           # Voice notes storage
│   └── telemetry/
│       └── AliciaTelemetry.kt          # OpenTelemetry setup
├── res/
│   ├── layout/                         # XML layouts
│   ├── values/                         # Strings, colors, themes
│   └── drawable/                       # Icons and graphics
├── assets/
│   ├── silero_vad.onnx                 # VAD model (bundled)
│   └── vosk-models/small-en-us/        # Wake word model (bundled)
└── AndroidManifest.xml
```

## Permissions

| Permission | Purpose |
|------------|---------|
| `RECORD_AUDIO` | Voice input, wake word detection |
| `POST_NOTIFICATIONS` | Foreground service notification |
| `FOREGROUND_SERVICE` | Background wake word listening |
| `FOREGROUND_SERVICE_MICROPHONE` | Mic access in background |
| `BLUETOOTH_CONNECT` | Bluetooth headset support |
| `ACCESS_COARSE_LOCATION` | Location tool for agent |
| `WAKE_LOCK` | Keep CPU awake during processing |
| `RECEIVE_BOOT_COMPLETED` | Auto-start at boot |
| `INTERNET` | API communication |

## Onboarding Flow

First launch guides users through:

1. **Welcome** - Introduction
2. **Microphone** - Required for voice input
3. **Notifications** - Required for background service
4. **Bluetooth** - Optional, for headset support
5. **Location** - Optional, for location-aware responses
6. **Assistant** - Optional, set as default Android assistant
7. **Complete** - Ready to use

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Android App                          │
├─────────────────────────────────────────────────────────────┤
│  MainActivity          │  VoiceAssistantService             │
│  - Tap to speak        │  - Wake word detection (Vosk)      │
│  - Show response       │  - Triggers AliciaInteractionSession│
├────────────────────────┼────────────────────────────────────┤
│  AliciaApiClient (REST)│  AssistantWebSocket (WebSocket)    │
│  - Create conversation │  - Register device tools           │
│  - Send message (sync) │  - Handle tool requests            │
│  - Get preferences     │  - Return tool results             │
└────────────┬───────────┴──────────────┬─────────────────────┘
             │                          │
             ▼                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   alicia.hjkl.lol                           │
├─────────────────────────────────────────────────────────────┤
│  API Server          │  Agent                               │
│  - Conversations     │  - LLM responses                     │
│  - Messages          │  - MCP tool calls                    │
│  - Preferences       │  - Memory retrieval                  │
└──────────────────────┴──────────────────────────────────────┘
```

## Device Tools (MCP)

The agent can request device information via WebSocket:

| Tool | Returns |
|------|---------|
| `get_time` | Current time |
| `get_date` | Current date |
| `get_battery` | Battery level, charging status |
| `get_location` | GPS coordinates |
| `get_clipboard` | Clipboard text |
| `read_screen` | On-screen text (accessibility + OCR) |

## Privacy

- Wake word detection runs entirely on-device (Vosk)
- Voice activity detection runs on-device (Silero VAD)
- Voice commands are sent to self-hosted backend for:
  - Speech-to-text (Whisper)
  - AI response generation (LLM)
  - Text-to-speech (Kokoro)
- Voice notes stored locally only
- No third-party analytics or tracking
- All telemetry goes to self-hosted SigNoz instance
