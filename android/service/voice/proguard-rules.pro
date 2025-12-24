# Voice Service Module ProGuard Rules

# Keep voice service and related classes
-keep class org.localforge.alicia.service.voice.VoiceService { *; }
-keep class org.localforge.alicia.service.voice.VoiceServiceReceiver { *; }

# Keep wake word detector native interfaces
-keep class ai.picovoice.porcupine.** { *; }
-keep interface ai.picovoice.porcupine.** { *; }

# LiveKit - keep all public APIs
-keep class io.livekit.android.** { *; }
-keep interface io.livekit.android.** { *; }

# MessagePack - keep serialization classes
-keep class org.msgpack.** { *; }
-keepclassmembers class * {
    @org.msgpack.** *;
}

# Keep audio manager and related audio classes
-keep class org.localforge.alicia.service.voice.AudioManager { *; }
-keep class android.media.** { *; }

# Hilt
-keep class dagger.hilt.** { *; }
-keep class javax.inject.** { *; }
-keepclassmembers class * {
    @javax.inject.* *;
    @dagger.* *;
}

# Coroutines
-keepnames class kotlinx.coroutines.internal.MainDispatcherFactory {}
-keepnames class kotlinx.coroutines.CoroutineExceptionHandler {}
-keepclassmembers class kotlinx.coroutines.** {
    volatile <fields>;
}

# Keep native methods
-keepclasseswithmembernames class * {
    native <methods>;
}

# Keep enums
-keepclassmembers enum * {
    public static **[] values();
    public static ** valueOf(java.lang.String);
}

# Remove logging in release builds
-assumenosideeffects class android.util.Log {
    public static *** d(...);
    public static *** v(...);
}
