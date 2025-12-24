# Consumer ProGuard Rules for Voice Service Module
# These rules will be automatically applied to apps that depend on this module

# Keep VoiceService and public APIs
-keep public class org.localforge.alicia.service.voice.VoiceService { *; }
-keep public class org.localforge.alicia.service.voice.VoiceController { *; }
-keep public class org.localforge.alicia.service.voice.VoiceState { *; }
-keep public class org.localforge.alicia.service.voice.WakeWordDetector { *; }
-keep public class org.localforge.alicia.service.voice.WakeWordDetector$WakeWord { *; }
-keep public class org.localforge.alicia.service.voice.AudioManager { *; }
-keep public class org.localforge.alicia.service.voice.PowerAwareWakeWordDetector { *; }

# Keep Porcupine native methods
-keep class ai.picovoice.porcupine.** { *; }

# Keep LiveKit
-keep class io.livekit.android.** { *; }
