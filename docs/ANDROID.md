# Android Client

The Alicia Android client provides a native mobile interface for voice-based AI assistant interactions, integrating LiveKit for real-time communication and Porcupine for wake word detection.

## Overview

The Android application is built using modern Android development practices with Jetpack Compose for the UI, following MVVM architecture with Clean Architecture principles. It enables users to have voice conversations with Alicia through their mobile devices with always-listening wake word detection capabilities.

## Architecture

### Project Structure

The Android project is organized into 12 Gradle modules following Clean Architecture:

```
android/
├── app/                          # Application entry point
├── core/
│   ├── common/                   # Shared utilities and constants
│   ├── data/                     # Data layer implementations
│   ├── domain/                   # Business logic and models
│   ├── network/                  # Network communication (LiveKit)
│   └── database/                 # Local data persistence
├── feature/
│   ├── assistant/                # Main assistant conversation UI
│   ├── conversations/            # Conversation history
│   └── settings/                 # App configuration
└── service/
    ├── voice/                    # Voice processing and wake word
    └── hotkey/                   # Accessibility service integration
```

### MVVM + Clean Architecture

- **Presentation Layer**: Jetpack Compose UI with ViewModels
- **Domain Layer**: Use cases and business logic
- **Data Layer**: Repositories and data sources
- Dependency injection via Hilt/Dagger
- Reactive state management with Kotlin Flow and LiveData

## Key Components

### LiveKit Integration

**Location**: `/android/core/network/src/main/java/org/localforge/alicia/core/network/LiveKitManager.kt`

The LiveKitManager handles real-time audio streaming:

- Establishes WebRTC connections to LiveKit rooms
- Manages audio track subscription and publication
- Handles reconnection logic and connection state
- Processes incoming/outgoing audio streams
- Integrates with MessagePack protocol for data messages

### Wake Word Detection

**Location**: `/android/service/voice/src/main/java/org/localforge/alicia/service/voice/`

Powered by Picovoice Porcupine engine:

- Always-listening background service for "Hey Alicia" wake word
- Low-power consumption optimized detection
- Configurable sensitivity and wake word models
- Triggers conversation activation when detected
- Runs as Android foreground service for reliability

### Voice Controller State Machine

**Location**: `/android/service/voice/src/main/java/org/localforge/alicia/service/voice/VoiceController.kt`

Manages the voice conversation lifecycle through states:

- **Idle**: Waiting for wake word or manual activation
- **Listening**: Recording user speech
- **Processing**: Sending audio to server via LiveKit
- **Speaking**: Playing back assistant response
- **Error**: Handling failures with user feedback

Coordinates between wake word detection, LiveKit streaming, and UI updates.

### Protocol Handling

**Location**: `/android/core/network/src/main/java/org/localforge/alicia/core/network/ProtocolHandler.kt`

Implements MessagePack-based binary protocol for efficient data exchange:

- Serializes/deserializes conversation messages
- Handles acknowledgement (ACK) messages for reliability
- Processes metadata and control messages
- Provides type-safe protocol message classes
- Manages stanza ID tracking for message ordering

### Main UI

**Location**: `/android/feature/assistant/src/main/java/org/localforge/alicia/feature/assistant/`

- **AssistantViewModel.kt**: Manages UI state and business logic
- **AssistantScreen.kt**: Main conversation interface with Compose
- Real-time message display with streaming support
- Voice activity indicator and waveform visualization
- Manual push-to-talk and tap-to-activate controls

## Building the Project

### Standard Build (with Gradle wrapper)

```bash
cd android
./gradlew assembleDebug
./gradlew installDebug
```

### NixOS Build (using gradlew-nix)

For systems using Nix package manager:

```bash
cd android
./gradlew-nix assembleDebug
./gradlew-nix installDebug
```

The `gradlew-nix` wrapper ensures Gradle runs in a Nix-compatible environment.

### Build Variants

- **Debug**: Development build with logging enabled
- **Release**: Production build with ProGuard/R8 optimization

## Configuration

### Server Connection

Configure the backend server URL in the app settings or via build configuration:

- Default: Points to `localhost:8080` for emulator testing
- Production: Configure via settings screen to point to your server

### Wake Word Model

The Porcupine wake word model is included in the app resources. Custom wake words can be:

- Generated via Picovoice Console
- Placed in `assets/wake_words/`
- Referenced in app configuration

### Permissions

Required Android permissions:

- `RECORD_AUDIO`: For voice input
- `INTERNET`: For server communication
- `FOREGROUND_SERVICE`: For wake word detection service
- `POST_NOTIFICATIONS`: For service notifications

## Key Source Files

- **Entry Point**: `/android/app/src/main/java/org/localforge/alicia/MainActivity.kt`
- **LiveKit Manager**: `/android/core/network/src/main/java/org/localforge/alicia/core/network/LiveKitManager.kt`
- **Voice Controller**: `/android/service/voice/src/main/java/org/localforge/alicia/service/voice/VoiceController.kt`
- **Assistant ViewModel**: `/android/feature/assistant/src/main/java/org/localforge/alicia/feature/assistant/AssistantViewModel.kt`
- **Protocol Handler**: `/android/core/network/src/main/java/org/localforge/alicia/core/network/ProtocolHandler.kt`
- **Navigation**: `/android/app/src/main/java/org/localforge/alicia/navigation/AliciaNavigation.kt`
- **Theme**: `/android/app/src/main/java/org/localforge/alicia/ui/theme/AliciaTheme.kt`

## See Also

- [AGENT.md](AGENT.md) - Server-side agent architecture
- [SERVER.md](SERVER.md) - Backend HTTP API
- [CLI.md](CLI.md) - Command-line interface
