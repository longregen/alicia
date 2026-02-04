Android voice assistant app that combines on-device wake-word detection with cloud-based speech processing and AI responses. Written in Kotlin using Material Design 3.

ALWAYS build and install with ./build-and-install.sh

The app uses a self-hosted API backend at `https://alicia.hjkl.lol` for AI services:
- **Whisper** — speech-to-text transcription (`service/VoiceRecognitionManager.kt`)
- **Agent** — AI assistant via REST API (`service/AliciaApiClient.kt`)
- **Kokoro** — text-to-speech generation (`service/TtsManager.kt`)

HTTP client with retry/auth interceptors: `service/ApiClient.kt`

Wake-word detection runs entirely on-device using the **Vosk** library: `service/VoiceAssistantService.kt`

Voice activity detection (endpoint detection) uses **Silero VAD v5** via ONNX Runtime: `service/SileroVadDetector.kt`. The ~2MB model is bundled at `assets/silero_vad.onnx`.

## Architecture

| Feature | Transport | Pattern |
|---------|-----------|---------|
| Send user message | REST POST | `/api/v1/conversations/{id}/messages?sync=true` |
| Receive tool requests | WebSocket | `assistantMode: true`, handle `TYPE_TOOL_USE_REQUEST` |
| Conversations CRUD | REST | Same as web |
| Notes CRUD | REST | Same as web |
| Preferences | REST | GET/PATCH |

The Android app uses REST for all messaging (like the web interface) while maintaining a WebSocket connection exclusively for MCP device tools (battery, location, screen, clipboard).

## Features & Implementing Files

| Feature | Files |
|---------|-------|
| Voice commands (tap or wake word) | `MainActivity.kt`, `service/VoiceAssistantService.kt`, `service/AliciaInteractionSession.kt`, `service/SileroVadDetector.kt` |
| Voice notes (record, transcribe, play, edit, share, delete) | `VoiceNotesActivity.kt`, `NoteDetailActivity.kt`, `service/NoteSaver.kt`, `storage/NoteRepository.kt` |
| Screen context (reads on-screen text via accessibility + OCR) | `service/ScreenContextManager.kt`, `service/AliciaInteractionSession.kt` |
| Quick Settings tile | `service/VoiceAssistantTileService.kt` |
| Model management (download Vosk models) | `ModelManagerActivity.kt`, `service/ModelDownloadService.kt` |
| Settings (wake word, feedback, TTS speed) | `SettingsActivity.kt`, `storage/PreferencesManager.kt` |
| Boot auto-start | `receiver/BootReceiver.kt` |
| MCP device tools | `ws/AssistantWebSocket.kt`, `ws/ToolRegistry.kt`, `ws/ToolExecutor.kt`, `tools/*.kt` |

All source files live under `app/src/main/java/com/alicia/assistant/`.

## Permissions

| Permission | Purpose | Used by |
|------------|---------|---------|
| `RECORD_AUDIO` | Microphone input | `VoiceRecognitionManager`, `VoiceAssistantService` |
| `INTERNET` | Cloud API calls (Whisper, Agent, TTS) | `ApiClient`, `VoiceRecognitionManager`, `AliciaApiClient`, `TtsManager` |
| `FOREGROUND_SERVICE` + `_MICROPHONE` | Always-on wake word detection | `VoiceAssistantService` |
| `FOREGROUND_SERVICE_DATA_SYNC` | Background model downloads | `ModelDownloadService` |
| `POST_NOTIFICATIONS` | Service notification | `VoiceAssistantService`, `ModelDownloadService` |
| `WAKE_LOCK` | Keep device awake during processing | `VoiceAssistantService` |
| `RECEIVE_BOOT_COMPLETED` | Auto-start at boot | `BootReceiver` |
| `BLUETOOTH_CONNECT` | Headset button activation | `AliciaInteractionSession` |

Declared in `app/src/main/AndroidManifest.xml`.

## App Behavior

1. **At boot** — `BootReceiver` starts `VoiceAssistantService` as a foreground service for always-on wake word detection (if enabled in settings).
2. **Wake word detected** — Vosk matches the configured word (default: "alicia") in `VoiceAssistantService.checkForWakeWord()`, then calls `AliciaInteractionService.triggerAssistSession()`.
3. **During a session** — `AliciaInteractionSession` records audio using VAD-based endpoint detection (auto-stops after 1.5s of silence, 30s max). `VoiceRecognitionManager` sends captured audio to Whisper. The query goes to `AliciaApiClient.sendMessageSync()` (with optional screen context from `ScreenContextManager`). The response is spoken via `TtsManager`.
4. **TTS playback** — `TtsManager` pauses wake-word detection (`VoiceAssistantService.pauseDetection()`) during audio playback to prevent self-triggering, and resumes it on completion.
5. **MCP tools** — `AssistantWebSocket` maintains a WebSocket connection with `assistantMode: true` to receive `TYPE_TOOL_USE_REQUEST` messages for client-side tools. The agent can request device info (battery, location, screen content, clipboard) which executes locally and returns results via WebSocket.
6. **Voice notes** — `NoteSaver` transcribes with word-level timestamps via Whisper verbose mode, `NoteRepository` persists as JSON + audio. `VoiceNotesActivity` provides playback with synchronized word highlighting.
7. **Error recovery** — `VoiceAssistantService` retries up to 5 times with exponential backoff on recognition failure.

## Data Storage

| Data | Location | Managed by |
|------|----------|------------|
| Notes metadata | `filesDir/voice_notes_meta/*.json` | `NoteRepository` |
| Note audio | `filesDir/voice_notes/*.m4a` | `NoteSaver`, `NoteRepository` |
| Speech models | `filesDir/vosk-models/{modelId}/` | `ModelDownloadService`, `AliciaApplication` |
| Preferences | DataStore `alicia_prefs` | `PreferencesManager` |
| TTS cache | `cacheDir/tts_*.mp3` (auto-deleted) | `TtsManager` |
| VAD model | `assets/silero_vad.onnx` (bundled) | `SileroVadDetector` |

All data is local to the device. Network traffic is limited to API calls to the self-hosted backend.

## Key Classes

- `AliciaApplication.kt` — App init: notification channel, bundled model extraction, WebSocket initialization
- `MainViewModel.kt` — Lifecycle-aware state holder for settings and notes
- `model/Models.kt` — Data classes: `VoiceNote`, `TimestampedWord`, `VoskModelInfo`, `AppSettings`
- `model/RecognitionResult.kt` — Sealed class for recognition outcomes (Success/Error)
- `service/AliciaApiClient.kt` — REST API client for conversations, messages, notes, and preferences
- `service/AliciaInteractionSession.kt` — Voice interaction session that sends messages via REST API
- `ws/AssistantWebSocket.kt` — WebSocket client for MCP tool handling only (not for messaging)
- `service/SileroVadDetector.kt` — ONNX Runtime wrapper for Silero VAD v5; detects speech/silence boundaries at 16kHz (512-sample frames, ~31 fps). Speech threshold: 0.5, silence threshold: 0.3 (hysteresis)

## Available Vosk Models

Defined in `model/Models.kt` as `VoskModelInfo` enum:
- English Small (40MB, bundled) — default
- English Medium (128MB)
- Spanish, French, German, Japanese, Chinese (all small)

Downloaded from Alphacephei.com via `ModelDownloadService`.
