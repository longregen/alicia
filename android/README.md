# Alicia Voice Assistant - Native Android App

A privacy-focused voice assistant for Android that can be triggered by wake word or button to launch apps, play music, ask questions, and save voice notes.

## Features

- 🎤 **Voice Activation**: Tap button or say "Alicia" to activate
- 🚀 **App Launcher**: Open apps by voice (Spotify, Instagram, etc.)
- 🎵 **Music Control**: Play music with voice commands
- 📝 **Voice Notes**: Save reminders and notes for later
- 🤖 **Assistant Queries**: Ask about time, date, and more
- 🔒 **Privacy First**: All processing on-device, no cloud required
- 🎨 **Material Design 3**: Modern, beautiful UI
- 🔋 **Battery Optimized**: Efficient wake word detection

## Tech Stack

- **Language**: Kotlin
- **UI**: Material Design 3 with XML layouts
- **Speech Recognition**: Android SpeechRecognizer API
- **Text-to-Speech**: Android TTS
- **Storage**: DataStore Preferences
- **Min SDK**: 24 (Android 7.0)
- **Target SDK**: 34 (Android 14)

## Project Structure

```
app/
├── src/main/
│   ├── java/com/alicia/assistant/
│   │   ├── MainActivity.kt              # Main screen with voice activation
│   │   ├── SettingsActivity.kt          # Settings configuration
│   │   ├── VoiceNotesActivity.kt        # Voice notes management
│   │   ├── AliciaApplication.kt         # Application class
│   │   ├── model/
│   │   │   └── Models.kt                # Data models
│   │   ├── service/
│   │   │   ├── VoiceRecognitionManager.kt  # Speech-to-text
│   │   │   └── VoiceAssistantService.kt    # Background service
│   │   ├── ws/
│   │   │   ├── AssistantWebSocket.kt    # WebSocket for MCP tools
│   │   │   ├── ToolRegistry.kt          # Tool registration
│   │   │   └── ToolExecutor.kt          # Tool execution interface
│   │   ├── tools/                       # Device tool executors
│   │   └── storage/
│   │       └── PreferencesManager.kt    # Data persistence
│   ├── res/
│   │   ├── layout/                      # XML layouts
│   │   ├── values/                      # Strings, colors, themes
│   │   └── drawable/                    # Icons and graphics
│   └── AndroidManifest.xml
└── build.gradle.kts
```

## Building the App

### Prerequisites

1. **Android Studio** (Hedgehog 2023.1.1 or later)
   - Download from: https://developer.android.com/studio

2. **JDK 17** (included with Android Studio)

3. **Android SDK 34** (will be installed by Android Studio)

### Build Steps

1. **Open Project**
   ```bash
   # Extract the ZIP file
   unzip alicia-native.zip
   cd alicia-native
   
   # Open in Android Studio
   # File > Open > Select alicia-native folder
   ```

2. **Sync Gradle**
   - Android Studio will automatically sync Gradle
   - Wait for dependencies to download (~5-10 minutes first time)

3. **Build APK**
   - **Option A**: Via Android Studio
     - Build > Build Bundle(s) / APK(s) > Build APK(s)
     - APK will be in: `app/build/outputs/apk/debug/app-debug.apk`
   
   - **Option B**: Via Command Line
     ```bash
     # Debug APK (for testing)
     ./gradlew assembleDebug
     
     # Release APK (optimized, smaller size)
     ./gradlew assembleRelease
     ```

4. **Install on Device**
   - Connect Android device via USB (enable USB debugging)
   - Run > Run 'app' in Android Studio
   - Or use ADB:
     ```bash
     adb install app/build/outputs/apk/debug/app-debug.apk
     ```

### Release Build (Production)

For a production release APK:

1. **Generate Signing Key**
   ```bash
   keytool -genkey -v -keystore alicia-release.keystore \
     -alias alicia -keyalg RSA -keysize 2048 -validity 10000
   ```

2. **Configure Signing** in `app/build.gradle.kts`:
   ```kotlin
   android {
       signingConfigs {
           create("release") {
               storeFile = file("../alicia-release.keystore")
               storePassword = "your_password"
               keyAlias = "alicia"
               keyPassword = "your_password"
           }
       }
       buildTypes {
           release {
               signingConfig = signingConfigs.getByName("release")
               // ... existing config
           }
       }
   }
   ```

3. **Build Release APK**
   ```bash
   ./gradlew assembleRelease
   ```

## Permissions

The app requires the following permissions:

- **RECORD_AUDIO**: Voice commands and wake word detection
- **INTERNET**: (Optional) For future online features
- **FOREGROUND_SERVICE**: Background wake word listening
- **POST_NOTIFICATIONS**: Service notifications
- **WAKE_LOCK**: Keep CPU awake for wake word detection
- **RECEIVE_BOOT_COMPLETED**: Auto-start wake word service

## Usage

### Voice Commands

- **Launch Apps**: "Open Spotify", "Launch Instagram"
- **Play Music**: "Play some music", "Play [song name]"
- **Voice Notes**: "Leave a voice note to buy groceries"
- **Queries**: "What time is it?", "What's the date?"
- **Timers**: "Set a timer for 5 minutes"

### Settings

- Enable/disable wake word detection
- Adjust wake word sensitivity
- Toggle voice feedback (TTS)
- Toggle haptic feedback
- Clear command history
- Clear voice notes

## Troubleshooting

### Build Errors

**"SDK location not found"**
- Create `local.properties` in project root:
  ```
  sdk.dir=/path/to/Android/Sdk
  ```

**"Gradle sync failed"**
- File > Invalidate Caches / Restart
- Delete `.gradle` folder and sync again

**"Kotlin version mismatch"**
- Update Kotlin plugin in Android Studio
- Sync Gradle again

### Runtime Issues

**"Microphone permission denied"**
- Go to Settings > Apps > Alicia > Permissions
- Enable Microphone permission

**"Speech recognition not working"**
- Ensure device has Google app installed
- Check internet connection (required for Android SpeechRecognizer)
- Try offline STT library (Vosk) for fully offline operation

**"Wake word not detecting"**
- Enable wake word in Settings
- Grant microphone permission
- Ensure app is not battery optimized (Settings > Battery)

## Future Enhancements

- [ ] Offline STT with Vosk library
- [ ] Custom wake word training
- [ ] More skills (weather, calendar, etc.)
- [ ] Conversation history
- [ ] Multi-language support
- [ ] Widget for quick activation

## Privacy

- Wake-word detection runs on-device using Vosk
- Voice commands and chat messages are sent to a self-hosted backend at alicia.hjkl.lol for AI processing (Whisper STT, LLM agent, Kokoro TTS)
- Voice notes are stored locally only
- No third-party analytics or tracking

## License

This project is provided as-is for personal use.

## Support

For issues or questions, please refer to the source code comments or Android documentation.

## Credits

Built with:
- Android SDK
- Material Design 3
- Kotlin Coroutines
- DataStore Preferences
